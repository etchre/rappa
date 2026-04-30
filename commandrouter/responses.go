package commandrouter

import (
	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func RespondError(event *events.ApplicationCommandInteractionCreate, message string) {
	if err := event.CreateMessage(discord.NewMessageCreate().WithContent(message)); err != nil {
		fmt.Fprintf(os.Stderr, "command error response failed: %v\n", err)
	}
}

func UpdateResponse(event *events.ApplicationCommandInteractionCreate, message string) {
	if _, err := event.Client().Rest.UpdateInteractionResponse(
		event.ApplicationID(),
		event.Token(),
		discord.NewMessageUpdate().WithContent(message),
	); err != nil {
		fmt.Fprintf(os.Stderr, "update command response failed: %v\n", err)
	}
}
