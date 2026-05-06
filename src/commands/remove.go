package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"rappa/commandrouter"
	"rappa/commands/helpers"
	"rappa/music"
)

var Remove = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "remove",
		Description: "Remove a track from the queue",
		Options: []discord.ApplicationCommandOption{
			helpers.QueueNumberOption("The queued track number to remove"),
		},
	},
	Handle: handleRemove,
}

func handleRemove(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	queueNumber, err := helpers.ParseQueueNumber(event.SlashCommandInteractionData())
	if err != nil {
		commandrouter.RespondError(event, err.Error())
		return
	}

	track, err := ctx.Player.Remove(ctx.GuildID, queueNumber)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to remove track: %v", err))
		return
	}

	commandrouter.RespondError(event, fmt.Sprintf("Removed from queue: %s", music.TrackTitle(track)))
}
