package utils

import (
	"net/url"
	"regexp"
	"strings"
)

const YouTubeSearchPrefix = "ytsearch:"

var TopicSuffixPattern = regexp.MustCompile(`(?i)\s*-\s*topic$`)

func IsURL(input string) bool {
	parsed, err := url.Parse(input)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

func IsYouTubeURL(input string) bool {
	parsed, err := url.Parse(input)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "youtube.com" || host == "www.youtube.com" || host == "music.youtube.com" || host == "youtu.be"
}

func IsYouTubeVideoID(identifier string) bool {
	if len(identifier) != 11 {
		return false
	}

	for _, r := range identifier {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			continue
		}
		return false
	}

	return true
}

func YouTubeVideoID(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return ""
	}

	parsed, err := url.Parse(identifier)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "youtu.be" {
		videoID := strings.Trim(strings.TrimPrefix(parsed.EscapedPath(), "/"), "/")
		if IsYouTubeVideoID(videoID) {
			return videoID
		}
		return ""
	}

	if host != "youtube.com" && host != "www.youtube.com" && host != "music.youtube.com" {
		return ""
	}
	if parsed.Path != "/watch" || parsed.Query().Get("list") != "" {
		return ""
	}

	videoID := parsed.Query().Get("v")
	if IsYouTubeVideoID(videoID) {
		return videoID
	}
	return ""
}

func YouTubeMusicTrackURL(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return ""
	}

	parsed, err := url.Parse(identifier)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		if IsYouTubeVideoID(identifier) {
			return "https://music.youtube.com/watch?v=" + identifier
		}
		return ""
	}

	host := strings.ToLower(parsed.Hostname())
	if host != "youtube.com" && host != "www.youtube.com" && host != "music.youtube.com" && host != "youtu.be" {
		return ""
	}

	videoID := parsed.Query().Get("v")
	if host == "youtu.be" {
		videoID = strings.Trim(strings.TrimPrefix(parsed.EscapedPath(), "/"), "/")
	}
	if !IsYouTubeVideoID(videoID) {
		return ""
	}

	musicURL := url.URL{
		Scheme: "https",
		Host:   "music.youtube.com",
		Path:   "/watch",
	}
	query := musicURL.Query()
	query.Set("v", videoID)
	musicURL.RawQuery = query.Encode()

	return musicURL.String()
}
