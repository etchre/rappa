package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"rappa/commandrouter"
	"rappa/commands/helpers"
	"rappa/music"
)

var Loop = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "loop",
		Description: "Toggle looping for the current track",
	},
	Handle: handleLoop,
}

func handleLoop(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	result, err := ctx.Player.ToggleLoop(ctx.GuildID)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to toggle loop: %v", err))
		return
	}

	status := "disabled"
	if result.Looping {
		status = "enabled"
	}
	commandrouter.RespondError(event, fmt.Sprintf("Loop %s for: %s", status, music.TrackTitle(result.Track)))
}
