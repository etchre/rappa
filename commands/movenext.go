package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

var MoveNext = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "movenext",
		Description: "Move a queued track to the front of the queue",
		Options: []discord.ApplicationCommandOption{
			queueNumberOption("The queued track number to move next"),
		},
	},
	Handle: handleMoveNext,
}

func handleMoveNext(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	queueNumber, err := parseQueueNumber(event.SlashCommandInteractionData())
	if err != nil {
		commandrouter.RespondError(event, err.Error())
		return
	}

	track, err := ctx.Player.MoveNext(ctx.GuildID, queueNumber)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to move track: %v", err))
		return
	}

	commandrouter.RespondError(event, fmt.Sprintf("Moved to queue position #1: %s", trackTitle(track)))
}
