package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

const diddyURL = "https://youtu.be/B_1KwX2M-Mc"

var Diddy = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "diddy",
		Description: "5 parties at diddys",
	},
	Handle: handleDiddy,
}

func handleDiddy(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	handleAddTrack(ctx, event, diddyURL, addLast)
}
