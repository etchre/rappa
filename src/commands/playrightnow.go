package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
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
			discord.ApplicationCommandOptionBool{
				Name:        "shuffle",
				Description: "Shuffle playlist or album tracks before queueing",
				Required:    false,
			},
		},
	},
	Handle: handlePlayRightNow,
}

func handlePlayRightNow(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	helpers.HandleAddTrack(ctx, event, helpers.PlayQuery(data), helpers.PlayNow)
}
