package test

import (
	"testing"

	"ytdlpPlayer/music"
)

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
			name:       "video id",
			identifier: "abcdefghijk",
			want:       "https://music.youtube.com/watch?v=abcdefghijk",
		},
		{
			name:       "non youtube url",
			identifier: "https://soundcloud.com/artist/track",
			want:       "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := music.YouTubeMusicTrackURL(test.identifier); got != test.want {
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
			if got := music.YouTubeVideoID(test.identifier); got != test.want {
				t.Fatalf("YouTubeVideoID(%q) = %q, want %q", test.identifier, got, test.want)
			}
		})
	}
}
