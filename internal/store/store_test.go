package store

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCursorRoundTrip(t *testing.T) {
	want := Cursor{
		UpdatedAt: time.Date(2026, 6, 13, 12, 0, 0, 123456789, time.UTC),
		ID:        uuid.MustParse("11111111-2222-3333-4444-555555555555"),
	}

	got, err := DecodeCursor(want.Encode())
	if err != nil {
		t.Fatalf("DecodeCursor: %v", err)
	}
	if !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, want.UpdatedAt)
	}
	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}
}

func TestDecodeCursorRejectsGarbage(t *testing.T) {
	for _, s := range []string{"not base64 !!!", "", "Zm9vYmFy"} {
		if _, err := DecodeCursor(s); err == nil {
			t.Errorf("DecodeCursor(%q) = nil error, want error", s)
		}
	}
}
