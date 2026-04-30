package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

var Leave = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "leave",
		Description: "Stop playback, clear the queue, and leave voice",
	},
	Handle: handleLeave,
}

func handleLeave(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	if err := ctx.Player.Stop(ctx.Context, ctx.GuildID); err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to stop playback: %v", err))
		return
	}
	if err := event.Client().UpdateVoiceState(ctx.Context, ctx.GuildID, nil, false, false); err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to leave voice: %v", err))
		return
	}

	commandrouter.RespondError(event, "Stopped playback, cleared the queue, and left voice.")
}
