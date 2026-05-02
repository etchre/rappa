package music

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

func (p *Player) Skip(ctx context.Context, guildID snowflake.ID) (SkipResult, error) {
	p.mu.Lock()
	playback := p.playback(guildID)
	playback.looping = false
	if len(playback.queue) == 0 {
		wasPlaying := playback.playing
		playback.playing = false
		playback.current = nil
		playback.paused = false
		p.mu.Unlock()

		if wasPlaying {
			p.notifyPlaybackIdle(ctx, guildID)
			if err := p.clearLavalinkTrack(ctx, guildID); err != nil {
				return SkipResult{}, err
			}
		}

		return SkipResult{Stopped: wasPlaying}, nil
	}

	next := playback.queue[0]
	playback.queue = playback.queue[1:]
	playback.current = &next
	playback.playing = true
	playback.paused = false
	needsAsyncLoad := next.Preparation == trackPreparationPremiumLikely && next.PreparedIdentifier != ""
	p.mu.Unlock()

	// Premium tracks require an extra Lavalink load that can exceed Discord's
	// interaction response deadline. Fire the load in the background so we can
	// return the skip result immediately.
	if needsAsyncLoad {
		go func() {
			if err := p.playQueuedTrack(ctx, guildID, next); err != nil {
				fmt.Fprintf(os.Stderr, "[skip] async premium play failed: %v\n", err)
			}
			go p.prepareQueueAhead(guildID)
		}()
	} else {
		if err := p.playQueuedTrack(ctx, guildID, next); err != nil {
			return SkipResult{}, err
		}
		go p.prepareQueueAhead(guildID)
	}

	nextTrack := next.Track
	return SkipResult{Next: &nextTrack}, nil
}

func (p *Player) Stop(ctx context.Context, guildID snowflake.ID) error {
	p.mu.Lock()
	playback := p.playback(guildID)
	wasPlaying := playback.playing
	playback.playing = false
	playback.current = nil
	playback.queue = nil
	playback.paused = false
	playback.looping = false
	p.mu.Unlock()

	if !wasPlaying {
		return nil
	}

	p.notifyPlaybackIdle(ctx, guildID)
	return p.clearLavalinkTrack(ctx, guildID)
}

func (p *Player) OnTrackEnd(player disgolink.Player, event lavalink.TrackEndEvent) {
	if event.Reason == lavalink.TrackEndReasonLoadFailed {
		p.handlePlaybackFailure(context.Background(), player, event.Track)
		return
	}

	if !event.Reason.MayStartNext() {
		return
	}

	// Guard against tracks that "finish" almost instantly (e.g. cold-start
	// failures where Lavalink sends Finished instead of LoadFailed). Also
	// catch the case where TrackException arrived first and set the flag.
	p.mu.Lock()
	playback := p.playback(player.GuildID())
	hadException := playback.exceptionTrack == event.Track.Encoded
	playback.exceptionTrack = ""
	startTime := playback.playStartTime
	elapsed := time.Since(startTime)
	p.mu.Unlock()

	if hadException || (!startTime.IsZero() && elapsed < 3*time.Second) {
		fmt.Printf("[failure] track ended suspiciously fast (%.1fs) or after exception, treating as failure: %s\n",
			elapsed.Seconds(), trackTitle(event.Track))
		p.handlePlaybackFailure(context.Background(), player, event.Track)
		return
	}

	if err := p.playNext(context.Background(), player.GuildID()); err != nil {
		fmt.Fprintf(os.Stderr, "play next failed: %v\n", err)
	}
}

func (p *Player) OnTrackException(player disgolink.Player, event lavalink.TrackExceptionEvent) {
	debugLavalinkException("playback", playerNodeName(player), trackTitle(event.Track), event.Exception)

	p.mu.Lock()
	p.playback(player.GuildID()).exceptionTrack = event.Track.Encoded
	p.mu.Unlock()

	p.handlePlaybackFailure(context.Background(), player, event.Track)
}

