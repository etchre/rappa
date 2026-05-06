package music

import (
	"log/slog"
	"strings"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
)

func debugTrackLoad(node disgolink.Node, originalIdentifier string, playableIdentifier string) {
	slog.Debug("loading track",
		"node", nodeName(node),
		"input", originalIdentifier,
		"resolved", playableIdentifier,
	)
}

func debugTrackLoadError(node disgolink.Node, originalIdentifier string, playableIdentifier string, err error) {
	slog.Debug("load request failed",
		"node", nodeName(node),
		"input", originalIdentifier,
		"resolved", playableIdentifier,
		"err", err,
	)
	if looksAccountGated(err.Error()) {
		slog.Debug("likely account/premium-gated track",
			"input", originalIdentifier,
			"resolved", playableIdentifier,
		)
	}
}

func debugTrackLoadEmpty(node disgolink.Node, originalIdentifier string, playableIdentifier string) {
	slog.Debug("no tracks found",
		"node", nodeName(node),
		"input", originalIdentifier,
		"resolved", playableIdentifier,
	)
}

func debugTrackPlayError(player disgolink.Player, track lavalink.Track, err error) {
	slog.Debug("play update failed",
		"node", playerNodeName(player),
		"track", TrackTitle(track),
		"err", err,
	)
	if looksAccountGated(err.Error()) {
		slog.Debug("likely account/premium-gated track",
			"track", TrackTitle(track),
		)
	}
}

func debugLavalinkException(stage string, nodeName string, subject string, exception lavalink.Exception) {
	if looksAccountGated(exception.Message, exception.Cause, exception.CauseStackTrace) {
		slog.Debug("lavalink exception, likely premium track",
			"stage", stage,
			"node", nodeName,
			"subject", subject,
		)
		return
	}

	slog.Warn("lavalink exception",
		"stage", stage,
		"node", nodeName,
		"subject", subject,
		"severity", exception.Severity,
		"message", exception.Message,
	)
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
