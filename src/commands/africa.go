package commands

import (
	"math/rand/v2"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

const (
	africaTotoURL   = "https://www.youtube.com/watch?v=QAo_Ycocl1E"
	africaWeezerURL = "https://www.youtube.com/watch?v=QaKcTVP8IJs"
)

var Africa = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "africa",
		Description: "are you feeling lucky?",
	},
	Handle: handleAfrica,
}

func handleAfrica(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	url := africaTotoURL
	if rand.IntN(6) == 0 {
		url = africaWeezerURL
	}
	helpers.HandleAddTrack(ctx, event, url, helpers.AddLast)
}
