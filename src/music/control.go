package music

import (
	"context"
	"fmt"
	"os"

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
	p.mu.Unlock()

	if err := p.playQueuedTrack(ctx, guildID, next); err != nil {
		return SkipResult{}, err
	}
	go p.prepareQueueAhead(guildID)

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
		p.startPremiumFallback(context.Background(), player, event.Track)
		return
	}

	if !event.Reason.MayStartNext() {
		return
	}

	if err := p.playNext(context.Background(), player.GuildID()); err != nil {
		fmt.Fprintf(os.Stderr, "play next failed: %v\n", err)
	}
}

func (p *Player) OnTrackException(player disgolink.Player, event lavalink.TrackExceptionEvent) {
	if !looksAccountGated(event.Exception.Message, event.Exception.Cause, event.Exception.CauseStackTrace) {
		debugLavalinkException("playback", playerNodeName(player), trackTitle(event.Track), event.Exception)
		return
	}

	p.startPremiumFallback(context.Background(), player, event.Track)
}

func (p *Player) startPremiumFallback(ctx context.Context, player disgolink.Player, track lavalink.Track) bool {
	p.mu.Lock()
	playback := p.playback(player.GuildID())
	if playback.premiumFallbackBusy {
		p.mu.Unlock()
		return true
	}
	if playback.current == nil || playback.current.Track.Encoded != track.Encoded {
		var currentTitle string
		if playback.current != nil {
			currentTitle = trackTitle(playback.current.Track)
		}
		p.mu.Unlock()
		fmt.Printf(
			"Ignored stale playback failure for %s; current track is %q\n",
			trackTitle(track),
			currentTitle,
		)
		return false
	}
	if !playback.current.PremiumFailureLogged {
		fmt.Printf("Failed to play, likely a premium track: %s\n", trackTitle(track))
		playback.current.PremiumFailureLogged = true
	}

	current := *playback.current
	if !current.ResolvedRetryAttempted {
		playback.current.ResolvedRetryAttempted = true
		playback.premiumFallbackBusy = true
		p.mu.Unlock()
		go p.retryResolvedThenFallback(ctx, player, track)
		return true
	}

	if !current.PremiumAllowed {
		if !playback.current.PremiumRefusalLogged {
			fmt.Printf(
				"Refused to use premium fallback for %s requester_id=%s allowed_user_ids=%q\n",
				requesterName(current),
				requesterID(current),
				current.PremiumAllowedUserIDs,
			)
			playback.current.PremiumRefusalLogged = true
		}
		p.mu.Unlock()
		go func() {
			if p.recoverFailedTrackBySearch(ctx, player.GuildID(), track) {
				return
			}
			if err := p.advanceAfterFailedTrack(ctx, player.GuildID(), track); err != nil {
				fmt.Fprintf(os.Stderr, "advance after refused premium fallback failed: %v\n", err)
			}
		}()
		return true
	}
	fmt.Printf("Trying premium fallback for %s\n", trackTitle(track))
	playback.premiumFallbackBusy = true
	p.mu.Unlock()

	go func() {
		if err := p.fallbackToPremium(ctx, player.GuildID(), track); err != nil {
			fmt.Fprintf(os.Stderr, "premium fallback failed: %v\n", err)
			if p.recoverFailedTrackBySearch(ctx, player.GuildID(), track) {
				return
			}
			if advanceErr := p.advanceAfterFailedTrack(ctx, player.GuildID(), track); advanceErr != nil {
				fmt.Fprintf(os.Stderr, "advance after premium fallback failed: %v\n", advanceErr)
			}
		}
	}()

	return true
}

func (p *Player) retryResolvedThenFallback(ctx context.Context, player disgolink.Player, failedTrack lavalink.Track) {
	guildID := player.GuildID()
	if err := p.retryWithResolvedLavalinkTrack(ctx, guildID, failedTrack); err == nil {
		p.setPremiumFallbackBusy(guildID, false)
		return
	} else {
		fmt.Fprintf(os.Stderr, "resolved lavalink retry failed: %v\n", err)
	}

	p.mu.Lock()
	playback := p.playback(guildID)
	if playback.current == nil || playback.current.Track.Encoded != failedTrack.Encoded {
		playback.premiumFallbackBusy = false
		p.mu.Unlock()
		return
	}

	current := *playback.current
	if !current.PremiumAllowed {
		if !playback.current.PremiumRefusalLogged {
			fmt.Printf(
				"Refused to use premium fallback for %s requester_id=%s allowed_user_ids=%q\n",
				requesterName(current),
				requesterID(current),
				current.PremiumAllowedUserIDs,
			)
			playback.current.PremiumRefusalLogged = true
		}
		playback.premiumFallbackBusy = false
		p.mu.Unlock()

		if p.recoverFailedTrackBySearch(ctx, guildID, failedTrack) {
			return
		}
		if err := p.advanceAfterFailedTrack(ctx, guildID, failedTrack); err != nil {
			fmt.Fprintf(os.Stderr, "advance after refused premium fallback failed: %v\n", err)
		}
		return
	}
	p.mu.Unlock()

	fmt.Printf("Trying premium fallback for %s\n", trackTitle(failedTrack))
	if err := p.fallbackToPremium(ctx, guildID, failedTrack); err != nil {
		fmt.Fprintf(os.Stderr, "premium fallback failed: %v\n", err)
		if p.recoverFailedTrackBySearch(ctx, guildID, failedTrack) {
			return
		}
		if advanceErr := p.advanceAfterFailedTrack(ctx, guildID, failedTrack); advanceErr != nil {
			fmt.Fprintf(os.Stderr, "advance after premium fallback failed: %v\n", advanceErr)
		}
	}
}

