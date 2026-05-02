package music

import (
	"fmt"

	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

func (p *Player) ClearQueue(guildID snowflake.ID) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	playback := p.playback(guildID)
	cleared := len(playback.queue)
	playback.queue = nil

	return cleared
}

func (p *Player) MoveNext(guildID snowflake.ID, queueNumber int) (lavalink.Track, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	playback := p.playback(guildID)
	index, err := queueIndex(queueNumber, len(playback.queue))
	if err != nil {
		return lavalink.Track{}, err
	}

	item := playback.queue[index]
	if index == 0 {
		return item.Track, nil
	}

	playback.queue = append(playback.queue[:index], playback.queue[index+1:]...)
	playback.queue = append([]queuedTrack{item}, playback.queue...)
	go p.prepareQueueAhead(guildID)

	return item.Track, nil
}

func (p *Player) Remove(guildID snowflake.ID, queueNumber int) (lavalink.Track, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	playback := p.playback(guildID)
	index, err := queueIndex(queueNumber, len(playback.queue))
	if err != nil {
		return lavalink.Track{}, err
	}

	item := playback.queue[index]
	playback.queue = append(playback.queue[:index], playback.queue[index+1:]...)

	return item.Track, nil
}

func (p *Player) Queue(guildID snowflake.ID) QueueSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()

	playback := p.playback(guildID)
	queued := tracksFromQueue(playback.queue)

	var current *lavalink.Track
	var position lavalink.Duration
	var volume int
	if playback.current != nil {
		currentTrack := playback.current.Track
		current = &currentTrack

		player := p.lavalinkPlayer(guildID)
		position = player.Position()
		volume = player.Volume()
	}

	return QueueSnapshot{
		Current:  current,
		Queued:   queued,
		Position: position,
		Volume:   volume,
	}
}

func queueIndex(queueNumber int, queueLength int) (int, error) {
	if queueLength == 0 {
		return 0, fmt.Errorf("the queue is empty")
	}
	if queueNumber < 1 || queueNumber > queueLength {
		return 0, fmt.Errorf("queue number %d is invalid; choose a number between 1 and %d", queueNumber, queueLength)
	}

	return queueNumber - 1, nil
}
