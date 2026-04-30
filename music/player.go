package music

import (
	"sync"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

type Player struct {
	lavalink disgolink.Client

	mu     sync.Mutex
	guilds map[snowflake.ID]*guildPlayback
}

type guildPlayback struct {
	current *lavalink.Track
	queue   []lavalink.Track
	playing bool
}

type QueueResult struct {
	Track    lavalink.Track
	Queued   bool
	Position int
}

type QueueSnapshot struct {
	Current *lavalink.Track
	Queued  []lavalink.Track
}

type SkipResult struct {
	Next    *lavalink.Track
	Stopped bool
}

func NewPlayer(lavalinkClient disgolink.Client) *Player {
	return &Player{
		lavalink: lavalinkClient,
		guilds:   map[snowflake.ID]*guildPlayback{},
	}
}

func (p *Player) playback(guildID snowflake.ID) *guildPlayback {
	playback := p.guilds[guildID]
	if playback == nil {
		playback = &guildPlayback{}
		p.guilds[guildID] = playback
	}

	return playback
}
