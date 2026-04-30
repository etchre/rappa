package commands

import "ytdlpPlayer/commandrouter"

var All = map[string]commandrouter.Command{
	"clear":    Clear,
	"diddy":    Diddy,
	"leave":    Leave,
	"movenext": MoveNext,
	"play":     Play,
	"playnext": PlayNext,
	"queue":    Queue,
	"remove":   Remove,
	"skip":     Skip,
	"stop":     Stop,
}
