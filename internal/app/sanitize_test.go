package app

import "testing"

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		artist, title, want string
	}{
		{"AC/DC", "Back in Black", "AC-DC - Back in Black"},
		{"", "Title: With Colon", "Title- With Colon"},
		{"Art*ist", "Track?Name", "Art-ist - Track-Name"},
		{"", "Song", "Song"},
		{"Artist", "Song", "Artist - Song"},
		{"  spaces  ", "  title  ", "spaces - title"},
		{"", `back\slash`, "back-slash"},
	}
	for _, c := range cases {
		got := sanitizeFilename(c.artist, c.title)
		if got != c.want {
			t.Errorf("sanitizeFilename(%q, %q) = %q, want %q", c.artist, c.title, got, c.want)
		}
	}
}
