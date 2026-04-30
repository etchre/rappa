package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

var Stop = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "stop",
		Description: "Stop playback and clear the queue without leaving voice",
	},
	Handle: handleStop,
}

func handleStop(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	if err := ctx.Player.Stop(ctx.Context, ctx.GuildID); err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to stop playback: %v", err))
		return
	}

	commandrouter.RespondError(event, "Stopped playback and cleared the queue.")
}
