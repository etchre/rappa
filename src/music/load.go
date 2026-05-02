package music

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
)

const youtubeSearchPrefix = "ytsearch:"
const premiumSourcePrefix = "rappa-premium:"
const youtubeResolverTimeout = 10 * time.Second

const (
	collectionKindAlbum    = "album"
	collectionKindPlaylist = "playlist"
)

var topicSuffixPattern = regexp.MustCompile(`(?i)\s*-\s*topic$`)

type playableLoad struct {
	Tracks         []lavalink.Track
	CollectionName string
	CollectionKind string
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
	identifier = playableIdentifierForNode(ctx, node, identifier)
	debugTrackLoad(node, originalIdentifier, identifier)

	result, err := node.LoadTracks(ctx, identifier)
	if err != nil {
		debugTrackLoadError(node, originalIdentifier, identifier, err)
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

func playableIdentifierForNode(ctx context.Context, node disgolink.Node, input string) string {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, premiumSourcePrefix) {
		identifier := strings.TrimPrefix(input, premiumSourcePrefix)
		return premiumSourcePrefix + resolvedYouTubeIdentifier(ctx, identifier)
	}

	identifier := playableIdentifier(resolvedYouTubeIdentifier(ctx, input))
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

func resolvedYouTubeIdentifier(ctx context.Context, input string) string {
	videoID := YouTubeVideoID(input)
	if videoID == "" {
		return input
	}

	resolvedID, err := resolveYouTubeVideoID(ctx, videoID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "youtube music id resolver failed for %s: %v\n", videoID, err)
		return input
	}
	if resolvedID == "" {
		return input
	}

	return YouTubeMusicTrackURL(resolvedID)
}

func resolveYouTubeVideoID(ctx context.Context, videoID string) (string, error) {
	resolverCtx, cancel := context.WithTimeout(ctx, youtubeResolverTimeout)
	defer cancel()

	cmd := exec.CommandContext(resolverCtx, pythonCommand(), resolverScriptPath(), videoID)
	output, err := cmd.CombinedOutput()
	if resolverCtx.Err() != nil {
		return "", fmt.Errorf("resolver timed out")
	}
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}

	resolvedID := strings.TrimSpace(string(output))
	if !isYouTubeVideoID(resolvedID) {
		return "", fmt.Errorf("resolver returned invalid video id %q", resolvedID)
	}

	return resolvedID, nil
}

func pythonCommand() string {
	if value := strings.TrimSpace(os.Getenv("YTMUSIC_RESOLVER_PYTHON")); value != "" {
		return value
	}
	return "python3"
}

func resolverScriptPath() string {
	if value := strings.TrimSpace(os.Getenv("YTMUSIC_RESOLVER_SCRIPT")); value != "" {
		return value
	}

	for _, candidate := range []string{
		"/app/ytmusic_yt_dlp_test.py",
		"ytmusic_yt_dlp_test.py",
		filepath.Join("..", "ytmusic_yt_dlp_test.py"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return "ytmusic_yt_dlp_test.py"
}

func isURL(input string) bool {
	parsed, err := url.Parse(input)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
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

func YouTubeVideoID(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return ""
	}

	parsed, err := url.Parse(identifier)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "youtu.be" {
		videoID := strings.Trim(strings.TrimPrefix(parsed.EscapedPath(), "/"), "/")
		if isYouTubeVideoID(videoID) {
			return videoID
		}
		return ""
	}

	if host != "youtube.com" && host != "www.youtube.com" && host != "music.youtube.com" {
		return ""
	}
	if parsed.Path != "/watch" || parsed.Query().Get("list") != "" {
		return ""
	}

	videoID := parsed.Query().Get("v")
	if isYouTubeVideoID(videoID) {
		return videoID
	}
	return ""
}

func resolvedTrackIdentifier(ctx context.Context, track lavalink.Track) string {
	for _, candidate := range []string{originalTrackIdentifier(track), track.Info.Identifier} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}

		if videoID := YouTubeVideoID(candidate); videoID != "" {
			return resolvedYouTubeIdentifier(ctx, candidate)
		}
		if isYouTubeVideoID(candidate) {
			resolvedID, err := resolveYouTubeVideoID(ctx, candidate)
			if err != nil {
				fmt.Fprintf(os.Stderr, "youtube music id resolver failed for %s: %v\n", candidate, err)
				return YouTubeMusicTrackURL(candidate)
			}
			return YouTubeMusicTrackURL(resolvedID)
		}
		if musicURL := YouTubeMusicTrackURL(candidate); musicURL != "" {
			return musicURL
		}
	}

	return ""
}

func YouTubeMusicTrackURL(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return ""
	}

	parsed, err := url.Parse(identifier)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		if isYouTubeVideoID(identifier) {
			return "https://music.youtube.com/watch?v=" + identifier
		}
		return ""
	}

	host := strings.ToLower(parsed.Hostname())
	if host != "youtube.com" && host != "www.youtube.com" && host != "music.youtube.com" && host != "youtu.be" {
		return ""
	}

	videoID := parsed.Query().Get("v")
	if host == "youtu.be" {
		videoID = strings.Trim(strings.TrimPrefix(parsed.EscapedPath(), "/"), "/")
	}
	if !isYouTubeVideoID(videoID) {
		return ""
	}

	musicURL := url.URL{
		Scheme: "https",
		Host:   "music.youtube.com",
		Path:   "/watch",
	}
	query := musicURL.Query()
	query.Set("v", videoID)
	musicURL.RawQuery = query.Encode()

	return musicURL.String()
}

func isYouTubeVideoID(identifier string) bool {
	if len(identifier) != 11 {
		return false
	}

	for _, r := range identifier {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			continue
		}
		return false
	}

	return true
}

func recoveryQuery(track lavalink.Track) string {
	author := topicSuffixPattern.ReplaceAllString(strings.TrimSpace(track.Info.Author), "")
	title := strings.TrimSpace(track.Info.Title)

	if author == "" {
		return title
	}
	if title == "" {
		return author
	}

	return author + " " + title
}

func collectionKind(identifier string) string {
	if strings.Contains(identifier, "OLAK5uy") {
		return collectionKindAlbum
	}

	return collectionKindPlaylist
}
