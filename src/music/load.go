package music

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
)

const youtubeSearchPrefix = "ytsearch:"
const premiumSourcePrefix = "rappa-premium:"
const (
	collectionKindAlbum    = "album"
	collectionKindPlaylist = "playlist"
)

type playableLoad struct {
	Tracks         []lavalink.Track
	CollectionName string
	CollectionKind string
}

type ytDLPPlaylist struct {
	Title   string       `json:"title"`
	Entries []ytDLPEntry `json:"entries"`
}

type ytDLPEntry struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	WebpageURL string `json:"webpage_url"`
	Title      string `json:"title"`
	Channel    string `json:"channel"`
	Uploader   string `json:"uploader"`
	Duration   int64  `json:"duration"`
}

func (p *Player) Search(ctx context.Context, query string, limit int) ([]lavalink.Track, error) {
	query = strings.TrimSpace(query)
	if query == "" || isURL(query) {
		return nil, nil
	}

	node := p.node()
	if node == nil {
		return nil, fmt.Errorf("no lavalink node is available")
	}

	result, err := node.LoadTracks(ctx, youtubeSearchPrefix+query)
	if err != nil {
		return nil, fmt.Errorf("search tracks: %w", err)
	}
	if result.LoadType != lavalink.LoadTypeSearch {
		return nil, nil
	}

	tracks, ok := result.Data.(lavalink.Search)
	if !ok {
		return nil, fmt.Errorf("unexpected search result data: %T", result.Data)
	}
	if len(tracks) > limit {
		tracks = tracks[:limit]
	}

	return tracks, nil
}

func loadPlayableTracks(ctx context.Context, node disgolink.Node, identifier string) (playableLoad, error) {
	originalIdentifier := identifier
	if isYouTubeMusicCollectionURL(originalIdentifier) && !strings.HasPrefix(originalIdentifier, premiumSourcePrefix) {
		if loaded, err := loadYouTubeMusicCollectionWithYTDLP(ctx, node, originalIdentifier); err == nil {
			return loaded, nil
		} else {
			fmt.Printf("yt-dlp collection expansion failed, falling back to Lavalink playlist loader: %v\n", err)
		}
	}

	identifier = playableIdentifierForNode(node, identifier)
	debugTrackLoad(node, originalIdentifier, identifier)

	result, err := node.LoadTracks(ctx, identifier)
	if err != nil {
		debugTrackLoadError(node, originalIdentifier, err)
		return playableLoad{}, fmt.Errorf("load tracks: %w", err)
	}

	switch result.LoadType {
	case lavalink.LoadTypeTrack:
		track, ok := result.Data.(lavalink.Track)
		if !ok {
			return playableLoad{}, fmt.Errorf("unexpected track load result data: %T", result.Data)
		}
		return playableLoad{Tracks: []lavalink.Track{track}}, nil

	case lavalink.LoadTypeSearch:
		tracks, ok := result.Data.(lavalink.Search)
		if !ok {
			return playableLoad{}, fmt.Errorf("unexpected search load result data: %T", result.Data)
		}
		if len(tracks) == 0 {
			return playableLoad{}, fmt.Errorf("search returned no tracks")
		}
		return playableLoad{Tracks: selectSearchResult(tracks)}, nil

	case lavalink.LoadTypePlaylist:
		playlist, ok := result.Data.(lavalink.Playlist)
		if !ok {
			return playableLoad{}, fmt.Errorf("unexpected playlist load result data: %T", result.Data)
		}
		if len(playlist.Tracks) == 0 {
			return playableLoad{}, fmt.Errorf("playlist returned no tracks")
		}
		return playableLoad{
			Tracks:         playlist.Tracks,
			CollectionName: playlist.Info.Name,
			CollectionKind: collectionKind(originalIdentifier),
		}, nil

	case lavalink.LoadTypeEmpty:
		debugTrackLoadEmpty(node, originalIdentifier, identifier)
		return playableLoad{}, fmt.Errorf("no tracks found for %q", identifier)

	case lavalink.LoadTypeError:
		if loadErr, ok := result.Data.(lavalink.Exception); ok {
			debugLavalinkException("load", nodeName(node), originalIdentifier, loadErr)
			return playableLoad{}, fmt.Errorf("lavalink load error: %w", loadErr)
		}
		return playableLoad{}, fmt.Errorf("lavalink returned an unknown load error")

	default:
		return playableLoad{}, fmt.Errorf("unsupported lavalink load type %q", result.LoadType)
	}
}

