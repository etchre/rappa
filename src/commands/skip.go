package commands

import (
	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

var Skip = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "skip",
		Description: "Skip the current track",
	},
	Handle: handleSkip,
}

func handleSkip(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	result, err := ctx.Player.Skip(ctx.Context, ctx.GuildID)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to skip: %v", err))
		return
	}

	if result.Next != nil {
		snapshot := ctx.Player.Queue(ctx.GuildID)
		if snapshot.Current != nil {
			embed := helpers.NowPlayingEmbed(*snapshot.Current, nil, snapshot.Position, snapshot.Volume, "")
			if err := event.CreateMessage(discord.NewMessageCreate().WithContent("Skipped!").WithEmbeds(embed)); err != nil {
				fmt.Fprintf(os.Stderr, "skip response failed: %v\n", err)
			}
			return
		}

		commandrouter.RespondError(event, fmt.Sprintf("Skipped. Now playing: %s", helpers.TrackTitle(*result.Next)))
		return
	}
	if !result.Stopped {
		commandrouter.RespondError(event, "Nothing is playing and the queue is empty.")
		return
	}

	commandrouter.RespondError(event, "Skipped. Queue is empty, so playback stopped.")
}
