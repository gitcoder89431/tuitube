// Package dedupe groups tracks that are the same song and picks which copy to
// keep.
//
// The five curated channels chase the same hits, so one recording commonly
// appears as several distinct YouTube videos. Matching is version-blind: a
// track and its slowed, sped-up or remixed edits collapse into one group, and
// only a single row survives. That is deliberately aggressive — it trades some
// genuinely different edits for a library without near-identical rows.
//
// Nothing here touches the database. Callers decide what to do with the groups.
package dedupe

import (
	"regexp"
	"sort"
	"strings"
)

type Candidate struct {
	YoutubeID   string
	Artist      string
	SongTitle   string
	RawTitle    string
	PublishedAt int64
	Station     string
}

type Group struct {
	// Key is the normalised identity the members share.
	Key string
	// Keep is the copy that survives; Drop are the redundant ones.
	Keep Candidate
	Drop []Candidate
}

var (
	// versionMarkerRe matches qualifiers describing an alternate rendering of
	// the same underlying song. Stripping them is what makes matching
	// version-blind.
	versionMarkerRe = regexp.MustCompile(`(?i)\b(?:slowed|sped\s*up|spedup|speed\s*up|reverb(?:ed)?|nightcore|8d|bass\s*boosted|tiktok|remix|edit|acoustic|instrumental|live|cover|version|mix|extended|radio\s*edit|club\s*mix|bootleg)\b`)
	// featRe drops featured-artist clauses, which are inconsistently placed and
	// would otherwise split a group.
	featRe = regexp.MustCompile(`(?i)\b(?:feat\.?|ft\.?|featuring|with)\b.*$`)
	// punctRe reduces everything else to words.
	punctRe      = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
	multiSpaceRe = regexp.MustCompile(`\s+`)
)

// Options controls how aggressively copies are collapsed.
type Options struct {
	// KeepVersions treats alternate renderings as distinct songs: a slowed
	// edit, a remix, and a guest-featuring cut each survive. Only byte-level
	// restatements of the same version are collapsed.
	KeepVersions bool
}

// normalize reduces a string to a comparable identity.
func normalize(s string, opts Options) string {
	s = strings.ToLower(s)
	if !opts.KeepVersions {
		s = featRe.ReplaceAllString(s, " ")
		s = versionMarkerRe.ReplaceAllString(s, " ")
	}
	s = punctRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(multiSpaceRe.ReplaceAllString(s, " "))
}

// Key is the identity two candidates must share to be duplicates.
func Key(c Candidate, opts Options) string {
	return normalize(c.Artist, opts) + "|" + normalize(c.SongTitle, opts)
}

// markerCount counts version qualifiers in the original title. A copy with
// none is the plain recording and is preferred.
func markerCount(c Candidate) int {
	return len(versionMarkerRe.FindAllString(c.RawTitle, -1))
}

// better reports whether a should be kept over b: fewest version markers first
// (prefer the plain recording), then the earliest upload, then the youtube_id
// so the choice is deterministic across runs.
func better(a, b Candidate) bool {
	if ma, mb := markerCount(a), markerCount(b); ma != mb {
		return ma < mb
	}
	if a.PublishedAt != b.PublishedAt {
		// Treat a missing date as newest so a dated copy wins.
		if a.PublishedAt == 0 {
			return false
		}
		if b.PublishedAt == 0 {
			return true
		}
		return a.PublishedAt < b.PublishedAt
	}
	return a.YoutubeID < b.YoutubeID
}

// Find groups candidates that share an identity, returning only those with
// something to drop. Candidates with no artist or no title are skipped: their
// identity is too weak to match on safely.
func Find(candidates []Candidate, opts Options) []Group {
	byKey := map[string][]Candidate{}
	for _, c := range candidates {
		if strings.TrimSpace(c.Artist) == "" || strings.TrimSpace(c.SongTitle) == "" {
			continue
		}
		k := Key(c, opts)
		if k == "|" || strings.TrimSpace(k) == "|" {
			continue
		}
		byKey[k] = append(byKey[k], c)
	}

	var groups []Group
	for k, members := range byKey {
		if len(members) < 2 {
			continue
		}
		sort.Slice(members, func(i, j int) bool { return better(members[i], members[j]) })
		groups = append(groups, Group{Key: k, Keep: members[0], Drop: members[1:]})
	}
	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].Drop) != len(groups[j].Drop) {
			return len(groups[i].Drop) > len(groups[j].Drop)
		}
		return groups[i].Key < groups[j].Key
	})
	return groups
}
