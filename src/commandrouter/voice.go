package commandrouter

import (
	"fmt"

	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

func CallerVoiceChannelID(event *events.ApplicationCommandInteractionCreate) (snowflake.ID, error) {
	guildID := event.GuildID()
	if guildID == nil {
		return 0, fmt.Errorf("this command can only be used in a server")
	}

	userID := event.User().ID
	voiceState, err := event.Client().Rest.GetUserVoiceState(*guildID, userID)
	if err == nil && voiceState.ChannelID != nil {
		return *voiceState.ChannelID, nil
	}

	if cachedVoiceState, ok := event.Client().Caches.VoiceState(*guildID, userID); ok && cachedVoiceState.ChannelID != nil {
		return *cachedVoiceState.ChannelID, nil
	}

	return 0, fmt.Errorf("join a voice channel before using /play")
}
