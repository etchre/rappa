package music

import (
	"context"
	"fmt"

	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

func (p *Player) Add(ctx context.Context, guildID snowflake.ID, identifier string) (QueueResult, error) {
	return p.add(ctx, guildID, identifier, false)
}

func (p *Player) AddNext(ctx context.Context, guildID snowflake.ID, identifier string) (QueueResult, error) {
	return p.add(ctx, guildID, identifier, true)
}

func (p *Player) add(ctx context.Context, guildID snowflake.ID, identifier string, next bool) (QueueResult, error) {
	node := p.lavalink.BestNode()
	if node == nil {
		return QueueResult{}, fmt.Errorf("no lavalink node is available")
	}

	track, err := loadPlayableTrack(ctx, node, identifier)
	if err != nil {
		return QueueResult{}, err
	}

	p.mu.Lock()
	playback := p.playback(guildID)
	if playback.playing {
		if next {
			playback.queue = append([]lavalink.Track{track}, playback.queue...)
		} else {
			playback.queue = append(playback.queue, track)
		}
		position := len(playback.queue)
		if next {
			position = 1
		}
		p.mu.Unlock()

		return QueueResult{
			Track:    track,
			Queued:   true,
			Position: position,
		}, nil
	}
	playback.playing = true
	playback.current = &track
	p.mu.Unlock()

	if err := p.playTrack(ctx, guildID, track); err != nil {
		p.mu.Lock()
		playback := p.playback(guildID)
		playback.playing = false
		playback.current = nil
		p.mu.Unlock()

		return QueueResult{}, err
	}

	return QueueResult{Track: track}, nil
}
