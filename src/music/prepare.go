package music

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

const defaultQueuePreparationLookahead = 5

type preparedTrackResult struct {
	state      trackPreparation
	identifier string
	track      *lavalink.Track
}

func (p *Player) prepareQueueAhead(guildID snowflake.ID) {
	for {
		item, ok := p.nextQueuedTrackToPrepare(guildID)
		if !ok {
			return
		}

		result := p.prepareQueuedTrack(context.Background(), item)
		p.applyPreparedTrack(guildID, item.ID, result)
	}
}

func (p *Player) nextQueuedTrackToPrepare(guildID snowflake.ID) (queuedTrack, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	playback := p.playback(guildID)
	limit := queuePreparationLookahead()
	if limit <= 0 {
		return queuedTrack{}, false
	}
	if len(playback.queue) < limit {
		limit = len(playback.queue)
	}

	for i := 0; i < limit; i++ {
		if playback.queue[i].Preparation != trackPreparationNone {
			continue
		}

		playback.queue[i].Preparation = trackPreparationInProgress
		return playback.queue[i], true
	}

	return queuedTrack{}, false
}

func queuePreparationLookahead() int {
	value := os.Getenv("QUEUE_PREPARATION_LOOKAHEAD")
	if value == "" {
		return defaultQueuePreparationLookahead
	}

	limit, err := strconv.Atoi(value)
	if err != nil {
		return defaultQueuePreparationLookahead
	}
	return limit
}

func (p *Player) prepareQueuedTrack(ctx context.Context, item queuedTrack) preparedTrackResult {
	identifier := unresolvedTrackIdentifier(item.Track)
	if identifier == "" {
		return preparedTrackResult{state: trackPreparationFailed}
	}

	// Only probe YouTube tracks.
	if YouTubeVideoID(identifier) == "" {
		return preparedTrackResult{state: trackPreparationFailed}
	}

	fmt.Printf("[prepare] probing %s\n", trackTitle(item.Track))
	result, err := probeIdentifier(ctx, identifier)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[prepare] probe error for %s: %v\n", trackTitle(item.Track), err)
	}

	switch result {
	case probeResultPremium:
		fmt.Printf("[prepare] %s -> premium, will use premium plugin\n", trackTitle(item.Track))
		return preparedTrackResult{
			state:      trackPreparationPremiumLikely,
			identifier: identifier,
		}

	case probeResultUnavailable:
		fmt.Printf("[prepare] %s -> unavailable, trying resolver\n", trackTitle(item.Track))
		resolved := resolvedYouTubeIdentifier(ctx, identifier)
		if resolved == identifier {
			fmt.Fprintf(os.Stderr, "[prepare] resolver returned same identifier for %s\n", trackTitle(item.Track))
			return preparedTrackResult{state: trackPreparationFailed}
		}
		fmt.Printf("[prepare] %s resolved -> %s, pre-loading via lavalink\n", trackTitle(item.Track), resolved)
		node := p.node()
		if node == nil {
			return preparedTrackResult{state: trackPreparationFailed}
		}
		loaded, err := loadPlayableTracks(ctx, node, resolved)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[prepare] resolved load failed for %s: %v\n", trackTitle(item.Track), err)
			return preparedTrackResult{state: trackPreparationFailed}
		}
		if len(loaded.Tracks) == 0 {
			return preparedTrackResult{state: trackPreparationFailed}
		}
		preparedTrack := loaded.Tracks[0]
		return preparedTrackResult{
			state:      trackPreparationResolvedLavalink,
			identifier: resolved,
			track:      &preparedTrack,
		}

	default:
		fmt.Printf("[prepare] %s -> available, pre-loading via lavalink\n", trackTitle(item.Track))
		node := p.node()
		if node == nil {
			return preparedTrackResult{state: trackPreparationFailed}
		}
		loaded, err := loadPlayableTracks(ctx, node, identifier)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[prepare] load failed for %s: %v\n", trackTitle(item.Track), err)
			return preparedTrackResult{state: trackPreparationFailed}
		}
		if len(loaded.Tracks) == 0 {
			return preparedTrackResult{state: trackPreparationFailed}
		}
		preparedTrack := loaded.Tracks[0]
		return preparedTrackResult{
			state:      trackPreparationResolvedLavalink,
			identifier: identifier,
			track:      &preparedTrack,
		}
	}
}

func (p *Player) applyPreparedTrack(guildID snowflake.ID, itemID uint64, result preparedTrackResult) {
	p.mu.Lock()
	defer p.mu.Unlock()

	playback := p.playback(guildID)
	for i := range playback.queue {
		if playback.queue[i].ID != itemID {
			continue
		}

		playback.queue[i].Preparation = result.state
		playback.queue[i].PreparedIdentifier = result.identifier
		playback.queue[i].PreparedTrack = result.track
		return
	}
}
