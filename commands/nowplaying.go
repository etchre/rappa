package commands

import (
	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

var NowPlaying = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "nowplaying",
		Description: "Show the currently playing track",
	},
	Handle: handleNowPlaying,
}

func handleNowPlaying(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	snapshot := ctx.Player.Queue(ctx.GuildID)
	if snapshot.Current == nil {
		commandrouter.RespondError(event, "Nothing is playing.")
		return
	}

	embed := helpers.NowPlayingEmbed(*snapshot.Current, nil, snapshot.Position, snapshot.Volume, "")
	if err := event.CreateMessage(discord.NewMessageCreate().WithEmbeds(embed)); err != nil {
		fmt.Fprintf(os.Stderr, "now playing response failed: %v\n", err)
	}
}
