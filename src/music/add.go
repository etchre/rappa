package music

import (
	"context"
	"fmt"

	"github.com/disgoorg/snowflake/v2"
)

func (p *Player) Add(ctx context.Context, guildID snowflake.ID, identifier string, options AddOptions) (QueueResult, error) {
	return p.add(ctx, guildID, identifier, false, options)
}

func (p *Player) AddNext(ctx context.Context, guildID snowflake.ID, identifier string, options AddOptions) (QueueResult, error) {
	return p.add(ctx, guildID, identifier, true, options)
}

func (p *Player) PlayNow(ctx context.Context, guildID snowflake.ID, identifier string, options AddOptions) (QueueResult, error) {
	node := p.node()
	if node == nil {
		return QueueResult{}, fmt.Errorf("no lavalink node is available")
	}

	loaded, err := loadPlayableTracks(ctx, node, identifier)
	if err != nil {
		return QueueResult{}, err
	}
	tracks := loaded.Tracks
	if options.Shuffle && len(tracks) > 1 {
		tracks = shuffledTracks(tracks)
	}
	items := queuedTracks(tracks, options)
	item := items[0]

	p.mu.Lock()
	playback := p.playback(guildID)
	previousCurrent := playback.current
	previousQueue := playback.queue
	wasPlaying := playback.playing
	wasLooping := playback.looping
	playback.playing = true
	playback.current = &item
	playback.queue = append(items[1:], playback.queue...)
	playback.looping = false
	p.mu.Unlock()

	if err := p.playTrack(ctx, guildID, item.Track); err != nil {
		p.mu.Lock()
		playback := p.playback(guildID)
		playback.playing = wasPlaying
		playback.current = previousCurrent
		playback.queue = previousQueue
		playback.looping = wasLooping
		p.mu.Unlock()

		return QueueResult{}, err
	}

	return QueueResult{
		Track:          item.Track,
		Tracks:         tracks,
		Added:          len(tracks),
		CollectionName: loaded.CollectionName,
		CollectionKind: loaded.CollectionKind,
	}, nil
}

func (p *Player) add(ctx context.Context, guildID snowflake.ID, identifier string, next bool, options AddOptions) (QueueResult, error) {
	node := p.node()
	if node == nil {
		return QueueResult{}, fmt.Errorf("no lavalink node is available")
	}

	loaded, err := loadPlayableTracks(ctx, node, identifier)
	if err != nil {
		return QueueResult{}, err
	}
	tracks := loaded.Tracks
	if options.Shuffle && len(tracks) > 1 {
		tracks = shuffledTracks(tracks)
	}
	items := queuedTracks(tracks, options)
	item := items[0]

	p.mu.Lock()
	playback := p.playback(guildID)
	if playback.playing {
		if next {
			playback.queue = append(items, playback.queue...)
		} else {
			playback.queue = append(playback.queue, items...)
		}
		position := len(playback.queue)
		if next {
			position = 1
		} else {
			position = position - len(tracks) + 1
		}
		p.mu.Unlock()

		return QueueResult{
			Track:          item.Track,
			Tracks:         tracks,
			Queued:         true,
			Position:       position,
			Added:          len(tracks),
			CollectionName: loaded.CollectionName,
			CollectionKind: loaded.CollectionKind,
		}, nil
	}
	previousQueue := playback.queue
	playback.playing = true
	playback.current = &item
	playback.queue = append(playback.queue, items[1:]...)
	p.mu.Unlock()

	if err := p.playTrack(ctx, guildID, item.Track); err != nil {
		p.mu.Lock()
		playback := p.playback(guildID)
		playback.playing = false
		playback.current = nil
		playback.queue = previousQueue
		p.mu.Unlock()

		return QueueResult{}, err
	}

	return QueueResult{
		Track:          item.Track,
		Tracks:         tracks,
		Added:          len(tracks),
		CollectionName: loaded.CollectionName,
		CollectionKind: loaded.CollectionKind,
	}, nil
}
