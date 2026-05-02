package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

var RemoveSlice = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "removeslice",
		Description: "Remove a range of tracks from the queue",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{
				Name:        "from",
				Description: "The starting queue position to remove from",
				Required:    true,
			},
			discord.ApplicationCommandOptionInt{
				Name:        "to",
				Description: "The ending queue position to remove to (inclusive, defaults to end of queue)",
				Required:    false,
			},
		},
	},
	Handle: handleRemoveSlice,
}

func handleRemoveSlice(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	data := event.SlashCommandInteractionData()
	from := data.Int("from")

	to, hasTo := data.OptInt("to")

	if !hasTo {
		snapshot := ctx.Player.Queue(ctx.GuildID)
		to = len(snapshot.Queued)
	}

	removed, err := ctx.Player.RemoveSlice(ctx.GuildID, from, to)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to remove tracks: %v", err))
		return
	}

	if len(removed) == 1 {
		commandrouter.RespondError(event, fmt.Sprintf("Removed 1 track from the queue (position %d).", from))
	} else {
		commandrouter.RespondError(event, fmt.Sprintf("Removed %d tracks from the queue (positions %d–%d).", len(removed), from, from+len(removed)-1))
	}
}
