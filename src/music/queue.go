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

func (p *Player) RemoveSlice(guildID snowflake.ID, from int, to int) ([]lavalink.Track, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	playback := p.playback(guildID)
	length := len(playback.queue)
	if length == 0 {
		return nil, fmt.Errorf("the queue is empty")
	}
	if from < 1 || from > length {
		return nil, fmt.Errorf("'from' position %d is invalid; choose a number between 1 and %d", from, length)
	}
	if to < from {
		return nil, fmt.Errorf("'to' position %d must be greater than or equal to 'from' position %d", to, from)
	}
	if to > length {
		to = length
	}

	startIdx := from - 1
	removed := make([]lavalink.Track, to-from+1)
	for i, item := range playback.queue[startIdx:to] {
		removed[i] = item.Track
	}
	playback.queue = append(playback.queue[:startIdx], playback.queue[to:]...)

	return removed, nil
}

func (p *Player) Move(guildID snowflake.ID, from int, to int) (lavalink.Track, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	playback := p.playback(guildID)
	length := len(playback.queue)

	fromIdx, err := queueIndex(from, length)
	if err != nil {
		return lavalink.Track{}, fmt.Errorf("'from' %v", err)
	}
	toIdx, err := queueIndex(to, length)
	if err != nil {
		return lavalink.Track{}, fmt.Errorf("'to' %v", err)
	}
	if fromIdx == toIdx {
		return playback.queue[fromIdx].Track, nil
	}

	item := playback.queue[fromIdx]
	playback.queue = append(playback.queue[:fromIdx], playback.queue[fromIdx+1:]...)
	playback.queue = append(playback.queue[:toIdx], append([]queuedTrack{item}, playback.queue[toIdx:]...)...)

	return item.Track, nil
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
