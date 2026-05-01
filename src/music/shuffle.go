package music

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

func (p *Player) ShuffleQueue(guildID snowflake.ID) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	playback := p.playback(guildID)
	if len(playback.queue) < 2 {
		return len(playback.queue), nil
	}

	shuffleQueuedTracks(playback.queue)
	return len(playback.queue), nil
}

func (p *Player) ShuffleAll(ctx context.Context, guildID snowflake.ID) (lavalink.Track, int, error) {
	p.mu.Lock()
	playback := p.playback(guildID)
	if playback.current == nil {
		p.mu.Unlock()
		return lavalink.Track{}, 0, fmt.Errorf("nothing is playing")
	}
	if len(playback.queue) == 0 {
		p.mu.Unlock()
		return lavalink.Track{}, 0, fmt.Errorf("the queue is empty")
	}

	current := *playback.current
	nextIndex := rand.Intn(len(playback.queue))
	next := playback.queue[nextIndex]

	queue := append([]queuedTrack{}, playback.queue[:nextIndex]...)
	queue = append(queue, playback.queue[nextIndex+1:]...)
	queue = append(queue, current)
	shuffleQueuedTracks(queue)

	playback.current = &next
	playback.queue = queue
	playback.playing = true
	playback.paused = false
	playback.looping = false
	p.mu.Unlock()

	if err := p.playTrack(ctx, guildID, next.Track); err != nil {
		return lavalink.Track{}, 0, err
	}

	return next.Track, len(queue), nil
}

func shuffledTracks(tracks []lavalink.Track) []lavalink.Track {
	shuffled := append([]lavalink.Track{}, tracks...)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled
}

func shuffleQueuedTracks(tracks []queuedTrack) {
	rand.Shuffle(len(tracks), func(i, j int) {
		tracks[i], tracks[j] = tracks[j], tracks[i]
	})
}
