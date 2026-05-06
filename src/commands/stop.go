package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"rappa/commandrouter"
)

var Stop = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "stop",
		Description: "Stop playback and clear the queue without leaving voice",
	},
	Handle: handleStop,
}

func handleStop(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if err := ctx.Player.Stop(ctx.Context, ctx.GuildID); err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to stop playback: %v", err))
		return
	}

	commandrouter.RespondError(event, "Stopped playback and cleared the queue.")
}
