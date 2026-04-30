package commands

import "ytdlpPlayer/commandrouter"

var All = map[string]commandrouter.Command{
	"clear":        Clear,
	"diddy":        Diddy,
	"leave":        Leave,
	"loop":         Loop,
	"movenext":     MoveNext,
	"play":         Play,
	"playnext":     PlayNext,
	"playrightnow": PlayRightNow,
	"queue":        Queue,
	"remove":       Remove,
	"restart":      Restart,
	"skip":         Skip,
	"stop":         Stop,
}
