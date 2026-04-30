package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

var Loop = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "loop",
		Description: "Toggle looping for the current track",
	},
	Handle: handleLoop,
}

func handleLoop(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	result, err := ctx.Player.ToggleLoop(ctx.GuildID)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to toggle loop: %v", err))
		return
	}

	status := "disabled"
	if result.Looping {
		status = "enabled"
	}
	commandrouter.RespondError(event, fmt.Sprintf("Loop %s for: %s", status, trackTitle(result.Track)))
}
