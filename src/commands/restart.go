package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
	"ytdlpPlayer/music"
)

var Restart = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "restart",
		Description: "Restart the current track from the beginning",
	},
	Handle: handleRestart,
}

func handleRestart(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	track, err := ctx.Player.Restart(ctx.Context, ctx.GuildID)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to restart track: %v", err))
		return
	}

	commandrouter.RespondError(event, fmt.Sprintf("Restarted: %s", music.TrackTitle(track)))
}
