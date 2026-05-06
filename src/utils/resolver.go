package utils

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const YouTubeResolverTimeout = 10 * time.Second

func ResolveYouTubeVideoID(ctx context.Context, videoID string) (string, error) {
	resolverCtx, cancel := context.WithTimeout(ctx, YouTubeResolverTimeout)
	defer cancel()

	cmd := exec.CommandContext(resolverCtx, PythonCommand(), ResolverScriptPath(), videoID)
	output, err := cmd.CombinedOutput()
	if resolverCtx.Err() != nil {
		return "", fmt.Errorf("resolver timed out")
	}
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}

	resolvedID := strings.TrimSpace(string(output))
	if !IsYouTubeVideoID(resolvedID) {
		return "", fmt.Errorf("resolver returned invalid video id %q", resolvedID)
	}

	return resolvedID, nil
}

func ResolvedYouTubeIdentifier(ctx context.Context, input string) string {
	videoID := YouTubeVideoID(input)
	if videoID == "" {
		return input
	}

	resolvedID, err := ResolveYouTubeVideoID(ctx, videoID)
	if err != nil {
		slog.Error("youtube music id resolver failed", "id", videoID, "err", err)
		return input
	}
	if resolvedID == "" {
		return input
	}

	return YouTubeMusicTrackURL(resolvedID)
}

func PythonCommand() string {
	if value := strings.TrimSpace(os.Getenv("YTMUSIC_RESOLVER_PYTHON")); value != "" {
		return value
	}
	return "python3"
}

func ResolverScriptPath() string {
	if value := strings.TrimSpace(os.Getenv("YTMUSIC_RESOLVER_SCRIPT")); value != "" {
		return value
	}

	for _, candidate := range []string{
		"/app/ytmusic_yt_dlp_test.py",
		"ytmusic_yt_dlp_test.py",
		filepath.Join("..", "ytmusic_yt_dlp_test.py"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return "ytmusic_yt_dlp_test.py"
}
