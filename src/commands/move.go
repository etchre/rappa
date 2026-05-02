package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

var Move = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "move",
		Description: "Move a song to a different position in the queue",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{
				Name:        "from",
				Description: "The queue position of the song to move",
				Required:    true,
			},
			discord.ApplicationCommandOptionInt{
				Name:        "to",
				Description: "The queue position to move it in front of",
				Required:    true,
			},
		},
	},
	Handle: handleMove,
}

func handleMove(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	data := event.SlashCommandInteractionData()
	from := data.Int("from")
	to := data.Int("to")

	track, err := ctx.Player.Move(ctx.GuildID, from, to)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to move track: %v", err))
		return
	}

	commandrouter.RespondError(event, fmt.Sprintf("Moved %s to position %d.", helpers.TrackTitle(track), to))
}
