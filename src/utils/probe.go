package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const YtDlpProbeTimeout = 10 * time.Second

type ProbeResult int

const (
	ProbeResultAvailable   ProbeResult = iota
	ProbeResultPremium
	ProbeResultUnavailable
)

func (r ProbeResult) String() string {
	switch r {
	case ProbeResultAvailable:
		return "available"
	case ProbeResultPremium:
		return "premium"
	case ProbeResultUnavailable:
		return "unavailable"
	default:
		return "unknown"
	}
}

func ProbeIdentifier(ctx context.Context, identifier string) (ProbeResult, error) {
	probeCtx, cancel := context.WithTimeout(ctx, YtDlpProbeTimeout)
	defer cancel()

	cmd := exec.CommandContext(
		probeCtx,
		YtDlpCommand(),
		"--no-config",
		"--no-warnings",
		"--print", "%(id)s",
		"--extractor-args",
		"youtube:player_client=web",
		identifier,
	)
	output, err := cmd.CombinedOutput()
	if probeCtx.Err() != nil {
		return ProbeResultAvailable, fmt.Errorf("yt-dlp probe timed out")
	}

	text := strings.ToLower(string(output))

	if err == nil {
		return ProbeResultAvailable, nil
	}

	if strings.Contains(text, "only available to music premium members") ||
		(strings.Contains(text, "music premium") && strings.Contains(text, "member")) {
		return ProbeResultPremium, nil
	}

	if strings.Contains(text, "video unavailable") ||
		strings.Contains(text, "video is not available") ||
		strings.Contains(text, "this video is no longer available") {
		return ProbeResultUnavailable, nil
	}

	return ProbeResultAvailable, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
}

func YtDlpCommand() string {
	if value := strings.TrimSpace(os.Getenv("YTDLP_COMMAND")); value != "" {
		return value
	}
	return "yt-dlp"
}
