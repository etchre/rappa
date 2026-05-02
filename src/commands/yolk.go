package commands

import (
	"fmt"
	"math/rand/v2"
	"os"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
	"ytdlpPlayer/music"
)

var yolkURLs = []string{
	"https://youtu.be/-arkKErUdLs?si=vQtWUsZUaiX_o9Cs", // ethan
}

var Yolk = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "yolk",
		Description: "peer into the minds of the finest yolk",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionBool{
				Name:        "all",
				Description: "Queue all yolk in a random order",
				Required:    false,
			},
		},
	},
	Handle: handleYolk,
}

func handleYolk(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	if data.Bool("all") {
		handleYolkAll(ctx, event)
		return
	}

	helpers.HandleAddTrack(ctx, event, yolkURLs[rand.IntN(len(yolkURLs))], helpers.AddLast)
}

func handleYolkAll(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}

	if err := event.DeferCreateMessage(false); err != nil {
		fmt.Fprintf(os.Stderr, "defer yolk response failed: %v\n", err)
		return
	}

	voiceChannelID, err := commandrouter.CallerVoiceChannelID(event)
	if err != nil {
		commandrouter.UpdateResponse(event, err.Error())
		return
	}
	ctx.NoteVoiceState(event.User().ID, voiceChannelID)

	if err := event.Client().UpdateVoiceState(ctx.Context, ctx.GuildID, &voiceChannelID, false, true); err != nil {
		commandrouter.UpdateResponse(event, fmt.Sprintf("Failed to join voice: %v", err))
		return
	}

	if err := ctx.Player.WaitUntilVoiceReady(ctx.Context, ctx.GuildID, 10*time.Second); err != nil {
		commandrouter.UpdateResponse(event, fmt.Sprintf("Failed to connect voice to Lavalink: %v", err))
		return
	}

	options := music.AddOptions{
		PremiumFallbackAllowed: ctx.PremiumFallbackAllowed(event.User().ID),
		RequesterName:          event.User().EffectiveName(),
		RequesterID:            event.User().ID.String(),
		PremiumAllowedUserIDs:  ctx.PremiumAllowedUserIDs,
	}

	added := 0
	for _, i := range rand.Perm(len(yolkURLs)) {
		_, err := ctx.Player.Add(ctx.Context, ctx.GuildID, yolkURLs[i], options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "yolk add failed for %s: %v\n", yolkURLs[i], err)
			continue
		}
		added++
	}

	if added == 0 {
		commandrouter.UpdateResponse(event, "Failed to queue any yolk songs.")
		return
	}

	commandrouter.UpdateResponse(event, fmt.Sprintf("Queued %d yolk song(s).", added))
}
