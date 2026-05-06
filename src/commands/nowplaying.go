package commands

import (
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"rappa/commandrouter"
	"rappa/commands/helpers"
)

var NowPlaying = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "nowplaying",
		Description: "Show the currently playing track",
	},
	Handle: handleNowPlaying,
}

func handleNowPlaying(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	snapshot := ctx.Player.Queue(ctx.GuildID)
	if snapshot.Current == nil {
		commandrouter.RespondError(event, "Nothing is playing.")
		return
	}

	embed := helpers.NowPlayingEmbed(*snapshot.Current, nil, snapshot.Position, snapshot.Volume, "")
	if err := event.CreateMessage(discord.NewMessageCreate().WithEmbeds(embed)); err != nil {
		slog.Error("now playing response failed", "err", err)
	}
}
