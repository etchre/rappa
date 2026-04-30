package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

var Remove = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "remove",
		Description: "Remove a track from the queue",
		Options: []discord.ApplicationCommandOption{
			queueNumberOption("The queued track number to remove"),
		},
	},
	Handle: handleRemove,
}

func handleRemove(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	queueNumber, err := parseQueueNumber(event.SlashCommandInteractionData())
	if err != nil {
		commandrouter.RespondError(event, err.Error())
		return
	}

	track, err := ctx.Player.Remove(ctx.GuildID, queueNumber)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to remove track: %v", err))
		return
	}

	commandrouter.RespondError(event, fmt.Sprintf("Removed from queue: %s", trackTitle(track)))
}
