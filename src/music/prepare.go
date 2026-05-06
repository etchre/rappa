package music

import (
	"context"
	"log/slog"
	"os"
	"strconv"

	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"

	"rappa/utils"
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
	if utils.YouTubeVideoID(identifier) == "" {
		return preparedTrackResult{state: trackPreparationFailed}
	}

	slog.Debug("probing track for preparation", "track", TrackTitle(item.Track))
	result, err := utils.ProbeIdentifier(ctx, identifier)
	if err != nil {
		slog.Error("probe failed during preparation", "track", TrackTitle(item.Track), "err", err)
	}

	switch result {
	case utils.ProbeResultPremium:
		slog.Debug("prep result: premium", "track", TrackTitle(item.Track))
		return preparedTrackResult{
			state:      trackPreparationPremiumLikely,
			identifier: identifier,
		}

	case utils.ProbeResultUnavailable:
		slog.Debug("prep result: unavailable, trying resolver", "track", TrackTitle(item.Track))
		resolved := utils.ResolvedYouTubeIdentifier(ctx, identifier)
		if resolved == identifier {
			slog.Warn("resolver returned same identifier during prep", "track", TrackTitle(item.Track))
			return preparedTrackResult{state: trackPreparationFailed}
		}
		slog.Debug("resolved during prep, pre-loading", "track", TrackTitle(item.Track), "resolved", resolved)
		node := p.node()
		if node == nil {
			return preparedTrackResult{state: trackPreparationFailed}
		}
		loaded, err := loadPlayableTracks(ctx, node, resolved)
		if err != nil {
			slog.Error("resolved load failed during prep", "track", TrackTitle(item.Track), "err", err)
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
		slog.Debug("prep result: available, pre-loading", "track", TrackTitle(item.Track))
		node := p.node()
		if node == nil {
			return preparedTrackResult{state: trackPreparationFailed}
		}
		loaded, err := loadPlayableTracks(ctx, node, identifier)
		if err != nil {
			slog.Error("load failed during prep", "track", TrackTitle(item.Track), "err", err)
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
