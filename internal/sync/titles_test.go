package sync

import "testing"

var cleanTitleTests = []struct {
	in   string
	want string
}{
	{"Song Title 🎵", "Song Title"},
	{"Song [Official MV]", "Song"},
	{"Song【Cover】", "Song [Cover]"}, // non-ASCII bracket → preserved as [Cover]
	{"Song【日本語】", "Song [日本語]"},
	{"Song *FREE DOWNLOAD*", "Song"},
	{"Song_Title", "Song-Title"},
	{"Song[Lyrics]", "Song"}, // stripped, no remaining [ so no space injected
	{"  Song   Title  ", "Song Title"},
	{"Song ♪ Title", "Song Title"},
	{"Song「カバー」", "Song [カバー]"},
	{"SONG TITLE", "SONG TITLE"},
	{"Song [unclosed", "Song [unclosed"},
}

func TestCleanTitle(t *testing.T) {
	for _, tc := range cleanTitleTests {
		got := CleanTitle(tc.in)
		if got != tc.want {
			t.Errorf("CleanTitle(%q)\n  got  %q\n  want %q", tc.in, got, tc.want)
		}
	}
}
