package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

const juneURL = "https://www.youtube.com/watch?v=gaQvOVOZoTQ"

var June = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "june",
		Description: "what is this jit doing on the calculator?",
	},
	Handle: handleJune,
}

func handleJune(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	helpers.HandleAddTrack(ctx, event, juneURL, helpers.AddLast)
}