func (p *Player) retryWithResolvedLavalinkTrack(ctx context.Context, guildID snowflake.ID, failedTrack lavalink.Track) error {
	identifier := resolvedTrackIdentifier(ctx, failedTrack)
	if identifier == "" {
		return fmt.Errorf("cannot determine original track URL for resolved lavalink retry")
	}

	fmt.Printf("Trying resolved Lavalink retry for %s identifier=%q\n", trackTitle(failedTrack), identifier)
	loaded, err := loadPlayableTracks(ctx, p.node(), identifier)
	if err != nil {
		return fmt.Errorf("load resolved lavalink retry track: %w", err)
	}
	if len(loaded.Tracks) == 0 {
		return fmt.Errorf("resolved lavalink retry returned no tracks")
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
	if !shouldPlay {
		return nil
	}

	return p.playTrack(ctx, guildID, retryTrack)
}

func (p *Player) setPremiumFallbackBusy(guildID snowflake.ID, busy bool) {
	p.mu.Lock()
	p.playback(guildID).premiumFallbackBusy = busy
	p.mu.Unlock()
}

func (p *Player) logPremiumFailure(guildID snowflake.ID, track lavalink.Track) {
	p.mu.Lock()
	defer p.mu.Unlock()

	playback := p.playback(guildID)
	if playback.current != nil && playback.current.Track.Encoded == track.Encoded && playback.current.PremiumFailureLogged {
		return
	}

	fmt.Printf("Failed to play, likely a premium track: %s\n", trackTitle(track))
	if playback.current != nil && playback.current.Track.Encoded == track.Encoded {
		playback.current.PremiumFailureLogged = true
	}
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
			preparedTrack := *item.PreparedTrack
			p.replaceCurrentTrack(guildID, item.ID, preparedTrack)
			return p.playTrack(ctx, guildID, preparedTrack)
		}
	case trackPreparationPremiumLikely:
		if item.PreparedIdentifier != "" {
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

	p.notifyPlaybackActive(ctx, guildID)
	fmt.Printf("Now playing on Lavalink node %q: %s\n", playerNodeName(player), trackTitle(track))
	return nil
}

func (p *Player) fallbackToPremium(ctx context.Context, guildID snowflake.ID, failedTrack lavalink.Track) error {
	identifier := resolvedTrackIdentifier(ctx, failedTrack)
	if identifier == "" {
		return fmt.Errorf("cannot determine original track URL for premium fallback")
	}

	defer func() {
		p.mu.Lock()
		p.playback(guildID).premiumFallbackBusy = false
		p.mu.Unlock()
	}()

	loaded, err := loadPlayableTracks(ctx, p.node(), premiumPlayableIdentifier(identifier))
	if err != nil {
		return fmt.Errorf("load premium fallback track: %w", err)
	}
	if len(loaded.Tracks) == 0 {
		return fmt.Errorf("premium fallback returned no tracks")
	}

	premiumTrack := loaded.Tracks[0]
	premiumTrack.Info = failedTrack.Info

	p.mu.Lock()
	playback := p.playback(guildID)
	if playback.current == nil || playback.current.Track.Encoded != failedTrack.Encoded {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()

	p.mu.Lock()
	playback = p.playback(guildID)
	shouldPlay := false
	if playback.current != nil && playback.current.Track.Encoded == failedTrack.Encoded {
		playback.current.Track = premiumTrack
		playback.playing = true
		playback.paused = false
		shouldPlay = true
	}
	p.mu.Unlock()
	if !shouldPlay {
		return nil
	}

	return p.playTrack(ctx, guildID, premiumTrack)
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

	fmt.Printf("Trying search recovery for %s query=%q\n", trackTitle(failedTrack), query)
	loaded, err := loadPlayableTracks(ctx, p.node(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "search recovery failed: %v\n", err)
		return false
	}
	if len(loaded.Tracks) == 0 {
		fmt.Fprintf(os.Stderr, "search recovery returned no tracks query=%q\n", query)
		return false
	}

	recoveredTrack := loaded.Tracks[0]
	if recoveredTrack.Encoded == failedTrack.Encoded {
		fmt.Fprintf(os.Stderr, "search recovery returned the same failed track query=%q\n", query)
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
		fmt.Fprintf(os.Stderr, "play search recovery failed: %v\n", err)
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
