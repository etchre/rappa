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

	if includeJokes {
		commands["diddy"] = Diddy
		commands["e"] = E
		commands["june"] = June
		commands["mae"] = Mae
	}

	return commands
}

func HandleComponent(ctx commandrouter.Context, event *events.ComponentInteractionCreate) {
	helpers.HandleComponent(ctx, event)
}
