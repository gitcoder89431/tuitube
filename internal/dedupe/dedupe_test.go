package dedupe

import "testing"

func c(yid, artist, title, raw string, pub int64) Candidate {
	return Candidate{YoutubeID: yid, Artist: artist, SongTitle: title, RawTitle: raw, PublishedAt: pub}
}

// Version-blind matching: the plain track and its edits are one group.
func TestFindCollapsesVersions(t *testing.T) {
	in := []Candidate{
		c("a", "Tom Odell", "Another Love", "Tom Odell - Another Love (Slowed)", 300),
		c("b", "Tom Odell", "Another Love", "Tom Odell - Another Love", 100),
		c("c", "Tom Odell", "Another Love", "Tom Odell - Another Love (Sped Up)", 200),
	}
	got := Find(in, Options{})
	if len(got) != 1 {
		t.Fatalf("got %d groups, want 1", len(got))
	}
	if got[0].Keep.YoutubeID != "b" {
		t.Errorf("kept %q, want %q (plain recording preferred over edits)", got[0].Keep.YoutubeID, "b")
	}
	if len(got[0].Drop) != 2 {
		t.Errorf("dropping %d, want 2", len(got[0].Drop))
	}
}

// With no plain copy, the earliest upload wins.
func TestFindPrefersEarliestWhenAllMarked(t *testing.T) {
	in := []Candidate{
		c("late", "X", "Song", "X - Song (Remix)", 900),
		c("early", "X", "Song", "X - Song (Slowed)", 100),
	}
	got := Find(in, Options{})
	if len(got) != 1 || got[0].Keep.YoutubeID != "early" {
		t.Fatalf("kept %+v, want early", got[0].Keep.YoutubeID)
	}
}

// Featured-artist clauses are placed inconsistently and must not split a group.
func TestFindIgnoresFeatPlacement(t *testing.T) {
	in := []Candidate{
		c("a", "beabadoobee", "All I Did Was Dream of You ft. The Marias", "raw a", 100),
		c("b", "beabadoobee", "All I Did Was Dream of You", "raw b", 200),
	}
	if got := Find(in, Options{}); len(got) != 1 {
		t.Fatalf("got %d groups, want 1 (feat clause should not split)", len(got))
	}
}

// Different songs by the same artist must stay separate.
func TestFindKeepsDistinctSongsApart(t *testing.T) {
	in := []Candidate{
		c("a", "Oasis", "Wonderwall", "Oasis - Wonderwall", 1),
		c("b", "Oasis", "Champagne Supernova", "Oasis - Champagne Supernova", 2),
	}
	if got := Find(in, Options{}); len(got) != 0 {
		t.Fatalf("got %d groups, want 0", len(got))
	}
}

// Same title by different artists is not a duplicate.
func TestFindKeepsDifferentArtistsApart(t *testing.T) {
	in := []Candidate{
		c("a", "Artist One", "Halo", "Artist One - Halo", 1),
		c("b", "Artist Two", "Halo", "Artist Two - Halo", 2),
	}
	if got := Find(in, Options{}); len(got) != 0 {
		t.Fatalf("got %d groups, want 0", len(got))
	}
}

// Rows with a missing artist or title have too weak an identity to match on.
func TestFindSkipsWeakIdentities(t *testing.T) {
	in := []Candidate{
		c("a", "", "Some Song", "raw", 1),
		c("b", "", "Some Song", "raw", 2),
		c("c", "Artist", "", "raw", 3),
		c("d", "Artist", "", "raw", 4),
	}
	if got := Find(in, Options{}); len(got) != 0 {
		t.Fatalf("got %d groups, want 0 (weak identities must be skipped)", len(got))
	}
}

// The same input must always produce the same keeper.
func TestFindIsDeterministic(t *testing.T) {
	in := []Candidate{
		c("zzz", "X", "Song", "X - Song", 100),
		c("aaa", "X", "Song", "X - Song", 100),
	}
	for range 20 {
		got := Find(in, Options{})
		if got[0].Keep.YoutubeID != "aaa" {
			t.Fatalf("kept %q, want aaa", got[0].Keep.YoutubeID)
		}
	}
}

func TestNormalizeStripsNoise(t *testing.T) {
	cases := [][2]string{
		{"Another Love (Slowed)", "another love"},
		{"Another Love (Sped Up)", "another love"},
		{"Song ft. Someone", "song"},
		{"Song feat. Someone", "song"},
		{"Sure  Thing!!", "sure thing"},
	}
	for _, tc := range cases {
		if got := normalize(tc[0], Options{}); got != tc[1] {
			t.Errorf("normalize(%q) = %q, want %q", tc[0], got, tc[1])
		}
	}
}

// With KeepVersions, alternate renderings survive as distinct songs.
func TestFindKeepVersions(t *testing.T) {
	in := []Candidate{
		c("plain", "Tom Odell", "Another Love", "Tom Odell - Another Love", 100),
		c("slow", "Tom Odell", "Another Love (Slowed)", "Tom Odell - Another Love (Slowed)", 200),
		c("dupe", "Tom Odell", "Another Love", "Tom Odell - Another Love", 300),
	}
	got := Find(in, Options{KeepVersions: true})
	if len(got) != 1 {
		t.Fatalf("got %d groups, want 1 (only the two plain copies collapse)", len(got))
	}
	if len(got[0].Drop) != 1 {
		t.Fatalf("dropping %d, want 1 — the slowed edit must survive", len(got[0].Drop))
	}
	if got[0].Drop[0].YoutubeID != "dupe" {
		t.Errorf("dropped %q, want %q", got[0].Drop[0].YoutubeID, "dupe")
	}
}
