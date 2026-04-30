package music

import (
	"context"
	"fmt"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
)

func loadPlayableTrack(ctx context.Context, node disgolink.Node, identifier string) (lavalink.Track, error) {
	result, err := node.LoadTracks(ctx, identifier)
	if err != nil {
		return lavalink.Track{}, fmt.Errorf("load tracks: %w", err)
	}

	switch result.LoadType {
	case lavalink.LoadTypeTrack:
		track, ok := result.Data.(lavalink.Track)
		if !ok {
			return lavalink.Track{}, fmt.Errorf("unexpected track load result data: %T", result.Data)
		}
		return track, nil

	case lavalink.LoadTypeSearch:
		tracks, ok := result.Data.(lavalink.Search)
		if !ok {
			return lavalink.Track{}, fmt.Errorf("unexpected search load result data: %T", result.Data)
		}
		if len(tracks) == 0 {
			return lavalink.Track{}, fmt.Errorf("search returned no tracks")
		}
		return tracks[0], nil

	case lavalink.LoadTypePlaylist:
		playlist, ok := result.Data.(lavalink.Playlist)
		if !ok {
			return lavalink.Track{}, fmt.Errorf("unexpected playlist load result data: %T", result.Data)
		}
		if len(playlist.Tracks) == 0 {
			return lavalink.Track{}, fmt.Errorf("playlist returned no tracks")
		}
		return playlist.Tracks[0], nil

	case lavalink.LoadTypeEmpty:
		return lavalink.Track{}, fmt.Errorf("no tracks found for %q", identifier)

	case lavalink.LoadTypeError:
		if loadErr, ok := result.Data.(lavalink.Exception); ok {
			return lavalink.Track{}, fmt.Errorf("lavalink load error: %w", loadErr)
		}
		return lavalink.Track{}, fmt.Errorf("lavalink returned an unknown load error")

	default:
		return lavalink.Track{}, fmt.Errorf("unsupported lavalink load type %q", result.LoadType)
	}
}
