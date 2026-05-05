package music

import (
	"github.com/disgoorg/disgolink/v3/lavalink"
)

func TrackTitle(track lavalink.Track) string {
	if track.Info.Author == "" {
		return track.Info.Title
	}

	return track.Info.Author + " - " + track.Info.Title
}
