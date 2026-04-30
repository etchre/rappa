package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgolink/v3/lavalink"

	"ytdlpPlayer/commandrouter"
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
		if err := event.CreateMessage(discord.NewMessageCreate().WithContent(formatQueue(nil, snapshot.Queued))); err != nil {
			fmt.Fprintf(os.Stderr, "queue response failed: %v\n", err)
		}
		return
	}

	components := queuePageComponents(0, len(snapshot.Queued))
	message := discord.NewMessageCreate().WithEmbeds(queueEmbed(*snapshot.Current, snapshot.Queued, 0))
	if len(components) > 0 {
		message = message.WithComponents(components...)
	}

	if err := event.CreateMessage(message); err != nil {
		fmt.Fprintf(os.Stderr, "queue response failed: %v\n", err)
	}
}

func formatQueue(current *lavalink.Track, queued []lavalink.Track) string {
	var builder strings.Builder
	builder.WriteString("Queue:\n")

	if current != nil {
		builder.WriteString("**Now playing: ")
		builder.WriteString(trackTitle(*current))
		builder.WriteString("**\n")
	}

	for i, track := range queued {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, trackTitle(track)))
	}

	return builder.String()
}

func trackTitle(track lavalink.Track) string {
	return fmt.Sprintf("%s - %s", track.Info.Author, track.Info.Title)
}
