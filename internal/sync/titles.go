package sync

import (
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// bracketPairs lists all bracket styles to normalize.
// Standard [] are included so their contents get the same treatment.
var bracketPairs = [][2]string{
	{"[", "]"},
	{"【", "】"},
	{"「", "」"},
	{"（", "）"},
}

var (
	// matches common emoji blocks
	emojiRe = regexp.MustCompile(
		`[\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}\x{1F1E0}-\x{1F1FF}]+`)
	// matches *ALL CAPS PROMO BLOCKS*
	capsBlockRe = regexp.MustCompile(`\*\b[A-Z ]+\b\*`)
	// missing space before [ — e.g. "Song[Lyrics]" → "Song [Lyrics]"
	missingSpaceBracket = regexp.MustCompile(`(\S)\[`)
	// collapses runs of whitespace
	multiSpaceRe = regexp.MustCompile(`\s{2,}`)
	// Japanese character ranges (hiragana, katakana, kanji)
	japaneseRe = regexp.MustCompile(`[一-龠ぁ-ゔァ-ヴー々〆〤ヶ]`)
)

var bannedChars = []string{"♪"}

// CleanArtist strips non-basic-Latin characters from a parsed artist name.
// NFKD normalisation runs first so accented letters (é → e + combining mark)
// survive as their base ASCII letter rather than vanishing entirely.
func CleanArtist(artist string) string {
	var b strings.Builder
	for _, r := range norm.NFKD.String(artist) {
		if r <= 0x7E {
			b.WriteRune(r)
		}
		// combining marks and non-Latin scripts are dropped
	}
	return strings.TrimSpace(multiSpaceRe.ReplaceAllString(b.String(), " "))
}

// CleanTitle normalises a raw YouTube video title for display.
// Ported from https://github.com/KraXen72/shira (MIT).
func CleanTitle(title string) string {
	for _, c := range bannedChars {
		title = strings.ReplaceAll(title, c, "")
	}

	// Normalise all bracket styles: keep content if it references a cover or
	// contains Japanese; otherwise strip the whole group.
	for _, pair := range bracketPairs {
		lb, rb := pair[0], pair[1]
		var out strings.Builder
		remaining := title
		for {
			start := strings.Index(remaining, lb)
			if start < 0 {
				out.WriteString(remaining)
				break
			}
			rest := remaining[start+len(lb):]
			end := strings.Index(rest, rb)
			if end < 0 {
				out.WriteString(remaining)
				break
			}
			content := rest[:end]
			out.WriteString(remaining[:start])
			if strings.Contains(strings.ToLower(content), "cover") || japaneseRe.MatchString(content) {
				out.WriteString("[" + content + "]")
			}
			// advance past the closing bracket to avoid re-scanning replaced content
			remaining = rest[end+len(rb):]
		}
		title = out.String()
	}

	title = emojiRe.ReplaceAllString(title, "")
	title = capsBlockRe.ReplaceAllString(title, "")
	title = missingSpaceBracket.ReplaceAllString(title, "${1} [")
	title = multiSpaceRe.ReplaceAllString(title, " ")
	title = strings.ReplaceAll(title, "_", "-")
	return strings.TrimSpace(title)
}
