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

func UpdateResponseEmbed(event *events.ApplicationCommandInteractionCreate, embed discord.Embed) {
	UpdateResponseContentEmbed(event, "", embed)
}

func UpdateResponseContentEmbed(event *events.ApplicationCommandInteractionCreate, content string, embed discord.Embed) {
	if _, err := event.Client().Rest.UpdateInteractionResponse(
		event.ApplicationID(),
		event.Token(),
		discord.NewMessageUpdate().WithContent(content).WithEmbeds(embed),
	); err != nil {
		fmt.Fprintf(os.Stderr, "update command embed response failed: %v\n", err)
	}
}
