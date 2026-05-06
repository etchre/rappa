package test

import (
	"testing"

	"rappa/utils"
)

func TestIsURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "https url",
			input: "https://music.youtube.com/watch?v=abcdefghijk",
			want:  true,
		},
		{
			name:  "http url",
			input: "http://example.com/path",
			want:  true,
		},
		{
			name:  "search text",
			input: "artist song name",
			want:  false,
		},
		{
			name:  "host without scheme",
			input: "youtube.com/watch?v=abcdefghijk",
			want:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := utils.IsURL(test.input); got != test.want {
				t.Fatalf("IsURL(%q) = %v, want %v", test.input, got, test.want)
			}
		})
	}
}

func TestIsYouTubeURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "youtube url",
			input: "https://youtube.com/watch?v=abcdefghijk",
			want:  true,
		},
		{
			name:  "www youtube url",
			input: "https://www.youtube.com/watch?v=abcdefghijk",
			want:  true,
		},
		{
			name:  "music youtube url",
			input: "https://music.youtube.com/watch?v=abcdefghijk",
			want:  true,
		},
		{
			name:  "short youtube url",
			input: "https://youtu.be/abcdefghijk",
			want:  true,
		},
		{
			name:  "non youtube url",
			input: "https://soundcloud.com/artist/track",
			want:  false,
		},
		{
			name:  "raw video id",
			input: "abcdefghijk",
			want:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := utils.IsYouTubeURL(test.input); got != test.want {
				t.Fatalf("IsYouTubeURL(%q) = %v, want %v", test.input, got, test.want)
			}
		})
	}
}

func TestIsYouTubeVideoID(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       bool
	}{
		{
			name:       "valid id",
			identifier: "abcdefghijk",
			want:       true,
		},
		{
			name:       "valid id with dash and underscore",
			identifier: "abc-def_ghi",
			want:       true,
		},
		{
			name:       "too short",
			identifier: "abcdefghij",
			want:       false,
		},
		{
			name:       "too long",
			identifier: "abcdefghijkl",
			want:       false,
		},
		{
			name:       "invalid character",
			identifier: "abcdefghi!k",
			want:       false,
		},
		{
			name:       "empty",
			identifier: "",
			want:       false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := utils.IsYouTubeVideoID(test.identifier); got != test.want {
				t.Fatalf("IsYouTubeVideoID(%q) = %v, want %v", test.identifier, got, test.want)
			}
		})
	}
}

func TestYouTubeMusicTrackURL(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       string
	}{
		{
			name:       "www youtube watch url",
			identifier: "https://www.youtube.com/watch?v=abcdefghijk",
			want:       "https://music.youtube.com/watch?v=abcdefghijk",
		},
		{
			name:       "music youtube watch url",
			identifier: "https://music.youtube.com/watch?v=abcdefghijk",
			want:       "https://music.youtube.com/watch?v=abcdefghijk",
		},
		{
			name:       "short youtube url",
			identifier: "https://youtu.be/abcdefghijk",
			want:       "https://music.youtube.com/watch?v=abcdefghijk",
		},
		{
			name:       "short youtube url with trailing slash",
			identifier: "https://youtu.be/abcdefghijk/",
			want:       "https://music.youtube.com/watch?v=abcdefghijk",
		},
		{
			name:       "video id",
			identifier: "abcdefghijk",
			want:       "https://music.youtube.com/watch?v=abcdefghijk",
		},
		{
			name:       "invalid video id",
			identifier: "abcdefghi!k",
			want:       "",
		},
		{
			name:       "non youtube url",
			identifier: "https://soundcloud.com/artist/track",
			want:       "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := utils.YouTubeMusicTrackURL(test.identifier); got != test.want {
				t.Fatalf("YouTubeMusicTrackURL(%q) = %q, want %q", test.identifier, got, test.want)
			}
		})
	}
}

func TestYouTubeVideoID(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       string
	}{
		{
			name:       "music youtube watch url",
			identifier: "https://music.youtube.com/watch?v=abcdefghijk",
			want:       "abcdefghijk",
		},
		{
			name:       "www youtube watch url",
			identifier: "https://www.youtube.com/watch?v=abcdefghijk",
			want:       "abcdefghijk",
		},
		{
			name:       "short youtube url",
			identifier: "https://youtu.be/abcdefghijk",
			want:       "abcdefghijk",
		},
		{
			name:       "short youtube url with trailing slash",
			identifier: "https://youtu.be/abcdefghijk/",
			want:       "abcdefghijk",
		},
		{
			name:       "playlist url stays out of resolver",
			identifier: "https://music.youtube.com/watch?v=abcdefghijk&list=OLAK5uy_example",
			want:       "",
		},
		{
			name:       "raw id stays out of normal load resolver",
			identifier: "abcdefghijk",
			want:       "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := utils.YouTubeVideoID(test.identifier); got != test.want {
				t.Fatalf("YouTubeVideoID(%q) = %q, want %q", test.identifier, got, test.want)
			}
		})
	}
}
