package db

import "testing"

func TestTombstoneRoundTrip(t *testing.T) {
	d := openMemDB(t)

	if err := d.AddTombstones([]string{"a", "b"}, PruneReasonDead, nil); err != nil {
		t.Fatalf("add dead: %v", err)
	}
	if err := d.AddTombstones([]string{"c"}, PruneReasonDuplicate,
		map[string]string{"c": "keeper"}); err != nil {
		t.Fatalf("add duplicate: %v", err)
	}

	set, err := d.TombstonedIDs()
	if err != nil {
		t.Fatalf("ids: %v", err)
	}
	for _, want := range []string{"a", "b", "c"} {
		if _, ok := set[want]; !ok {
			t.Errorf("id %q missing from tombstone set", want)
		}
	}

	counts, err := d.CountTombstones()
	if err != nil {
		t.Fatalf("counts: %v", err)
	}
	if counts[PruneReasonDead] != 2 || counts[PruneReasonDuplicate] != 1 {
		t.Errorf("counts = %v, want dead=2 duplicate=1", counts)
	}

	entries, err := d.ListTombstones(PruneReasonDuplicate, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 1 || entries[0].ReplacedBy != "keeper" {
		t.Errorf("entries = %+v, want one replaced_by=keeper", entries)
	}
}

// Re-pruning the same id must not fail, and must not lose the replacement.
func TestAddTombstonesIsIdempotent(t *testing.T) {
	d := openMemDB(t)
	ids := []string{"x"}
	if err := d.AddTombstones(ids, PruneReasonDuplicate, map[string]string{"x": "keeper"}); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := d.AddTombstones(ids, PruneReasonDuplicate, nil); err != nil {
		t.Fatalf("second: %v", err)
	}
	entries, _ := d.ListTombstones("", 0)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].ReplacedBy != "keeper" {
		t.Errorf("replaced_by = %q, want it preserved across re-prune", entries[0].ReplacedBy)
	}
}

func TestForgetTombstones(t *testing.T) {
	d := openMemDB(t)
	if err := d.AddTombstones([]string{"a", "b", "c"}, PruneReasonDead, nil); err != nil {
		t.Fatalf("add: %v", err)
	}
	n, err := d.ForgetTombstones([]string{"a"})
	if err != nil || n != 1 {
		t.Fatalf("forget one: n=%d err=%v", n, err)
	}
	set, _ := d.TombstonedIDs()
	if _, ok := set["a"]; ok {
		t.Error("a should be forgotten")
	}
	if len(set) != 2 {
		t.Errorf("remaining = %d, want 2", len(set))
	}
	if n, err = d.ForgetTombstones(nil); err != nil || n != 2 {
		t.Fatalf("forget all: n=%d err=%v", n, err)
	}
	set, _ = d.TombstonedIDs()
	if len(set) != 0 {
		t.Errorf("remaining = %d, want 0", len(set))
	}
}
