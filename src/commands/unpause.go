package commands

import (
	"fmt"
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"rappa/commandrouter"
)

var Unpause = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "unpause",
		Description: "Resume playback if paused",
	},
	Handle: handleUnpause,
}

func handleUnpause(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	result, err := ctx.Player.Unpause(ctx.Context, ctx.GuildID)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to resume playback: %v", err))
		return
	}
	if !result.Changed {
		if err := event.CreateMessage(discord.NewMessageCreate().WithContent("Playback is already running.")); err != nil {
			slog.Error("unpause response failed", "err", err)
		}
		return
	}

	if err := event.CreateMessage(discord.NewMessageCreate().WithContent("Resumed playback.")); err != nil {
		slog.Error("unpause response failed", "err", err)
	}
}
