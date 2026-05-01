package music

import (
	"fmt"
	"sync"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

type Player struct {
	lavalink          disgolink.Client
	preferredNodeName string
	premiumNodeName   string

	mu     sync.Mutex
	guilds map[snowflake.ID]*guildPlayback
}

type guildPlayback struct {
	current             *lavalink.Track
	queue               []lavalink.Track
	premiumAllowed      map[string]bool
	playing             bool
	looping             bool
	premiumFallbackBusy bool
	voiceChannelID      *snowflake.ID
	voiceSessionID      string
	voiceServerToken    string
	voiceEndpoint       string
}

type QueueResult struct {
	Track          lavalink.Track
	Tracks         []lavalink.Track
	Queued         bool
	Position       int
	Added          int
	CollectionName string
	CollectionKind string
}

type AddOptions struct {
	PremiumFallbackAllowed bool
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

func NewPlayer(lavalinkClient disgolink.Client, preferredNodeName string, premiumNodeName string) *Player {
	return &Player{
		lavalink:          lavalinkClient,
		preferredNodeName: preferredNodeName,
		premiumNodeName:   premiumNodeName,
		guilds:            map[snowflake.ID]*guildPlayback{},
	}
}

func (p *Player) node() disgolink.Node {
	if p.preferredNodeName != "" {
		if node := p.lavalink.Node(p.preferredNodeName); node != nil {
			return node
		}
	}

	return p.lavalink.BestNode()
}

func (p *Player) premiumNode() disgolink.Node {
	if p.premiumNodeName == "" {
		return nil
	}

	return p.lavalink.Node(p.premiumNodeName)
}

func (p *Player) lavalinkPlayer(guildID snowflake.ID) disgolink.Player {
	node := p.node()
	return p.lavalinkPlayerOnNode(guildID, node)
}

func (p *Player) lavalinkPlayerOnNode(guildID snowflake.ID, node disgolink.Node) disgolink.Player {
	player := p.lavalink.PlayerOnNode(node, guildID)
	if node != nil && player.Node() != nil && player.Node().Config().Name != node.Config().Name {
		fmt.Printf(
			"[lavalink-debug] existing player node mismatch guild=%s existing_node=%q preferred_node=%q\n",
			guildID.String(),
			player.Node().Config().Name,
			node.Config().Name,
		)
	}

	return player
}

func (p *Player) playback(guildID snowflake.ID) *guildPlayback {
	playback := p.guilds[guildID]
	if playback == nil {
		playback = &guildPlayback{premiumAllowed: map[string]bool{}}
		p.guilds[guildID] = playback
	}

	return playback
}

func (playback *guildPlayback) setPremiumFallbackAllowed(tracks []lavalink.Track, allowed bool) {
	if playback.premiumAllowed == nil {
		playback.premiumAllowed = map[string]bool{}
	}
	for _, track := range tracks {
		playback.premiumAllowed[track.Encoded] = allowed
	}
}

func (playback *guildPlayback) premiumFallbackAllowedFor(track lavalink.Track) bool {
	if playback.premiumAllowed == nil {
		return false
	}
	return playback.premiumAllowed[track.Encoded]
}
