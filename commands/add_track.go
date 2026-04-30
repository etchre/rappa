package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/music"
)

type addMode int

const (
	addLast addMode = iota
	addNext
	playNow
)

func playQuery(data discord.SlashCommandInteractionData) string {
	if query, ok := data.OptString("query"); ok {
		return query
	}

	return data.String("link")
}

func handleAddTrack(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate, link string, mode addMode) {
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

	time.Sleep(2 * time.Second)

	var result music.QueueResult
	switch mode {
	case playNow:
		result, err = ctx.Player.PlayNow(ctx.Context, ctx.GuildID, link)
	case addNext:
		result, err = ctx.Player.AddNext(ctx.Context, ctx.GuildID, link)
	default:
		result, err = ctx.Player.Add(ctx.Context, ctx.GuildID, link)
	}
	if err != nil {
		commandrouter.UpdateResponse(event, fmt.Sprintf("Failed to play link: %v", err))
		return
	}

	title := trackTitle(result.Track)
	if result.Queued {
		if result.Added > 1 {
			commandrouter.UpdateResponse(event, fmt.Sprintf("Queued %d tracks starting at #%d: %s", result.Added, result.Position, title))
			return
		}

		commandrouter.UpdateResponse(event, fmt.Sprintf("Queued #%d: %s", result.Position, title))
		return
	}

	if result.Added > 1 {
		commandrouter.UpdateResponse(event, fmt.Sprintf("Now playing: %s\nQueued %d more track(s).", title, result.Added-1))
		return
	}

	commandrouter.UpdateResponse(event, fmt.Sprintf("Now playing: %s", title))
}
