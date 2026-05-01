package commandrouter

import (
	"sync"

	"github.com/disgoorg/snowflake/v2"
)

type StatusChannels struct {
	mu         sync.RWMutex
	configured map[snowflake.ID]snowflake.ID
	fallback   map[snowflake.ID]snowflake.ID
}

func NewStatusChannels() *StatusChannels {
	return &StatusChannels{
		configured: map[snowflake.ID]snowflake.ID{},
		fallback:   map[snowflake.ID]snowflake.ID{},
	}
}

func (channels *StatusChannels) Set(guildID snowflake.ID, channelID snowflake.ID) {
	channels.mu.Lock()
	defer channels.mu.Unlock()

	channels.configured[guildID] = channelID
}

func (channels *StatusChannels) SetFallbackIfUnset(guildID snowflake.ID, channelID snowflake.ID) {
	channels.mu.Lock()
	defer channels.mu.Unlock()

	if _, ok := channels.configured[guildID]; ok {
		return
	}
	if _, ok := channels.fallback[guildID]; ok {
		return
	}

	channels.fallback[guildID] = channelID
}

func (channels *StatusChannels) Get(guildID snowflake.ID) (snowflake.ID, bool) {
	channels.mu.RLock()
	defer channels.mu.RUnlock()

	if channelID, ok := channels.configured[guildID]; ok {
		return channelID, true
	}

	channelID, ok := channels.fallback[guildID]
	return channelID, ok
}