func (p *Player) handlePlaybackFailure(ctx context.Context, player disgolink.Player, track lavalink.Track) {
	guildID := player.GuildID()

	p.mu.Lock()
	playback := p.playback(guildID)
	if playback.premiumFallbackBusy {
		p.mu.Unlock()
		return
	}
	if playback.current == nil || playback.current.Track.Encoded != track.Encoded {
		p.mu.Unlock()
		fmt.Printf("[failure] ignored stale playback failure for %s\n", trackTitle(track))
		return
	}
	if playback.current.ResolvedRetryAttempted {
		p.mu.Unlock()
		fmt.Printf("[failure] already retried %s, falling back to search recovery\n", trackTitle(track))
		if p.recoverFailedTrackBySearch(ctx, guildID, track) {
			return
		}
		if err := p.advanceAfterFailedTrack(ctx, guildID, track); err != nil {
			fmt.Fprintf(os.Stderr, "[failure] advance after failed track: %v\n", err)
		}
		return
	}

	usedPremium := playback.current.UsedPremiumRoute
	playback.current.ResolvedRetryAttempted = true
	playback.premiumFallbackBusy = true
	p.mu.Unlock()

	route := "lavalink"
	if usedPremium {
		route = "premium"
	}
	fmt.Printf("[failure] playback failed for %s (route=%s), retrying with resolver\n", trackTitle(track), route)
	go p.retryWithResolver(ctx, guildID, track, usedPremium)
}

func (p *Player) retryWithResolver(ctx context.Context, guildID snowflake.ID, failedTrack lavalink.Track, usedPremium bool) {
	defer p.setPremiumFallbackBusy(guildID, false)

	fmt.Printf("[retry] resolving alternate video ID for %s\n", trackTitle(failedTrack))
	identifier := resolvedTrackIdentifier(ctx, failedTrack)
	if identifier == "" {
		fmt.Fprintf(os.Stderr, "[retry] resolver could not determine track URL for %s\n", trackTitle(failedTrack))
		p.failWithSearchRecovery(ctx, guildID, failedTrack)
		return
	}

	var loadIdentifier string
	if usedPremium {
		loadIdentifier = premiumPlayableIdentifier(identifier)
		fmt.Printf("[retry] resolved %s -> %s, retrying via premium plugin\n", trackTitle(failedTrack), identifier)
	} else {
		loadIdentifier = identifier
		fmt.Printf("[retry] resolved %s -> %s, retrying via lavalink\n", trackTitle(failedTrack), identifier)
	}

	node := p.node()
	if node == nil {
		fmt.Fprintf(os.Stderr, "[retry] no lavalink node available\n")
		if err := p.advanceAfterFailedTrack(ctx, guildID, failedTrack); err != nil {
			fmt.Fprintf(os.Stderr, "[retry] advance after failed track: %v\n", err)
		}
		return
	}

	loaded, err := loadPlayableTracks(ctx, node, loadIdentifier)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[retry] load failed for %s: %v\n", trackTitle(failedTrack), err)
		p.failWithSearchRecovery(ctx, guildID, failedTrack)
		return
	}
	if len(loaded.Tracks) == 0 {
		fmt.Fprintf(os.Stderr, "[retry] load returned no tracks for %s\n", trackTitle(failedTrack))
		p.failWithSearchRecovery(ctx, guildID, failedTrack)
		return
	}

	retryTrack := loaded.Tracks[0]
	p.mu.Lock()
	playback := p.playback(guildID)
	shouldPlay := false
	if playback.current != nil && playback.current.Track.Encoded == failedTrack.Encoded {
		playback.current.Track = retryTrack
		playback.playing = true
		playback.paused = false
		shouldPlay = true
	}
	p.mu.Unlock()

	if shouldPlay {
		if err := p.playTrack(ctx, guildID, retryTrack); err != nil {
			fmt.Fprintf(os.Stderr, "[retry] play failed for %s: %v\n", trackTitle(failedTrack), err)
			p.failWithSearchRecovery(ctx, guildID, failedTrack)
		}
	}
}

func (p *Player) setPremiumFallbackBusy(guildID snowflake.ID, busy bool) {
	p.mu.Lock()
	p.playback(guildID).premiumFallbackBusy = busy
	p.mu.Unlock()
}

func (p *Player) playNext(ctx context.Context, guildID snowflake.ID) error {
	p.mu.Lock()
	playback := p.playback(guildID)
	if playback.premiumFallbackBusy {
		p.mu.Unlock()
		return nil
	}
	if playback.looping && playback.current != nil {
		current := *playback.current
		p.mu.Unlock()
		return p.playTrack(ctx, guildID, current.Track)
	}

	if len(playback.queue) == 0 {
		playback.playing = false
		playback.current = nil
		playback.paused = false
		playback.looping = false
		p.mu.Unlock()
		p.notifyPlaybackIdle(ctx, guildID)
		return nil
	}

	next := playback.queue[0]
	playback.queue = playback.queue[1:]
	playback.current = &next
	playback.paused = false
	p.mu.Unlock()

	if err := p.playQueuedTrack(ctx, guildID, next); err != nil {
		return err
	}
	p.notifyAutoTrackStart(ctx, guildID)
	go p.prepareQueueAhead(guildID)
	return nil
}

