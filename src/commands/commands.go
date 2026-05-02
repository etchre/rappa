package commands

import (
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

func All(includeJokes bool) map[string]commandrouter.Command {
	commands := map[string]commandrouter.Command{
		"clear":        Clear,
		"leave":        Leave,
		"loop":         Loop,
		"move":         Move,
		"movenext":     MoveNext,
		"nowplaying":   NowPlaying,
		"pause":        Pause,
		"play":         Play,
		"playnext":     PlayNext,
		"playrightnow": PlayRightNow,
		"queue":        Queue,
		"remove":       Remove,
		"removeslice":  RemoveSlice,
		"restart":      Restart,
		"seek":         Seek,
		"setchannel":   SetChannel,
		"shuffle":      Shuffle,
		"shuffleall":   ShuffleAll,
		"skip":         Skip,
		"stop":         Stop,
		"unpause":      Unpause,
	}

	if includeJokes {
		commands["africa"] = Africa
		commands["diddy"] = Diddy
		commands["e"] = E
		commands["june"] = June
		commands["mae"] = Mae
		commands["yolk"] = Yolk
	}

	return commands
}

func HandleComponent(ctx commandrouter.Context, event *events.ComponentInteractionCreate) {
	helpers.HandleComponent(ctx, event)
}
