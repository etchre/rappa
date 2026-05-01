package commands

import (
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

var All = map[string]commandrouter.Command{
	"clear":        Clear,
	"diddy":        Diddy,
	"leave":        Leave,
	"loop":         Loop,
	"mae":          Mae,
	"movenext":     MoveNext,
	"nowplaying":   NowPlaying,
	"pause":        Pause,
	"play":         Play,
	"playnext":     PlayNext,
	"playrightnow": PlayRightNow,
	"queue":        Queue,
	"remove":       Remove,
	"restart":      Restart,
	"setchannel":   SetChannel,
	"shuffle":      Shuffle,
	"shuffleall":   ShuffleAll,
	"skip":         Skip,
	"stop":         Stop,
	"unpause":      Unpause,
}

func HandleComponent(ctx commandrouter.Context, event *events.ComponentInteractionCreate) {
	helpers.HandleComponent(ctx, event)
}
