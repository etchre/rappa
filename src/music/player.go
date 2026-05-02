package music

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

type Player struct {
	lavalink             disgolink.Client
	autoTrackStartNotify func(ctx context.Context, guildID snowflake.ID)
	trackFailureNotify   func(ctx context.Context, guildID snowflake.ID, track lavalink.Track)
	playbackActiveNotify func(ctx context.Context, guildID snowflake.ID)
	playbackIdleNotify   func(ctx context.Context, guildID snowflake.ID)
	nextQueuedTrackID    atomic.Uint64

	mu     sync.Mutex
	guilds map[snowflake.ID]*guildPlayback
}

type guildPlayback struct {
	current             *queuedTrack
	queue               []queuedTrack
	playing             bool
	paused              bool
	looping             bool
	premiumFallbackBusy bool
	exceptionTrack      string    // encoded track that received an exception
	playStartTime       time.Time // when the current track started playing
}

type queuedTrack struct {
	ID                     uint64
	Track                  lavalink.Track
	RecoveryQuery          string
	RecoveryAttempted      bool
	ResolvedRetryAttempted bool
	Preparation            trackPreparation
	PreparedIdentifier     string
	PreparedTrack          *lavalink.Track
	PremiumAllowed         bool
	UsedPremiumRoute       bool
	RequesterName          string
	RequesterID            string
	PremiumAllowedUserIDs  string
	PremiumFailureLogged   bool
	PremiumRefusalLogged   bool
}

type trackPreparation int

const (
	trackPreparationNone trackPreparation = iota
	trackPreparationInProgress
	trackPreparationResolvedLavalink
	trackPreparationPremiumLikely
	trackPreparationFailed
)

type QueueResult struct {
	Track          lavalink.Track
	Tracks         []lavalink.Track
	Queued         bool
	Position       int
	Added          int
	Shuffled       bool
	CollectionName string
	CollectionKind string
}

type AddOptions struct {
	PremiumFallbackAllowed bool
	RequesterName          string
	RequesterID            string
	PremiumAllowedUserIDs  string
	Shuffle                bool
	Limit                  int // max tracks to keep from a collection (0 = no limit)
}

type QueueSnapshot struct {
	Current  *lavalink.Track
	Queued   []lavalink.Track
	Position lavalink.Duration
	Volume   int
}

type SkipResult struct {
	Next    *lavalink.Track
	Stopped bool
}

type LoopResult struct {
	Track   lavalink.Track
	Looping bool
}

type PauseResult struct {
	Track   lavalink.Track
	Paused  bool
	Changed bool
}

func NewPlayer(lavalinkClient disgolink.Client) *Player {
	return &Player{
		lavalink: lavalinkClient,
		guilds:   map[snowflake.ID]*guildPlayback{},
	}
}

func (p *Player) SetAutoTrackStartNotifier(notify func(ctx context.Context, guildID snowflake.ID)) {
	p.autoTrackStartNotify = notify
}

func (p *Player) SetTrackFailureNotifier(notify func(ctx context.Context, guildID snowflake.ID, track lavalink.Track)) {
	p.trackFailureNotify = notify
}

func (p *Player) SetPlaybackActiveNotifier(notify func(ctx context.Context, guildID snowflake.ID)) {
	p.playbackActiveNotify = notify
}

func (p *Player) SetPlaybackIdleNotifier(notify func(ctx context.Context, guildID snowflake.ID)) {
	p.playbackIdleNotify = notify
}

func (p *Player) node() disgolink.Node {
	return p.lavalink.BestNode()
}

func (p *Player) lavalinkPlayer(guildID snowflake.ID) disgolink.Player {
	return p.lavalink.Player(guildID)
}

func (p *Player) playback(guildID snowflake.ID) *guildPlayback {
	playback := p.guilds[guildID]
	if playback == nil {
		playback = &guildPlayback{}
		p.guilds[guildID] = playback
	}

	return playback
}

func (p *Player) queuedTracks(tracks []lavalink.Track, options AddOptions) []queuedTrack {
	items := make([]queuedTrack, len(tracks))
	for i, track := range tracks {
		items[i] = queuedTrack{
			ID:                    p.nextQueuedTrackID.Add(1),
			Track:                 track,
			RecoveryQuery:         recoveryQuery(track),
			PremiumAllowed:        options.PremiumFallbackAllowed,
			RequesterName:         options.RequesterName,
			RequesterID:           options.RequesterID,
			PremiumAllowedUserIDs: options.PremiumAllowedUserIDs,
		}
	}
	return items
}

func tracksFromQueue(items []queuedTrack) []lavalink.Track {
	tracks := make([]lavalink.Track, len(items))
	for i, item := range items {
		tracks[i] = item.Track
	}
	return tracks
}

func requesterName(item queuedTrack) string {
	if item.RequesterName != "" {
		return item.RequesterName
	}
	return "unknown user"
}

func requesterID(item queuedTrack) string {
	if item.RequesterID != "" {
		return item.RequesterID
	}
	return "unknown"
}
