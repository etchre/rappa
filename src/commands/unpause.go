package commands

import (
	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

var Unpause = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "unpause",
		Description: "Resume playback if paused",
	},
	Handle: handleUnpause,
}

func handleUnpause(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	result, err := ctx.Player.Unpause(ctx.Context, ctx.GuildID)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to resume playback: %v", err))
		return
	}
	if !result.Changed {
		if err := event.CreateMessage(discord.NewMessageCreate().WithContent("Playback is already running.")); err != nil {
			fmt.Fprintf(os.Stderr, "unpause response failed: %v\n", err)
		}
		return
	}

	if err := event.CreateMessage(discord.NewMessageCreate().WithContent("Resumed playback.")); err != nil {
		fmt.Fprintf(os.Stderr, "unpause response failed: %v\n", err)
	}
}