func loadYouTubeMusicCollectionWithYTDLP(ctx context.Context, node disgolink.Node, identifier string) (playableLoad, error) {
	playlist, err := expandYouTubeMusicCollection(ctx, identifier)
	if err != nil {
		return playableLoad{}, err
	}
	if len(playlist.Entries) == 0 {
		return playableLoad{}, fmt.Errorf("yt-dlp returned no playlist entries")
	}

	tracks := make([]lavalink.Track, 0, len(playlist.Entries))
	for _, entry := range playlist.Entries {
		entryIdentifier := entryPlayableIdentifier(entry)
		if entryIdentifier == "" {
			fmt.Printf("skipping yt-dlp playlist entry without a playable URL: title=%q id=%q\n", entry.Title, entry.ID)
			continue
		}

		loaded, err := loadPlayableTracks(ctx, node, entryIdentifier)
		if err != nil {
			fmt.Printf("skipping yt-dlp playlist entry load failure: title=%q identifier=%q err=%v\n", entry.Title, entryIdentifier, err)
			continue
		}
		if len(loaded.Tracks) == 0 {
			continue
		}
		tracks = append(tracks, loaded.Tracks[0])
	}
	if len(tracks) == 0 {
		return playableLoad{}, fmt.Errorf("yt-dlp playlist entries could not be loaded by Lavalink")
	}

	return playableLoad{
		Tracks:         tracks,
		CollectionName: collectionName(identifier, playlist.Title),
		CollectionKind: collectionKind(identifier),
	}, nil
}

func expandYouTubeMusicCollection(ctx context.Context, identifier string) (ytDLPPlaylist, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"yt-dlp",
		"--flat-playlist",
		"--dump-single-json",
		"--no-warnings",
		identifier,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if ctx.Err() != nil {
		return ytDLPPlaylist{}, fmt.Errorf("yt-dlp playlist expansion timed out")
	}
	if err != nil {
		return ytDLPPlaylist{}, fmt.Errorf("yt-dlp playlist expansion failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var playlist ytDLPPlaylist
	if err := json.Unmarshal(output, &playlist); err != nil {
		return ytDLPPlaylist{}, fmt.Errorf("parse yt-dlp playlist JSON: %w", err)
	}

	return playlist, nil
}

func entryPlayableIdentifier(entry ytDLPEntry) string {
	for _, candidate := range []string{entry.WebpageURL, entry.URL} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if isURL(candidate) {
			return candidate
		}
		if strings.Contains(candidate, "youtube.com/watch") || strings.Contains(candidate, "youtu.be/") {
			return candidate
		}
	}

	if entry.ID == "" {
		return ""
	}

	return "https://music.youtube.com/watch?v=" + entry.ID
}

func playableIdentifierForNode(node disgolink.Node, input string) string {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, premiumSourcePrefix) {
		return input
	}

	identifier := playableIdentifier(input)
	return identifier
}

func premiumPlayableIdentifier(input string) string {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, premiumSourcePrefix) {
		return input
	}

	return premiumSourcePrefix + playableIdentifier(input)
}

func playableIdentifier(input string) string {
	input = strings.TrimSpace(input)
	if isURL(input) || hasSearchPrefix(input) {
		return input
	}

	return youtubeSearchPrefix + input
}

func isURL(input string) bool {
	parsed, err := url.Parse(input)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

func isYouTubeMusicCollectionURL(input string) bool {
	parsed, err := url.Parse(strings.TrimSpace(input))
	if err != nil {
		return false
	}

	host := strings.ToLower(parsed.Host)
	if host != "music.youtube.com" {
		return false
	}

	query := parsed.Query()
	return query.Get("list") != "" && (parsed.Path == "/playlist" || parsed.Path == "/watch")
}

func hasSearchPrefix(input string) bool {
	lower := strings.ToLower(input)
	return strings.HasPrefix(lower, "ytsearch:") ||
		strings.HasPrefix(lower, "ytmsearch:") ||
		strings.HasPrefix(lower, "scsearch:")
}

func selectSearchResult(tracks lavalink.Search) []lavalink.Track {
	return []lavalink.Track{tracks[0]}
}

func originalTrackIdentifier(track lavalink.Track) string {
	if track.Info.URI != nil && *track.Info.URI != "" {
		return *track.Info.URI
	}

	return track.Info.Identifier
}

func collectionKind(identifier string) string {
	if strings.Contains(identifier, "OLAK5uy") {
		return collectionKindAlbum
	}

	return collectionKindPlaylist
}

func collectionName(identifier string, fallback string) string {
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}

	if collectionKind(identifier) == collectionKindAlbum {
		return "Album"
	}

	return "Playlist"
}
