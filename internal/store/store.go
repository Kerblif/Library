// Package store defines the persistence port for the Library domain: the
// domain types the HTTP and MCP layers speak, the Repository interface they
// depend on, and sentinel errors that callers map to transport status codes.
//
// It is deliberately free of any database or transport dependency — the
// Postgres adapter (internal/store/postgres) is the only thing that imports
// internal/db, and internal/rest maps between these types and the generated
// api package.
package store

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors. Adapters return these; transport layers map them to status
// codes (404, 409, 422). Any other error is an internal failure (500).
var (
	// ErrNotFound is returned when a referenced entity does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict is returned when an operation is invalid for the entity's
	// current state (e.g. canonizing a note that is already canon).
	ErrConflict = errors.New("conflict")
	// ErrInvalid is returned when the request is semantically unprocessable
	// (e.g. a self-link, or a constraint the database rejects).
	ErrInvalid = errors.New("invalid")
)

// Category mirrors the note category enum.
type Category string

const (
	CategoryCanon           Category = "canon"
	CategoryAIDraft         Category = "ai_draft"
	CategoryAISuggestedEdit Category = "ai_suggested_edit"
)

// Note is a knowledge-board note with its resolved tag set.
type Note struct {
	ID          uuid.UUID
	Title       string
	Body        string
	Category    Category
	Tags        []string
	TargetID    *uuid.UUID
	CreatedBy   *string
	ExpiresAt   *time.Time
	CanonizedAt *time.Time
	CanonizedBy *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Archived    bool
	ArchivedAt  *time.Time
	ArchivedBy  *string
}

// Link is a directed edge between two notes.
type Link struct {
	ID        uuid.UUID
	SourceID  uuid.UUID
	TargetID  uuid.UUID
	CreatedAt time.Time
}

// NoteLinks splits a note's edges by direction.
type NoteLinks struct {
	Incoming []Link
	Outgoing []Link
}

// Tag is a tag in use with the number of notes that carry it.
type Tag struct {
	ID        uuid.UUID
	Name      string
	NoteCount int
}

// Counts holds active-note counts per category.
type Counts struct {
	Canon           int
	AIDraft         int
	AISuggestedEdit int
}

// CreateNoteInput creates a fresh canon or ai_draft note.
type CreateNoteInput struct {
	Title     string
	Body      string
	Category  Category
	Tags      []string
	CreatedBy *string
}

// SuggestEditInput proposes a change to a canon note, producing an
// ai_suggested_edit that targets it. Title defaults to the target's title.
type SuggestEditInput struct {
	TargetID  uuid.UUID
	Title     *string
	Body      string
	Tags      []string
	CreatedBy *string
}

// UpdateNoteInput is a partial in-place edit. A nil field is left unchanged; a
// non-nil Tags (even empty) replaces the note's tag set.
type UpdateNoteInput struct {
	ID    uuid.UUID
	Title *string
	Body  *string
	Tags  *[]string
}

// NoteFilter selects and paginates notes. A nil pointer disables that filter.
// Archived nil means "all"; Cursor nil means the first page.
type NoteFilter struct {
	Category *Category
	Archived *bool
	Tags     []string
	LinkedTo *uuid.UUID
	Query    *string
	Limit    int
	Cursor   *Cursor
}

// NotesPage is one keyset page of notes. Next is nil on the last page.
type NotesPage struct {
	Items []Note
	Next  *Cursor
}

// Cursor is the keyset position for note pagination: the (updated_at, id) of
// the last row of a page. Notes are ordered by (updated_at, id) descending.
type Cursor struct {
	UpdatedAt time.Time
	ID        uuid.UUID
}

type cursorWire struct {
	U time.Time `json:"u"`
	I uuid.UUID `json:"i"`
}

// Encode renders the cursor as an opaque, URL-safe token.
func (c Cursor) Encode() string {
	b, _ := json.Marshal(cursorWire{U: c.UpdatedAt, I: c.ID})
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeCursor parses a token produced by Cursor.Encode.
func DecodeCursor(s string) (Cursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, err
	}
	var w cursorWire
	if err := json.Unmarshal(b, &w); err != nil {
		return Cursor{}, err
	}
	return Cursor{UpdatedAt: w.U, ID: w.I}, nil
}

// Repository is the persistence port the service depends on.
type Repository interface {
	CreateNote(ctx context.Context, in CreateNoteInput) (Note, error)
	SuggestEdit(ctx context.Context, in SuggestEditInput) (Note, error)
	Canonize(ctx context.Context, noteID uuid.UUID, actor string) (Note, error)
	Archive(ctx context.Context, noteID uuid.UUID, actor string) (Note, error)
	Restore(ctx context.Context, noteID uuid.UUID, actor string) (Note, error)

	GetNote(ctx context.Context, id uuid.UUID) (Note, error)
	UpdateNote(ctx context.Context, in UpdateNoteInput) (Note, error)
	DeleteNote(ctx context.Context, id uuid.UUID) error
	ListNotes(ctx context.Context, f NoteFilter) (NotesPage, error)
	CountNotes(ctx context.Context) (Counts, error)

	NoteLinks(ctx context.Context, noteID uuid.UUID) (NoteLinks, error)
	CreateLink(ctx context.Context, sourceID, targetID uuid.UUID) (Link, error)
	DeleteLink(ctx context.Context, id uuid.UUID) error

	ListTags(ctx context.Context) ([]Tag, error)
}
