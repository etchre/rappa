package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgolink/v3/lavalink"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands/helpers"
)

func HandleAutocomplete(ctx commandrouter.Context, event *events.AutocompleteInteractionCreate) {
	data := event.Data
	if !isPlayCommand(data.CommandName) || data.Focused().Name != "query" {
		respondAutocomplete(event, nil)
		return
	}

	query := strings.TrimSpace(data.String("query"))
	if query == "" {
		respondAutocomplete(event, nil)
		return
	}
	if ctx.Player == nil {
		respondAutocomplete(event, nil)
		return
	}

	tracks, err := ctx.Player.Search(ctx.Context, query, 5)
	if err != nil {
		fmt.Fprintf(os.Stderr, "autocomplete search failed: %v\n", err)
		respondAutocomplete(event, nil)
		return
	}

	choices := make([]discord.AutocompleteChoice, 0, len(tracks))
	for _, track := range tracks {
		value := autocompleteValue(track)
		if value == "" {
			continue
		}

		choices = append(choices, discord.AutocompleteChoiceString{
			Name:  limitAutocompleteText(autocompleteName(track), 100),
			Value: limitAutocompleteText(value, 100),
		})
	}

	respondAutocomplete(event, choices)
}

func isPlayCommand(name string) bool {
	return name == "play" || name == "playnext" || name == "playrightnow"
}

func autocompleteName(track lavalink.Track) string {
	return fmt.Sprintf("%s [%s]", helpers.TrackTitle(track), helpers.FormatDuration(track.Info.Length))
}

func autocompleteValue(track lavalink.Track) string {
	if track.Info.URI != nil && *track.Info.URI != "" {
		if len(*track.Info.URI) <= 100 {
			return *track.Info.URI
		}
	}

	if track.Info.Identifier == "" {
		return ""
	}

	return "https://www.youtube.com/watch?v=" + track.Info.Identifier
}

func limitAutocompleteText(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	if limit <= 1 {
		return value[:limit]
	}

	return value[:limit-3] + "..."
}

func respondAutocomplete(event *events.AutocompleteInteractionCreate, choices []discord.AutocompleteChoice) {
	if err := event.AutocompleteResult(choices); err != nil {
		fmt.Fprintf(os.Stderr, "autocomplete response failed: %v\n", err)
	}
}
