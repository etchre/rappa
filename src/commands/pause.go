package commands

import (
	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

var Pause = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "pause",
		Description: "Toggle playback pause",
	},
	Handle: handlePause,
}

func handlePause(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

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
		fmt.Fprintf(os.Stderr, "pause response failed: %v\n", err)
	}
}
