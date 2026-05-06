package commands

import (
	"fmt"
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"rappa/commandrouter"
)

var Shuffle = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "shuffle",
		Description: "Shuffle the queued tracks",
	},
	Handle: handleShuffle,
}

func handleShuffle(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	count, err := ctx.Player.ShuffleQueue(ctx.GuildID)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to shuffle queue: %v", err))
		return
	}
	if count < 2 {
		commandrouter.RespondError(event, "Need at least 2 queued tracks to shuffle.")
		return
	}

	if err := event.CreateMessage(discord.NewMessageCreate().WithContent(fmt.Sprintf("Shuffled %d queued tracks.", count))); err != nil {
		slog.Error("shuffle response failed", "err", err)
	}
}
