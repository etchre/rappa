package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

var Restart = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "restart",
		Description: "Restart the current track from the beginning",
	},
	Handle: handleRestart,
}

func handleRestart(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	track, err := ctx.Player.Restart(ctx.Context, ctx.GuildID)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to restart track: %v", err))
		return
	}

	commandrouter.RespondError(event, fmt.Sprintf("Restarted: %s", trackTitle(track)))
}
