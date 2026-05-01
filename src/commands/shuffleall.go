package commands

import (
	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

var ShuffleAll = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "shuffleall",
		Description: "Shuffle the current track together with the queue",
	},
	Handle: handleShuffleAll,
}

func handleShuffleAll(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	_, count, err := ctx.Player.ShuffleAll(ctx.Context, ctx.GuildID)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to shuffle all: %v", err))
		return
	}

	snapshot := ctx.Player.Queue(ctx.GuildID)
	if snapshot.Current != nil {
		embed := helpers.NowPlayingEmbed(*snapshot.Current, snapshot.Queued, snapshot.Position, snapshot.Volume, "")
		if err := event.CreateMessage(discord.NewMessageCreate().WithContent(fmt.Sprintf("Shuffled current track with %d queued tracks.", count)).WithEmbeds(embed)); err != nil {
			fmt.Fprintf(os.Stderr, "shuffle all response failed: %v\n", err)
		}
		return
	}

	if err := event.CreateMessage(discord.NewMessageCreate().WithContent(fmt.Sprintf("Shuffled current track with %d queued tracks.", count))); err != nil {
		fmt.Fprintf(os.Stderr, "shuffle all response failed: %v\n", err)
	}
}
