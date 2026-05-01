package music

import (
	"context"
	"fmt"

	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

func (p *Player) TogglePause(ctx context.Context, guildID snowflake.ID) (PauseResult, error) {
	p.mu.Lock()
	playback := p.playback(guildID)
	if !playback.playing || playback.current == nil {
		p.mu.Unlock()
		return PauseResult{}, fmt.Errorf("nothing is playing")
	}

	paused := !playback.paused
	current := playback.current.Track
	playback.paused = paused
	p.mu.Unlock()

	if err := p.lavalinkPlayer(guildID).Update(ctx, lavalink.WithPaused(paused)); err != nil {
		p.mu.Lock()
		playback := p.playback(guildID)
		if playback.current != nil && playback.current.Track.Encoded == current.Encoded {
			playback.paused = !paused
		}
		p.mu.Unlock()
		return PauseResult{}, fmt.Errorf("update pause state: %w", err)
	}

	if paused {
		p.notifyPlaybackActive(ctx, guildID)
	} else {
		p.notifyPlaybackActive(ctx, guildID)
	}

	return PauseResult{Track: current, Paused: paused, Changed: true}, nil
}

func (p *Player) Unpause(ctx context.Context, guildID snowflake.ID) (PauseResult, error) {
	p.mu.Lock()
	playback := p.playback(guildID)
	if !playback.playing || playback.current == nil {
		p.mu.Unlock()
		return PauseResult{}, fmt.Errorf("nothing is playing")
	}
	if !playback.paused {
		current := playback.current.Track
		p.mu.Unlock()
		return PauseResult{Track: current, Paused: false, Changed: false}, nil
	}

	current := playback.current.Track
	playback.paused = false
	p.mu.Unlock()

	if err := p.lavalinkPlayer(guildID).Update(ctx, lavalink.WithPaused(false)); err != nil {
		p.mu.Lock()
		playback := p.playback(guildID)
		if playback.current != nil && playback.current.Track.Encoded == current.Encoded {
			playback.paused = true
		}
		p.mu.Unlock()
		return PauseResult{}, fmt.Errorf("resume playback: %w", err)
	}

	p.notifyPlaybackActive(ctx, guildID)
	return PauseResult{Track: current, Paused: false, Changed: true}, nil
}
