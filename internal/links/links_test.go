package links

import "testing"

func TestClassifyDead(t *testing.T) {
	cases := []struct {
		name string
		err  string
		want string
	}{
		{"private", "ERROR: [youtube] 18XN3U0lP7Y: Private video. Sign in if you've been granted access to this video", "Private video. Sign in if you've been granted access to this video"},
		{"unavailable", "ERROR: [youtube] CIzz_mRywh4: This video is unavailable", "This video is unavailable"},
		{"short unavailable", "ERROR: [youtube] abc: Video unavailable", "Video unavailable"},
		{"removed by uploader", "ERROR: [youtube] abc: Video has been removed by the uploader", "Video has been removed by the uploader"},
		{"terminated account", "ERROR: [youtube] abc: This video is no longer available because the account associated with this video has been terminated.", "This video is no longer available because the account associated with this video has been terminated."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason := classify(tc.err)
			if got != StatusDead {
				t.Errorf("classify(%q) = %v, want %v", tc.err, got, StatusDead)
			}
			if reason != tc.want {
				t.Errorf("reason = %q, want %q", reason, tc.want)
			}
		})
	}
}

// A region block reads as "unavailable" but the video may play fine elsewhere.
// Pruning these would delete working tracks from a library served to others.
func TestClassifyRegionBlockIsNotDead(t *testing.T) {
	cases := []string{
		"ERROR: [youtube] abc: Video unavailable. This video is not available in your country",
		"ERROR: [youtube] abc: The uploader has not made this video available in your country",
		"ERROR: [youtube] abc: Video unavailable. The uploader has closed their account and blocked it in your country",
	}
	for _, e := range cases {
		if got, _ := classify(e); got != StatusUnknown {
			t.Errorf("classify(%q) = %v, want %v (region blocks must never be pruned)", e, got, StatusUnknown)
		}
	}
}

// Anything unrecognised must stay unknown so a transient failure never deletes.
func TestClassifyUnrecognisedIsUnknown(t *testing.T) {
	cases := []string{
		"ERROR: unable to download video data: HTTP Error 503: Service Unavailable",
		"ERROR: [youtube] abc: Unable to extract player response",
		"ERROR: The read operation timed out",
		"ERROR: [youtube] abc: Sign in to confirm you're not a bot",
		"",
	}
	for _, e := range cases {
		if got, _ := classify(e); got != StatusUnknown {
			t.Errorf("classify(%q) = %v, want %v", e, got, StatusUnknown)
		}
	}
}

func TestClassifyStripsPrefix(t *testing.T) {
	_, reason := classify("ERROR: [youtube] dQw4w9WgXcQ: Private video")
	if reason != "Private video" {
		t.Errorf("reason = %q, want %q", reason, "Private video")
	}
}
