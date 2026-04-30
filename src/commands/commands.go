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
	"movenext":     MoveNext,
	"nowplaying":   NowPlaying,
	"play":         Play,
	"playnext":     PlayNext,
	"playrightnow": PlayRightNow,
	"queue":        Queue,
	"remove":       Remove,
	"restart":      Restart,
	"skip":         Skip,
	"stop":         Stop,
}

func HandleComponent(ctx commandrouter.Context, event *events.ComponentInteractionCreate) {
	helpers.HandleComponent(ctx, event)
}
