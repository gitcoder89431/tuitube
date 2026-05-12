package sync

import (
	"regexp"
	"strings"
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
		for {
			start := strings.Index(title, lb)
			if start < 0 {
				break
			}
			rest := title[start+len(lb):]
			end := strings.Index(rest, rb)
			if end < 0 {
				break
			}
			content := rest[:end]
			var replacement string
			if strings.Contains(strings.ToLower(content), "cover") || japaneseRe.MatchString(content) {
				replacement = "[" + content + "]"
			}
			title = title[:start] + replacement + title[start+len(lb)+end+len(rb):]
		}
	}

	title = emojiRe.ReplaceAllString(title, "")
	title = capsBlockRe.ReplaceAllString(title, "")
	title = missingSpaceBracket.ReplaceAllString(title, "${1} [")
	title = multiSpaceRe.ReplaceAllString(title, " ")
	title = strings.ReplaceAll(title, "_", "-")
	return strings.TrimSpace(title)
}
