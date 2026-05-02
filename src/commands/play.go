package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

var Play = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "play",
		Description: "Play a track from a link, or queue it if music is already playing",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:         "query",
				Description:  "A track link or YouTube search query",
				Required:     true,
				Autocomplete: true,
			},
			discord.ApplicationCommandOptionBool{
				Name:        "shuffle",
				Description: "Shuffle playlist or album tracks before queueing",
				Required:    false,
			},
			discord.ApplicationCommandOptionInt{
				Name:        "upto",
				Description: "Only queue the first N tracks from a playlist or album",
				Required:    false,
			},
		},
	},
	Handle: handlePlay,
}

func handlePlay(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	helpers.HandleAddTrack(ctx, event, helpers.PlayQuery(data), helpers.AddLast)
}
