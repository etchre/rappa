package music

import (
	"context"
	"fmt"

	"github.com/disgoorg/snowflake/v2"
)

func (p *Player) Add(ctx context.Context, guildID snowflake.ID, identifier string) (QueueResult, error) {
	return p.add(ctx, guildID, identifier, false)
}

func (p *Player) AddNext(ctx context.Context, guildID snowflake.ID, identifier string) (QueueResult, error) {
	return p.add(ctx, guildID, identifier, true)
}

func (p *Player) PlayNow(ctx context.Context, guildID snowflake.ID, identifier string) (QueueResult, error) {
	node := p.lavalink.BestNode()
	if node == nil {
		return QueueResult{}, fmt.Errorf("no lavalink node is available")
	}

	tracks, err := loadPlayableTracks(ctx, node, identifier)
	if err != nil {
		return QueueResult{}, err
	}
	track := tracks[0]

	p.mu.Lock()
	playback := p.playback(guildID)
	previousCurrent := playback.current
	previousQueue := playback.queue
	wasPlaying := playback.playing
	wasLooping := playback.looping
	playback.playing = true
	playback.current = &track
	playback.queue = append(tracks[1:], playback.queue...)
	playback.looping = false
	p.mu.Unlock()

	if err := p.playTrack(ctx, guildID, track); err != nil {
		p.mu.Lock()
		playback := p.playback(guildID)
		playback.playing = wasPlaying
		playback.current = previousCurrent
		playback.queue = previousQueue
		playback.looping = wasLooping
		p.mu.Unlock()

		return QueueResult{}, err
	}

	return QueueResult{Track: track, Added: len(tracks)}, nil
}

func (p *Player) add(ctx context.Context, guildID snowflake.ID, identifier string, next bool) (QueueResult, error) {
	node := p.lavalink.BestNode()
	if node == nil {
		return QueueResult{}, fmt.Errorf("no lavalink node is available")
	}

	tracks, err := loadPlayableTracks(ctx, node, identifier)
	if err != nil {
		return QueueResult{}, err
	}
	track := tracks[0]

	p.mu.Lock()
	playback := p.playback(guildID)
	if playback.playing {
		if next {
			playback.queue = append(tracks, playback.queue...)
		} else {
			playback.queue = append(playback.queue, tracks...)
		}
		position := len(playback.queue)
		if next {
			position = 1
		} else {
			position = position - len(tracks) + 1
		}
		p.mu.Unlock()

		return QueueResult{
			Track:    track,
			Queued:   true,
			Position: position,
			Added:    len(tracks),
		}, nil
	}
	previousQueue := playback.queue
	playback.playing = true
	playback.current = &track
	playback.queue = append(playback.queue, tracks[1:]...)
	p.mu.Unlock()

	if err := p.playTrack(ctx, guildID, track); err != nil {
		p.mu.Lock()
		playback := p.playback(guildID)
		playback.playing = false
		playback.current = nil
		playback.queue = previousQueue
		p.mu.Unlock()

		return QueueResult{}, err
	}

	return QueueResult{Track: track, Added: len(tracks)}, nil
}
