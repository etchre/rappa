package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgolink/v3/lavalink"

	"ytdlpPlayer/music"
)

func queuedEmbed(result music.QueueResult, requester string) (string, discord.Embed) {
	if result.Added > 1 {
		return collectionQueuedContent(result.CollectionKind), collectionQueuedEmbed(result, requester)
	}

	return "Song queued!", songQueuedEmbed(result.Track, requester)
}

func songQueuedEmbed(track lavalink.Track, requester string) discord.Embed {
	embed := discord.Embed{}.
		WithColor(musicEmbedColor).
		WithTitle(track.Info.Title).
		WithDescription(fmt.Sprintf("Duration: `%s`\nArtist: %s", formatDuration(track.Info.Length), track.Info.Author)).
		AddField("Requested by", requester, true)

	if track.Info.URI != nil {
		embed = embed.WithURL(*track.Info.URI)
	}
	if track.Info.ArtworkURL != nil {
		embed = embed.WithThumbnail(*track.Info.ArtworkURL)
	}

	return embed
}

func collectionQueuedEmbed(result music.QueueResult, requester string) discord.Embed {
	title := result.CollectionName
	if title == "" {
		title = trackTitle(result.Track)
	}

	embed := discord.Embed{}.
		WithColor(musicEmbedColor).
		WithTitle(title).
		WithDescription(fmt.Sprintf("Duration: `%s`", formatDuration(totalLength(result.Tracks)))).
		AddField("Tracks", collectionTrackPreview(result.Tracks), false).
		AddField("Requested by", requester, true)

	if result.Track.Info.URI != nil {
		embed = embed.WithURL(*result.Track.Info.URI)
	}
	if result.Track.Info.ArtworkURL != nil {
		embed = embed.WithThumbnail(*result.Track.Info.ArtworkURL)
	}

	return embed
}

func collectionTrackPreview(tracks []lavalink.Track) string {
	limit := 10
	if len(tracks) < limit {
		limit = len(tracks)
	}

	preview := numberedTracksFrom(tracks[:limit], 1)
	if len(tracks) > limit {
		preview += fmt.Sprintf("+%d more tracks...", len(tracks)-limit)
	}

	return preview
}

func collectionQueuedContent(kind string) string {
	if kind == "album" {
		return "Album queued!"
	}

	return "Playlist queued!"
}
