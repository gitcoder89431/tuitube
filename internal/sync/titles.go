package sync

import (
	"regexp"
	"strings"
	"unicode"

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
	// parenNoiseRe matches parenthesised upload boilerplate that carries no
	// musical information. Anything unrecognised inside parens is kept, so
	// meaningful qualifiers — (Slowed), (Sped Up), (Remix), (feat. X) — survive.
	parenNoiseRe = regexp.MustCompile(`(?i)\s*\([^()]*\b(?:lyrics?|letra|paroles|testo|legendado|tradu\w*|official|audio|visuali[sz]er|lyric\s*video|official\s*video|hd|hq|4k|full\s*song|color\s*coded)\b[^()]*\)`)
	// bare boilerplate outside brackets, e.g. "Song (TikTok Remix) Lyrics",
	// "Ready or Not Lyrics)" (unbalanced in the source), or a stray "LYRICS"
	// before a quoted snippet. A trailing ")" is consumed so unbalanced
	// sources do not leave one behind.
	trailingNoiseRe = regexp.MustCompile(`(?i)\s+(?:lyrics?|letra)\b\)?`)
	// empty or whitespace-only bracket groups left behind after stripping
	emptyGroupRe = regexp.MustCompile(`\s*[\(\[]\s*[\)\]]`)
	// artist/song separator: a dash variant with whitespace after it. Requiring
	// trailing space keeps hyphenated names such as "Jay-Z" intact.
	dashSplitRe = regexp.MustCompile(`\s*[-\x{2013}\x{2014}]\s+`)
)

var bannedChars = []string{"♪"}

// CleanArtist strips non-basic-Latin characters from a parsed artist name.
// NFKD normalisation runs first so accented letters (é → e + combining mark)
// survive as their base ASCII letter rather than vanishing entirely.
func CleanArtist(artist string) string {
	var b strings.Builder
	for _, r := range norm.NFC.String(artist) {
		switch {
		case r <= 0x7E:
			b.WriteRune(r)
		case unicode.Is(unicode.Latin, r):
			// Accented Latin names (Tiësto, Måneskin, ROSALÍA) are kept as
			// written. FTS5's unicode61 tokenizer folds diacritics, so an
			// ASCII query still matches them — stripping the accent would
			// only make the display worse.
			b.WriteRune(r)
		}
		// other scripts, combining marks and emoji are dropped
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
	title = parenNoiseRe.ReplaceAllString(title, "")
	title = trailingNoiseRe.ReplaceAllString(title, "")
	title = emptyGroupRe.ReplaceAllString(title, "")
	title = multiSpaceRe.ReplaceAllString(title, " ")
	return strings.TrimSpace(title)
}

// SplitArtistTitle splits a cleaned title into artist and song on the first
// dash variant followed by whitespace. Returns an empty artist when no
// separator is present.
func SplitArtistTitle(cleaned string) (artist, songTitle string) {
	if loc := dashSplitRe.FindStringIndex(cleaned); loc != nil {
		return strings.TrimSpace(cleaned[:loc[0]]), strings.TrimSpace(cleaned[loc[1]:])
	}
	return "", strings.TrimSpace(cleaned)
}
