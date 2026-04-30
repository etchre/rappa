package helpers

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

const queueComponentPrefix = "queue:"

func HandleComponent(ctx commandrouter.Context, event *events.ComponentInteractionCreate) {
	data, ok := event.Data.(discord.ButtonInteractionData)
	if !ok {
		return
	}

	customID := data.CustomID()
	if !strings.HasPrefix(customID, queueComponentPrefix) {
		return
	}

	handleQueuePageComponent(ctx, event, customID)
}

func QueuePageComponents(page int, queued int) []discord.LayoutComponent {
	pageCount := QueuePageCount(queued)
	if pageCount <= 1 {
		return nil
	}

	page = ClampQueuePage(page, queued)
	previous := discord.NewSecondaryButton("Previous", queueComponentID(page-1)).WithDisabled(page == 0)
	next := discord.NewSecondaryButton("Next", queueComponentID(page+1)).WithDisabled(page >= pageCount-1)

	return []discord.LayoutComponent{
		discord.NewActionRow(previous, next),
	}
}

func handleQueuePageComponent(ctx commandrouter.Context, event *events.ComponentInteractionCreate, customID string) {
	page, err := strconv.Atoi(strings.TrimPrefix(customID, queueComponentPrefix))
	if err != nil {
		respondComponentError(event, "That queue button is invalid.")
		return
	}

	if ctx.Player == nil {
		respondComponentError(event, "Music player is not ready yet.")
		return
	}

	snapshot := ctx.Player.Queue(ctx.GuildID)
	if snapshot.Current == nil && len(snapshot.Queued) == 0 {
		if err := event.UpdateMessage(
			discord.NewMessageUpdate().
				WithContent("The queue is empty.").
				ClearEmbeds().
				WithComponents(),
		); err != nil {
			fmt.Fprintf(os.Stderr, "empty queue page update failed: %v\n", err)
		}
		return
	}
	if snapshot.Current == nil {
		if err := event.UpdateMessage(
			discord.NewMessageUpdate().
				WithContent(FormatQueue(nil, snapshot.Queued)).
				ClearEmbeds().
				WithComponents(),
		); err != nil {
			fmt.Fprintf(os.Stderr, "queue page text update failed: %v\n", err)
		}
		return
	}

	page = ClampQueuePage(page, len(snapshot.Queued))
	components := QueuePageComponents(page, len(snapshot.Queued))
	if err := event.UpdateMessage(
		discord.NewMessageUpdate().
			ClearContent().
			WithEmbeds(QueueEmbed(*snapshot.Current, snapshot.Queued, page)).
			WithComponents(components...),
	); err != nil {
		fmt.Fprintf(os.Stderr, "queue page update failed: %v\n", err)
	}
}

func queueComponentID(page int) string {
	return fmt.Sprintf("%s%d", queueComponentPrefix, page)
}

func respondComponentError(event *events.ComponentInteractionCreate, message string) {
	if err := event.CreateMessage(discord.NewMessageCreate().WithContent(message).WithEphemeral(true)); err != nil {
		fmt.Fprintf(os.Stderr, "component error response failed: %v\n", err)
	}
}
