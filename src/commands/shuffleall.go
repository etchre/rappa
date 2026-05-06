package commands

import (
	"fmt"
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"rappa/commandrouter"
	"rappa/commands/helpers"
)

var ShuffleAll = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "shuffleall",
		Description: "Shuffle the current track together with the queue",
	},
	Handle: handleShuffleAll,
}

func handleShuffleAll(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	_, count, err := ctx.Player.ShuffleAll(ctx.Context, ctx.GuildID)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to shuffle all: %v", err))
		return
	}

	snapshot := ctx.Player.Queue(ctx.GuildID)
	if snapshot.Current != nil {
		embed := helpers.NowPlayingEmbed(*snapshot.Current, snapshot.Queued, snapshot.Position, snapshot.Volume, "")
		if err := event.CreateMessage(discord.NewMessageCreate().WithContent(fmt.Sprintf("Shuffled current track with %d queued tracks.", count)).WithEmbeds(embed)); err != nil {
			slog.Error("shuffle all response failed", "err", err)
		}
		return
	}

	if err := event.CreateMessage(discord.NewMessageCreate().WithContent(fmt.Sprintf("Shuffled current track with %d queued tracks.", count))); err != nil {
		slog.Error("shuffle all response failed", "err", err)
	}
}
