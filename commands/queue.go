package commands

import (
	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

var Queue = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "queue",
		Description: "Show the current music queue",
	},
	Handle: handleQueue,
}

func handleQueue(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	snapshot := ctx.Player.Queue(ctx.GuildID)
	if snapshot.Current == nil && len(snapshot.Queued) == 0 {
		commandrouter.RespondError(event, "The queue is empty.")
		return
	}
	if snapshot.Current == nil {
		if err := event.CreateMessage(discord.NewMessageCreate().WithContent(helpers.FormatQueue(nil, snapshot.Queued))); err != nil {
			fmt.Fprintf(os.Stderr, "queue response failed: %v\n", err)
		}
		return
	}

	components := helpers.QueuePageComponents(0, len(snapshot.Queued))
	message := discord.NewMessageCreate().WithEmbeds(helpers.QueueEmbed(*snapshot.Current, snapshot.Queued, 0))
	if len(components) > 0 {
		message = message.WithComponents(components...)
	}

	if err := event.CreateMessage(message); err != nil {
		fmt.Fprintf(os.Stderr, "queue response failed: %v\n", err)
	}
}
