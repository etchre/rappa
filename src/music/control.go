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

	if err := p.playTrack(ctx, guildID, next.Track); err != nil {
		return SkipResult{}, err
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
		if p.startPremiumFallback(context.Background(), player, event.Track) {
			return
		}
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
			if advanceErr := p.advanceAfterFailedTrack(ctx, player.GuildID(), track); advanceErr != nil {
				fmt.Fprintf(os.Stderr, "advance after premium fallback failed: %v\n", advanceErr)
			}
		}
	}()

	return true
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

	if err := p.playTrack(ctx, guildID, next.Track); err != nil {
		return err
	}
	p.notifyAutoTrackStart(ctx, guildID)
	return nil
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
	identifier := originalTrackIdentifier(failedTrack)
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
