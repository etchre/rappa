package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

const eURL = "https://www.youtube.com/watch?v=hehD3ealCpA"

var E = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "e",
		Description: "\"Heaven sees as my people see; Heaven hears as my people hear\" - Mencius; \"We carry the Flame!\"",
	},
	Handle: handleE,
}

func handleE(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	helpers.HandleAddTrack(ctx, event, eURL, helpers.AddLast)
}
