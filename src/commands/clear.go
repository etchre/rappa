package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

var Clear = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "clear",
		Description: "Clear the queue while keeping the current track playing",
	},
	Handle: handleClear,
}

func handleClear(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	cleared := ctx.Player.ClearQueue(ctx.GuildID)
	commandrouter.RespondError(event, fmt.Sprintf("Cleared %d queued track(s).", cleared))
}