func (p *Player) playQueuedTrack(ctx context.Context, guildID snowflake.ID, item queuedTrack) error {
	switch item.Preparation {
	case trackPreparationResolvedLavalink:
		if item.PreparedTrack != nil {
			fmt.Printf("[queue] using prepared lavalink track for %s\n", trackTitle(item.Track))
			preparedTrack := *item.PreparedTrack
			p.replaceCurrentTrack(guildID, item.ID, preparedTrack)
			return p.playTrack(ctx, guildID, preparedTrack)
		}
	case trackPreparationPremiumLikely:
		if item.PreparedIdentifier != "" {
			fmt.Printf("[queue] using prepared premium route for %s\n", trackTitle(item.Track))
			return p.playPreparedPremiumTrack(ctx, guildID, item)
		}
	}

	return p.playTrack(ctx, guildID, item.Track)
}

func (p *Player) playPreparedPremiumTrack(ctx context.Context, guildID snowflake.ID, item queuedTrack) error {
	loaded, err := loadPlayableTracks(ctx, p.node(), premiumPlayableIdentifier(item.PreparedIdentifier))
	if err != nil {
		return fmt.Errorf("load prepared premium track: %w", err)
	}
	if len(loaded.Tracks) == 0 {
		return fmt.Errorf("prepared premium returned no tracks")
	}

	premiumTrack := loaded.Tracks[0]
	premiumTrack.Info = item.Track.Info

	p.mu.Lock()
	playback := p.playback(guildID)
	if playback.current != nil && playback.current.ID == item.ID {
		playback.current.UsedPremiumRoute = true
	}
	p.mu.Unlock()

	p.replaceCurrentTrack(guildID, item.ID, premiumTrack)
	return p.playTrack(ctx, guildID, premiumTrack)
}

func (p *Player) replaceCurrentTrack(guildID snowflake.ID, itemID uint64, track lavalink.Track) {
	p.mu.Lock()
	defer p.mu.Unlock()

	playback := p.playback(guildID)
	if playback.current == nil || playback.current.ID != itemID {
		return
	}

	playback.current.Track = track
	playback.current.ResolvedRetryAttempted = true
}

func (p *Player) playTrack(ctx context.Context, guildID snowflake.ID, track lavalink.Track) error {
	player := p.lavalinkPlayer(guildID)
	if err := player.Update(ctx, lavalink.WithTrack(track), lavalink.WithPaused(false)); err != nil {
		debugTrackPlayError(player, track, err)
		return fmt.Errorf("play track: %w", err)
	}

	p.mu.Lock()
	p.playback(guildID).playStartTime = time.Now()
	p.mu.Unlock()

	p.notifyPlaybackActive(ctx, guildID)
	fmt.Printf("Now playing on Lavalink node %q: %s\n", playerNodeName(player), trackTitle(track))
	return nil
}

func (p *Player) failWithSearchRecovery(ctx context.Context, guildID snowflake.ID, failedTrack lavalink.Track) {
	if p.recoverFailedTrackBySearch(ctx, guildID, failedTrack) {
		return
	}
	fmt.Printf("[recovery] search recovery failed for %s, skipping track\n", trackTitle(failedTrack))
	if err := p.advanceAfterFailedTrack(ctx, guildID, failedTrack); err != nil {
		fmt.Fprintf(os.Stderr, "[recovery] advance after failed track: %v\n", err)
	}
}

