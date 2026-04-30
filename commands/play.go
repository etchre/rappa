package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

var Play = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "play",
		Description: "Play a track from a link, or queue it if music is already playing",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "link",
				Description: "The track link to play",
				Required:    true,
			},
		},
	},
	Handle: handlePlay,
}

func handlePlay(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	handleAddTrack(ctx, event, data.String("link"), addLast)
}
