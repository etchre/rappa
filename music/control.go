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
	if len(playback.queue) == 0 {
		wasPlaying := playback.playing
		playback.playing = false
		playback.current = nil
		p.mu.Unlock()

		if wasPlaying {
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
	p.mu.Unlock()

	if err := p.playTrack(ctx, guildID, next); err != nil {
		return SkipResult{}, err
	}

	return SkipResult{Next: &next}, nil
}

func (p *Player) Stop(ctx context.Context, guildID snowflake.ID) error {
	p.mu.Lock()
	playback := p.playback(guildID)
	wasPlaying := playback.playing
	playback.playing = false
	playback.current = nil
	playback.queue = nil
	p.mu.Unlock()

	if !wasPlaying {
		return nil
	}

	return p.clearLavalinkTrack(ctx, guildID)
}

func (p *Player) OnTrackEnd(player disgolink.Player, event lavalink.TrackEndEvent) {
	if !event.Reason.MayStartNext() {
		return
	}

	if err := p.playNext(context.Background(), player.GuildID()); err != nil {
		fmt.Fprintf(os.Stderr, "play next failed: %v\n", err)
	}
}

func (p *Player) playNext(ctx context.Context, guildID snowflake.ID) error {
	p.mu.Lock()
	playback := p.playback(guildID)
	if len(playback.queue) == 0 {
		playback.playing = false
		playback.current = nil
		p.mu.Unlock()
		return nil
	}

	next := playback.queue[0]
	playback.queue = playback.queue[1:]
	playback.current = &next
	p.mu.Unlock()

	return p.playTrack(ctx, guildID, next)
}

func (p *Player) playTrack(ctx context.Context, guildID snowflake.ID, track lavalink.Track) error {
	player := p.lavalink.Player(guildID)
	if err := player.Update(ctx, lavalink.WithTrack(track)); err != nil {
		return fmt.Errorf("play track: %w", err)
	}

	fmt.Printf("Now playing: %s - %s\n", track.Info.Author, track.Info.Title)
	return nil
}

func (p *Player) clearLavalinkTrack(ctx context.Context, guildID snowflake.ID) error {
	player := p.lavalink.Player(guildID)
	if err := player.Update(ctx, lavalink.WithNullTrack()); err != nil {
		return fmt.Errorf("stop track: %w", err)
	}

	return nil
}
