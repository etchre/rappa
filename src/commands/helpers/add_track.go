package helpers

import (
	"fmt"
	"os"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/music"
)

type AddMode int

const (
	AddLast AddMode = iota
	AddNext
	PlayNow
)

func PlayQuery(data discord.SlashCommandInteractionData) string {
	if query, ok := data.OptString("query"); ok {
		return query
	}

	return data.String("link")
}

func PlayShuffle(data discord.SlashCommandInteractionData) bool {
	return data.Bool("shuffle")
}

func HandleAddTrack(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate, link string, mode AddMode) {
	if ctx.Player == nil {
		commandrouter.RespondError(event, "Music player is not ready yet.")
		return
	}
	if link == "" {
		commandrouter.RespondError(event, "A link is required.")
		return
	}

	if err := event.DeferCreateMessage(false); err != nil {
		fmt.Fprintf(os.Stderr, "defer add track response failed: %v\n", err)
		return
	}

	voiceChannelID, err := commandrouter.CallerVoiceChannelID(event)
	if err != nil {
		commandrouter.UpdateResponse(event, err.Error())
		return
	}

	if err := event.Client().UpdateVoiceState(ctx.Context, ctx.GuildID, &voiceChannelID, false, true); err != nil {
		commandrouter.UpdateResponse(event, fmt.Sprintf("Failed to join voice: %v", err))
		return
	}
	if ctx.StatusChannels != nil {
		ctx.StatusChannels.SetFallbackIfUnset(ctx.GuildID, event.Channel().ID())
	}

	time.Sleep(2 * time.Second)

	var result music.QueueResult
	options := music.AddOptions{
		PremiumFallbackAllowed: ctx.PremiumFallbackAllowed(event.User().ID),
		RequesterName:          requesterName(event),
		RequesterID:            event.User().ID.String(),
		PremiumAllowedUserIDs:  ctx.PremiumAllowedUserIDs,
		Shuffle:                PlayShuffle(event.SlashCommandInteractionData()),
	}
	switch mode {
	case PlayNow:
		result, err = ctx.Player.PlayNow(ctx.Context, ctx.GuildID, link, options)
	case AddNext:
		result, err = ctx.Player.AddNext(ctx.Context, ctx.GuildID, link, options)
	default:
		result, err = ctx.Player.Add(ctx.Context, ctx.GuildID, link, options)
	}
	if err != nil {
		commandrouter.UpdateResponse(event, fmt.Sprintf("Failed to play link: %v", err))
		return
	}

	title := TrackTitle(result.Track)
	if result.Queued {
		content, embed := QueuedEmbed(result, event.User().Mention())
		commandrouter.UpdateResponseContentEmbed(event, content, embed)
		return
	}

	snapshot := ctx.Player.Queue(ctx.GuildID)
	if snapshot.Current != nil {
		commandrouter.UpdateResponseEmbed(event, NowPlayingEmbed(*snapshot.Current, snapshot.Queued, snapshot.Position, snapshot.Volume, event.User().Mention()))
		return
	}

	if result.Added > 1 {
		commandrouter.UpdateResponse(event, fmt.Sprintf("Now playing: %s\nQueued %d more track(s).", title, result.Added-1))
		return
	}

	commandrouter.UpdateResponse(event, fmt.Sprintf("Now playing: %s", title))
}

func requesterName(event *events.ApplicationCommandInteractionCreate) string {
	if member := event.Member(); member != nil {
		return member.EffectiveName()
	}

	return event.User().EffectiveName()
}
