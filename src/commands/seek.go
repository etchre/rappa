package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgolink/v3/lavalink"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

var Seek = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "seek",
		Description: "Seek forward or backward in the current track",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{
				Name:        "secs",
				Description: "Seconds to seek (negative to rewind)",
				Required:    true,
			},
		},
	},
	Handle: handleSeek,
}

func handleSeek(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	secs := event.SlashCommandInteractionData().Int("secs")
	offsetMs := lavalink.Duration(secs) * 1000

	result, err := ctx.Player.Seek(ctx.Context, ctx.GuildID, offsetMs)
	if err != nil {
		commandrouter.RespondError(event, fmt.Sprintf("Failed to seek: %v", err))
		return
	}

	direction := "forward"
	if secs < 0 {
		direction = "backward"
		secs = -secs
	}

	commandrouter.RespondError(event, fmt.Sprintf("Seeked %s %ds in %s [%s/%s]",
		direction, secs, helpers.TrackTitle(result.Track),
		helpers.FormatDuration(result.Position), helpers.FormatDuration(result.Track.Info.Length)))
}
