package helpers

import (
	"fmt"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgolink/v3/lavalink"
)

const musicEmbedColor = 0x2ECC71
const queuePageSize = 10

func NowPlayingEmbed(current lavalink.Track, queued []lavalink.Track, position lavalink.Duration, volume int, requester string) discord.Embed {
	embed := discord.Embed{}.
		WithTitle("Now Playing").
		WithColor(musicEmbedColor).
		WithDescription(trackLink(current))

	if requester != "" {
		embed = embed.AddField("Requested by", requester, true)
	}

	embed = embed.AddField("Progress", progressLine(current, position, volume), false)
	embed = embed.WithFooterText(fmt.Sprintf("Source: %s", current.Info.Author))
	if current.Info.ArtworkURL != nil {
		embed = embed.WithThumbnail(*current.Info.ArtworkURL)
	}

	if len(queued) == 0 {
		return embed
	}

	embed = embed.AddField("Up next", numberedTracks(queued, 3), false)
	embed = embed.AddField("In queue", queueCount(len(queued)), true)
	embed = embed.AddField("Total length", FormatDuration(totalLength(queued)), true)
	embed = embed.AddField("Page", "1 out of 1", true)

	return embed
}

func UnableToPlayEmbed(track lavalink.Track) discord.Embed {
	embed := discord.Embed{}.
		WithTitle("Unable to Play").
		WithColor(0xE74C3C).
		WithDescription(trackLink(track)).
		AddField("Artist", track.Info.Author, true)

	if track.Info.ArtworkURL != nil {
		embed = embed.WithThumbnail(*track.Info.ArtworkURL)
	}

	return embed
}

func QueueEmbed(current lavalink.Track, queued []lavalink.Track, page int) discord.Embed {
	page = ClampQueuePage(page, len(queued))
	pageCount := QueuePageCount(len(queued))

	embed := discord.Embed{}.
		WithTitle("Queue").
		WithColor(musicEmbedColor).
		AddField("Now playing", trackLink(current), false)

	if current.Info.ArtworkURL != nil {
		embed = embed.WithThumbnail(*current.Info.ArtworkURL)
	}

	if len(queued) == 0 {
		return embed.
			AddField("Up next", "Nothing queued.", false).
			AddField("In queue", "0 songs", true).
			AddField("Total length", "0:00", true).
			AddField("Page", "1 out of 1", true)
	}

	start := page * queuePageSize
	end := start + queuePageSize
	if end > len(queued) {
		end = len(queued)
	}

	return embed.
		AddField("Up next", NumberedTracksFrom(queued[start:end], start+1), false).
		AddField("In queue", queueCount(len(queued)), true).
		AddField("Total length", FormatDuration(totalLength(queued)), true).
		AddField("Page", fmt.Sprintf("%d out of %d", page+1, pageCount), true)
}

func FormatQueue(current *lavalink.Track, queued []lavalink.Track) string {
	var builder strings.Builder
	builder.WriteString("Queue:\n")

	if current != nil {
		builder.WriteString("**Now playing: ")
		builder.WriteString(TrackTitle(*current))
		builder.WriteString("**\n")
	}

	for i, track := range queued {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, TrackTitle(track)))
	}

	return builder.String()
}

func TrackTitle(track lavalink.Track) string {
	return fmt.Sprintf("%s - %s", track.Info.Author, track.Info.Title)
}

func NumberedTracksFrom(tracks []lavalink.Track, firstNumber int) string {
	if len(tracks) == 0 {
		return "Nothing queued."
	}

	var builder strings.Builder
	for i, track := range tracks {
		builder.WriteString(fmt.Sprintf("`%d.` %s `[%s]`\n", firstNumber+i, TrackTitle(track), FormatDuration(track.Info.Length)))
	}

	return builder.String()
}

func QueuePageCount(queued int) int {
	if queued == 0 {
		return 1
	}

	return (queued + queuePageSize - 1) / queuePageSize
}

func ClampQueuePage(page int, queued int) int {
	pageCount := QueuePageCount(queued)
	if page < 0 {
		return 0
	}
	if page >= pageCount {
		return pageCount - 1
	}

	return page
}

func FormatDuration(duration lavalink.Duration) string {
	if duration < 0 {
		duration = 0
	}

	totalSeconds := duration.Seconds()
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}

	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func trackLink(track lavalink.Track) string {
	title := track.Info.Title
	if track.Info.URI == nil || *track.Info.URI == "" {
		return fmt.Sprintf("**%s**", title)
	}

	return fmt.Sprintf("**[%s](%s)**", title, *track.Info.URI)
}

func progressLine(track lavalink.Track, position lavalink.Duration, volume int) string {
	if track.Info.IsStream {
		return fmt.Sprintf("`LIVE`  %d%%", volume)
	}

	length := track.Info.Length
	if length <= 0 {
		return fmt.Sprintf("`%s`  %d%%", FormatDuration(position), volume)
	}

	return fmt.Sprintf("`%s` `[%s/%s]` %d%%", progressBar(position, length), FormatDuration(position), FormatDuration(length), volume)
}

func progressBar(position lavalink.Duration, length lavalink.Duration) string {
	const width = 14
	if position < 0 {
		position = 0
	}
	if position > length {
		position = length
	}

	filled := 0
	if length > 0 {
		filled = int(position * width / length)
	}
	if filled >= width {
		filled = width - 1
	}

	return strings.Repeat("=", filled) + "o" + strings.Repeat("-", width-filled-1)
}

func numberedTracks(tracks []lavalink.Track, limit int) string {
	if len(tracks) < limit {
		limit = len(tracks)
	}

	return NumberedTracksFrom(tracks[:limit], 1)
}

func totalLength(tracks []lavalink.Track) lavalink.Duration {
	var total lavalink.Duration
	for _, track := range tracks {
		if !track.Info.IsStream {
			total += track.Info.Length
		}
	}

	return total
}

func queueCount(count int) string {
	if count == 1 {
		return "1 song"
	}

	return fmt.Sprintf("%d songs", count)
}
