package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

var PlayNext = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "playnext",
		Description: "Play a link next, ahead of the rest of the queue",
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
		},
	},
	Handle: handlePlayNext,
}

func handlePlayNext(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	helpers.HandleAddTrack(ctx, event, helpers.PlayQuery(data), helpers.AddNext)
}
