package commands

import (
	"fmt"
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"rappa/commandrouter"
)

var Pause = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "pause",
		Description: "Toggle playback pause",
	},
	Handle: handlePause,
}

func handlePause(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	result, err := ctx.Player.TogglePause(ctx.Context, ctx.GuildID)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to pause playback: %v", err))
		return
	}

	content := "Paused playback."
	if !result.Paused {
		content = "Resumed playback."
	}
	if err := event.CreateMessage(discord.NewMessageCreate().WithContent(content)); err != nil {
		slog.Error("pause response failed", "err", err)
	}
}
