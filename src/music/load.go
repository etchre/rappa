package music

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"

	"rappa/utils"
)

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

func (p *Player) Search(ctx context.Context, query string, limit int) ([]lavalink.Track, error) {
	query = strings.TrimSpace(query)
	if query == "" || utils.IsURL(query) {
		return nil, nil
	}

	node := p.node()
	if node == nil {
		return nil, fmt.Errorf("no lavalink node is available")
	}

	result, err := node.LoadTracks(ctx, utils.YouTubeSearchPrefix+query)
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

type probedLoad struct {
	playableLoad
	UsedPremiumRoute bool
}

func loadWithProbe(ctx context.Context, node disgolink.Node, identifier string) (probedLoad, error) {
	identifier = strings.TrimSpace(identifier)

	if utils.YouTubeVideoID(identifier) == "" {
		loaded, err := loadPlayableTracks(ctx, node, identifier)
		return probedLoad{playableLoad: loaded}, err
	}

	slog.Debug("probing track", "id", identifier)
	result, err := utils.ProbeIdentifier(ctx, identifier)
	if err != nil {
		slog.Error("yt-dlp probe failed", "id", identifier, "err", err)
	}

	switch result {
	case utils.ProbeResultPremium:
		slog.Debug("probe result: premium, loading metadata", "id", identifier)
		metaLoaded, metaErr := loadPlayableTracks(ctx, node, identifier)

		slog.Debug("loading via premium plugin", "id", identifier)
		loaded, err := loadPlayableTracks(ctx, node, premiumPlayableIdentifier(identifier))
		if err != nil {
			return probedLoad{}, err
		}

		if metaErr == nil && len(metaLoaded.Tracks) > 0 && len(loaded.Tracks) > 0 {
			for i := range loaded.Tracks {
				loaded.Tracks[i].Info = metaLoaded.Tracks[0].Info
			}
			slog.Debug("copied metadata to premium track", "track", TrackTitle(loaded.Tracks[0]))
		}

		return probedLoad{playableLoad: loaded, UsedPremiumRoute: true}, nil

	case utils.ProbeResultUnavailable:
		slog.Debug("probe result: unavailable, trying resolver", "id", identifier)
		resolved := utils.ResolvedYouTubeIdentifier(ctx, identifier)
		if resolved == identifier {
			slog.Warn("resolver returned same identifier", "id", identifier)
			return probedLoad{}, fmt.Errorf("resolver returned same unavailable identifier %q", identifier)
		}
		slog.Debug("resolved, loading via lavalink", "id", identifier, "resolved", resolved)
		loaded, err := loadPlayableTracks(ctx, node, resolved)
		return probedLoad{playableLoad: loaded}, err

	default:
		slog.Debug("probe result: available, loading", "id", identifier)
		loaded, err := loadPlayableTracks(ctx, node, identifier)
		return probedLoad{playableLoad: loaded}, err
	}
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
		return input
	}

	return playableIdentifier(input)
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
	if utils.IsURL(input) || hasSearchPrefix(input) {
		return input
	}

	return utils.YouTubeSearchPrefix + input
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

func unresolvedTrackIdentifier(track lavalink.Track) string {
	for _, candidate := range []string{originalTrackIdentifier(track), track.Info.Identifier} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if musicURL := utils.YouTubeMusicTrackURL(candidate); musicURL != "" {
			return musicURL
		}
		return candidate
	}

	return ""
}

func resolvedTrackIdentifier(ctx context.Context, track lavalink.Track) string {
	for _, candidate := range []string{originalTrackIdentifier(track), track.Info.Identifier} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}

		if videoID := utils.YouTubeVideoID(candidate); videoID != "" {
			return utils.ResolvedYouTubeIdentifier(ctx, candidate)
		}
		if utils.IsYouTubeVideoID(candidate) {
			resolvedID, err := utils.ResolveYouTubeVideoID(ctx, candidate)
			if err != nil {
				slog.Error("youtube music id resolver failed", "id", candidate, "err", err)
				return utils.YouTubeMusicTrackURL(candidate)
			}
			return utils.YouTubeMusicTrackURL(resolvedID)
		}
		if musicURL := utils.YouTubeMusicTrackURL(candidate); musicURL != "" {
			return musicURL
		}
	}

	return ""
}

func recoveryQuery(track lavalink.Track) string {
	author := utils.TopicSuffixPattern.ReplaceAllString(strings.TrimSpace(track.Info.Author), "")
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
