package music

import (
	"context"
	"fmt"
	"net/url"
	"strings"

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

func loadPlayableTracks(ctx context.Context, node disgolink.Node, identifier string) (playableLoad, error) {
	originalIdentifier := identifier
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
