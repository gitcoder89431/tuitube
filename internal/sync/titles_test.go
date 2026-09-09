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

var parenNoiseTests = []struct {
	in   string
	want string
}{
	// Boilerplate is stripped...
	{"Shakira - Última (Letra/Lyrics)", "Shakira - Última"},
	{"Song (Lyrics)", "Song"},
	{"Song (Letra / Lyrics )", "Song"},
	{"Song (Official Video)", "Song"},
	{"Song (Official Music Video)", "Song"},
	{"Song (Lyric Video)", "Song"},
	{"Song (Audio)", "Song"},
	{"Song (Visualizer)", "Song"},
	{"Smooth Operator (TikTok Remix) Lyrics", "Smooth Operator (TikTok Remix)"},
	// ...but musically meaningful qualifiers survive.
	{"Tom Odell - Another Love (Slowed)", "Tom Odell - Another Love (Slowed)"},
	{"Tom Odell - Another Love (Sped Up)", "Tom Odell - Another Love (Sped Up)"},
	{"Song (Remix)", "Song (Remix)"},
	{"Song (Acoustic)", "Song (Acoustic)"},
	{"Song ft. Someone", "Song ft. Someone"},
	{"Gianni Blu - Billie Jean (Afro Edit)", "Gianni Blu - Billie Jean (Afro Edit)"},
}

func TestCleanTitleParenNoise(t *testing.T) {
	for _, tc := range parenNoiseTests {
		if got := CleanTitle(tc.in); got != tc.want {
			t.Errorf("CleanTitle(%q)\n  got  %q\n  want %q", tc.in, got, tc.want)
		}
	}
}

var splitTests = []struct {
	in            string
	artist, title string
}{
	{"Tom Odell - Another Love", "Tom Odell", "Another Love"},
	{"Childish Gambino – Little Foot Big Foot", "Childish Gambino", "Little Foot Big Foot"}, // en dash
	{"Shakira — Última", "Shakira", "Última"},                                               // em dash
	{"SZA- Anything", "SZA", "Anything"},                                                    // no space before dash
	{"Lost Frequencies- Rise", "Lost Frequencies", "Rise"},
	{"Jay-Z - Money", "Jay-Z", "Money"},     // hyphenated name preserved
	{"Ke$ha - Tik-Tok", "Ke$ha", "Tik-Tok"}, // hyphen in song preserved
	{"No Separator Here", "", "No Separator Here"},
	{"Artist - Song - Live", "Artist", "Song - Live"}, // splits on first only
}

func TestSplitArtistTitle(t *testing.T) {
	for _, tc := range splitTests {
		a, s := SplitArtistTitle(tc.in)
		if a != tc.artist || s != tc.title {
			t.Errorf("SplitArtistTitle(%q)\n  got  artist=%q title=%q\n  want artist=%q title=%q",
				tc.in, a, s, tc.artist, tc.title)
		}
	}
}

func TestCleanArtistKeepsLatinAccents(t *testing.T) {
	keep := []string{"Tiësto", "Måneskin", "ROSALÍA", "Luísa Sonza", "Beéle", "XYLØ"}
	for _, in := range keep {
		if got := CleanArtist(in); got != in {
			t.Errorf("CleanArtist(%q) = %q, want unchanged (FTS folds diacritics already)", in, got)
		}
	}
	// Non-Latin scripts and emoji are still dropped.
	if got := CleanArtist("Nemu ネム"); got != "Nemu" {
		t.Errorf("CleanArtist(%q) = %q, want %q", "Nemu ネム", got, "Nemu")
	}
}

func TestCleanTitleBareLyricsToken(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Ready or Not Lyrics)", "Ready or Not"},
		{"1, 2, 3 (sped up) Lyrics ft. Jason Derulo", "1, 2, 3 (sped up) ft. Jason Derulo"},
		{"Home To Another One Lyrics)", "Home To Another One"},
		// A word merely starting with "lyric" is not a boilerplate token.
		{"The Lyricist", "The Lyricist"},
	}
	for _, tc := range cases {
		if got := CleanTitle(tc.in); got != tc.want {
			t.Errorf("CleanTitle(%q)\n  got  %q\n  want %q", tc.in, got, tc.want)
		}
	}
}
