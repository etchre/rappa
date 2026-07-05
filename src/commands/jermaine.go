package commands

import (
	"math/rand/v2"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

const (
	jermaineHeadsURL = "https://youtu.be/Jx9aPbgSmAo"
	jermaineTailsURL = "https://youtu.be/KvMY1uzSC1E"
)

var Jermaine = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "jermaine",
		Description: "flip a coin",
	},
	Handle: handleJermaine,
}

func handleJermaine(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	url := jermaineHeadsURL
	if rand.IntN(2) == 0 {
		url = jermaineTailsURL
	}
	helpers.HandleAddTrack(ctx, event, url, helpers.AddLast)
}
