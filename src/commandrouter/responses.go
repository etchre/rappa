package commandrouter

import (
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func RespondError(event *events.ApplicationCommandInteractionCreate, message string) {
	if err := event.CreateMessage(discord.NewMessageCreate().WithContent(message)); err != nil {
		slog.Error("command error response failed", "err", err)
	}
}

func UpdateResponse(event *events.ApplicationCommandInteractionCreate, message string) {
	if _, err := event.Client().Rest.UpdateInteractionResponse(
		event.ApplicationID(),
		event.Token(),
		discord.NewMessageUpdate().WithContent(message),
	); err != nil {
		slog.Error("update command response failed", "err", err)
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
		slog.Error("update command embed response failed", "err", err)
	}
}
