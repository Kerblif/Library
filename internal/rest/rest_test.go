package rest

import (
	"testing"

	"github.com/Kerblif/Library/internal/api"
)

func TestClamp(t *testing.T) {
	cases := []struct{ v, lo, hi, want int }{
		{0, 1, 100, 1},
		{20, 1, 100, 20},
		{500, 1, 100, 100},
		{1, 1, 100, 1},
		{100, 1, 100, 100},
	}
	for _, c := range cases {
		if got := clamp(c.v, c.lo, c.hi); got != c.want {
			t.Errorf("clamp(%d, %d, %d) = %d, want %d", c.v, c.lo, c.hi, got, c.want)
		}
	}
}

func TestArchivedFilter(t *testing.T) {
	mustBool := func(name string, got *bool, want bool) {
		if got == nil || *got != want {
			t.Errorf("archivedFilter(%s) = %v, want *%v", name, got, want)
		}
	}
	mustBool("nil", archivedFilter(nil), false)

	tv := api.ListNotesParamsArchivedTrue
	mustBool("true", archivedFilter(&tv), true)

	fv := api.ListNotesParamsArchivedFalse
	mustBool("false", archivedFilter(&fv), false)

	av := api.ListNotesParamsArchivedAll
	if got := archivedFilter(&av); got != nil {
		t.Errorf("archivedFilter(all) = %v, want nil", got)
	}
}

func TestNoteFilterDefaultsAndCursor(t *testing.T) {
	f, err := noteFilter(api.ListNotesParams{})
	if err != nil {
		t.Fatalf("noteFilter(empty): %v", err)
	}
	if f.Limit != 20 {
		t.Errorf("default Limit = %d, want 20", f.Limit)
	}
	if f.Archived == nil || *f.Archived != false {
		t.Errorf("default Archived = %v, want *false", f.Archived)
	}
	if f.Cursor != nil {
		t.Errorf("default Cursor = %v, want nil", f.Cursor)
	}

	bad := "not-a-cursor !!!"
	if _, err := noteFilter(api.ListNotesParams{Cursor: &bad}); err == nil {
		t.Error("noteFilter with bad cursor: got nil error, want error")
	}
}

func TestDerefTags(t *testing.T) {
	if got := derefTags(nil); got != nil {
		t.Errorf("derefTags(nil) = %v, want nil", got)
	}
	tags := []api.TagName{"a", "b"}
	got := derefTags(&tags)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("derefTags(&[a b]) = %v, want [a b]", got)
	}
}
