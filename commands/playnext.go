package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

var PlayNext = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "playnext",
		Description: "Play a link next, ahead of the rest of the queue",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "link",
				Description: "The track link to play next",
				Required:    true,
			},
		},
	},
	Handle: handlePlayNext,
}

func handlePlayNext(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	handleAddTrack(ctx, event, data.String("link"), addNext)
}
