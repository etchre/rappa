package commands

import (
	"fmt"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgolink/v3/lavalink"
)

const musicEmbedColor = 0x2ECC71
const queuePageSize = 10

func nowPlayingEmbed(current lavalink.Track, queued []lavalink.Track, position lavalink.Duration, volume int, requester string) discord.Embed {
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
	embed = embed.AddField("Total length", formatDuration(totalLength(queued)), true)
	embed = embed.AddField("Page", "1 out of 1", true)

	return embed
}

func queueEmbed(current lavalink.Track, queued []lavalink.Track, page int) discord.Embed {
	page = clampQueuePage(page, len(queued))
	pageCount := queuePageCount(len(queued))

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
		AddField("Up next", numberedTracksFrom(queued[start:end], start+1), false).
		AddField("In queue", queueCount(len(queued)), true).
		AddField("Total length", formatDuration(totalLength(queued)), true).
		AddField("Page", fmt.Sprintf("%d out of %d", page+1, pageCount), true)
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
		return fmt.Sprintf("`%s`  %d%%", formatDuration(position), volume)
	}

	return fmt.Sprintf("`%s` `[%s/%s]` %d%%", progressBar(position, length), formatDuration(position), formatDuration(length), volume)
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

	return numberedTracksFrom(tracks[:limit], 1)
}

func numberedTracksFrom(tracks []lavalink.Track, firstNumber int) string {
	if len(tracks) == 0 {
		return "Nothing queued."
	}

	var builder strings.Builder
	for i, track := range tracks {
		builder.WriteString(fmt.Sprintf("`%d.` %s `[%s]`\n", firstNumber+i, trackTitle(track), formatDuration(track.Info.Length)))
	}

	return builder.String()
}

func queuePageCount(queued int) int {
	if queued == 0 {
		return 1
	}

	return (queued + queuePageSize - 1) / queuePageSize
}

func clampQueuePage(page int, queued int) int {
	pageCount := queuePageCount(queued)
	if page < 0 {
		return 0
	}
	if page >= pageCount {
		return pageCount - 1
	}

	return page
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

func formatDuration(duration lavalink.Duration) string {
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
