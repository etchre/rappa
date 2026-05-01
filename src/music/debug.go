package music

import (
	"fmt"
	"strings"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
)

func debugTrackLoad(node disgolink.Node, originalIdentifier string, playableIdentifier string) {
	fmt.Printf(
		"[lavalink-debug] loading track on node=%q input=%q resolved_identifier=%q\n",
		nodeName(node),
		originalIdentifier,
		playableIdentifier,
	)
}

func debugTrackLoadError(node disgolink.Node, identifier string, err error) {
	fmt.Printf(
		"[lavalink-debug] load request failed node=%q input=%q error=%v\n",
		nodeName(node),
		identifier,
		err,
	)
	if looksAccountGated(err.Error()) {
		fmt.Printf("[lavalink-debug] likely account/premium-gated track input=%q\n", identifier)
	}
}

func debugTrackLoadEmpty(node disgolink.Node, originalIdentifier string, playableIdentifier string) {
	fmt.Printf(
		"[lavalink-debug] no tracks found node=%q input=%q resolved_identifier=%q\n",
		nodeName(node),
		originalIdentifier,
		playableIdentifier,
	)
}

func debugTrackPlayError(player disgolink.Player, track lavalink.Track, err error) {
	fmt.Printf(
		"[lavalink-debug] play update failed node=%q track=%q error=%v\n",
		playerNodeName(player),
		trackTitle(track),
		err,
	)
	if looksAccountGated(err.Error()) {
		fmt.Printf("[lavalink-debug] likely account/premium-gated track track=%q\n", trackTitle(track))
	}
}

func debugLavalinkException(stage string, nodeName string, subject string, exception lavalink.Exception) {
	fmt.Printf(
		"[lavalink-debug] %s exception node=%q subject=%q severity=%q message=%q cause=%q\n",
		stage,
		nodeName,
		subject,
		exception.Severity,
		exception.Message,
		exception.Cause,
	)

	if looksAccountGated(exception.Message, exception.Cause, exception.CauseStackTrace) {
		fmt.Printf("[lavalink-debug] likely account/premium-gated track subject=%q\n", subject)
	}
}

func nodeName(node disgolink.Node) string {
	if node == nil {
		return ""
	}

	return node.Config().Name
}

func playerNodeName(player disgolink.Player) string {
	if player == nil {
		return ""
	}

	return nodeName(player.Node())
}

func trackTitle(track lavalink.Track) string {
	if track.Info.Author == "" {
		return track.Info.Title
	}

	return track.Info.Author + " - " + track.Info.Title
}

func looksAccountGated(values ...string) bool {
	text := strings.ToLower(strings.Join(values, "\n"))
	accountGateMarkers := []string{
		"premium",
		"paid",
		"purchase",
		"payment required",
		"requires payment",
		"members-only",
		"members only",
		"subscriber",
		"sign in",
		"login",
		"account",
		"not available",
	}

	for _, marker := range accountGateMarkers {
		if strings.Contains(text, marker) {
			return true
		}
	}

	return false
}