func (p *Player) recoverFailedTrackBySearch(ctx context.Context, guildID snowflake.ID, failedTrack lavalink.Track) bool {
	p.mu.Lock()
	playback := p.playback(guildID)
	if playback.current == nil || playback.current.Track.Encoded != failedTrack.Encoded {
		p.mu.Unlock()
		return false
	}
	if playback.current.RecoveryAttempted || playback.current.RecoveryQuery == "" {
		p.mu.Unlock()
		return false
	}

	query := playback.current.RecoveryQuery
	playback.current.RecoveryAttempted = true
	playback.premiumFallbackBusy = true
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.playback(guildID).premiumFallbackBusy = false
		p.mu.Unlock()
	}()

	fmt.Printf("[recovery] searching for %s query=%q\n", trackTitle(failedTrack), query)
	loaded, err := loadPlayableTracks(ctx, p.node(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[recovery] search failed: %v\n", err)
		return false
	}
	if len(loaded.Tracks) == 0 {
		fmt.Fprintf(os.Stderr, "[recovery] search returned no tracks query=%q\n", query)
		return false
	}

	recoveredTrack := loaded.Tracks[0]
	if recoveredTrack.Encoded == failedTrack.Encoded {
		fmt.Fprintf(os.Stderr, "[recovery] search returned the same failed track query=%q\n", query)
		return false
	}

	p.mu.Lock()
	playback = p.playback(guildID)
	shouldPlay := false
	if playback.current != nil && playback.current.Track.Encoded == failedTrack.Encoded {
		playback.current.Track = recoveredTrack
		playback.playing = true
		playback.paused = false
		shouldPlay = true
	}
	p.mu.Unlock()
	if !shouldPlay {
		return false
	}

	if err := p.playTrack(ctx, guildID, recoveredTrack); err != nil {
		fmt.Fprintf(os.Stderr, "[recovery] play search result failed: %v\n", err)
		return false
	}

	return true
}

func (p *Player) advanceAfterFailedTrack(ctx context.Context, guildID snowflake.ID, failedTrack lavalink.Track) error {
	p.mu.Lock()
	playback := p.playback(guildID)
	if playback.current == nil || playback.current.Track.Encoded != failedTrack.Encoded {
		p.mu.Unlock()
		return nil
	}

	playback.looping = false
	if len(playback.queue) == 0 {
		playback.current = nil
		playback.playing = false
		playback.paused = false
		p.mu.Unlock()
		p.notifyTrackFailure(ctx, guildID, failedTrack)
		p.notifyPlaybackIdle(ctx, guildID)
		return p.clearLavalinkTrack(ctx, guildID)
	}

	next := playback.queue[0]
	playback.queue = playback.queue[1:]
	playback.current = &next
	playback.playing = true
	playback.paused = false
	p.mu.Unlock()

	p.notifyTrackFailure(ctx, guildID, failedTrack)
	if err := p.playTrack(ctx, guildID, next.Track); err != nil {
		return err
	}
	p.notifyAutoTrackStart(ctx, guildID)
	return nil
}

func (p *Player) notifyAutoTrackStart(ctx context.Context, guildID snowflake.ID) {
	if p.autoTrackStartNotify == nil {
		return
	}

	p.autoTrackStartNotify(ctx, guildID)
}

func (p *Player) notifyTrackFailure(ctx context.Context, guildID snowflake.ID, track lavalink.Track) {
	if p.trackFailureNotify == nil {
		return
	}

	p.trackFailureNotify(ctx, guildID, track)
}

func (p *Player) notifyPlaybackActive(ctx context.Context, guildID snowflake.ID) {
	if p.playbackActiveNotify == nil {
		return
	}

	p.playbackActiveNotify(ctx, guildID)
}

func (p *Player) notifyPlaybackIdle(ctx context.Context, guildID snowflake.ID) {
	if p.playbackIdleNotify == nil {
		return
	}

	p.playbackIdleNotify(ctx, guildID)
}

func (p *Player) clearLavalinkTrack(ctx context.Context, guildID snowflake.ID) error {
	player := p.lavalinkPlayer(guildID)
	if err := player.Update(ctx, lavalink.WithNullTrack()); err != nil {
		return fmt.Errorf("stop track: %w", err)
	}

	return nil
}

func (p *Player) ToggleLoop(guildID snowflake.ID) (LoopResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	playback := p.playback(guildID)
	if !playback.playing || playback.current == nil {
		return LoopResult{}, fmt.Errorf("nothing is playing")
	}

	playback.looping = !playback.looping
	return LoopResult{
		Track:   playback.current.Track,
		Looping: playback.looping,
	}, nil
}

func (p *Player) Restart(ctx context.Context, guildID snowflake.ID) (lavalink.Track, error) {
	p.mu.Lock()
	playback := p.playback(guildID)
	if !playback.playing || playback.current == nil {
		p.mu.Unlock()
		return lavalink.Track{}, fmt.Errorf("nothing is playing")
	}
	current := playback.current.Track
	p.mu.Unlock()

	player := p.lavalinkPlayer(guildID)
	if err := player.Update(ctx, lavalink.WithPosition(0)); err != nil {
		return lavalink.Track{}, fmt.Errorf("restart track: %w", err)
	}

	return current, nil
}
