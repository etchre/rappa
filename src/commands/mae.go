package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

const fackURL = "https://www.youtube.com/watch?v=BOI8OGIy6cA"

var Mae = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "mae",
		Description: "oh fuck i think its stuck",
	},
	Handle: handleMae,
}

func handleMae(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	helpers.HandleAddTrack(ctx, event, fackURL, helpers.AddLast)
}
