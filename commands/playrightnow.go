package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

var PlayRightNow = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "playrightnow",
		Description: "Replace the current track immediately while keeping the queue intact",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "query",
				Description: "A track link or YouTube search query",
				Required:    true,
			},
		},
	},
	Handle: handlePlayRightNow,
}

func handlePlayRightNow(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	handleAddTrack(ctx, event, playQuery(data), playNow)
}
