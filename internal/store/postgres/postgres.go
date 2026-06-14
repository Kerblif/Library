// Package postgres implements store.Repository. It is the only package that
// imports internal/db.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Kerblif/Library/internal/db"
	"github.com/Kerblif/Library/internal/store"
)

type Repo struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

var _ store.Repository = (*Repo)(nil)

func New(ctx context.Context, dsn string) (*Repo, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return &Repo{pool: pool, q: db.New(pool)}, nil
}

func (r *Repo) Close() { r.pool.Close() }

// inTx runs fn in a transaction, rolling back unless it returns nil and commits.
func (r *Repo) inTx(ctx context.Context, fn func(q *db.Queries) (store.Note, error)) (store.Note, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return store.Note{}, err
	}
	defer tx.Rollback(ctx)

	n, err := fn(r.q.WithTx(tx))
	if err != nil {
		return store.Note{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return store.Note{}, err
	}
	return n, nil
}

func (r *Repo) CreateNote(ctx context.Context, in store.CreateNoteInput) (store.Note, error) {
	return r.inTx(ctx, func(q *db.Queries) (store.Note, error) {
		n, err := q.CreateNote(ctx, db.CreateNoteParams{
			Title:     in.Title,
			Body:      in.Body,
			Category:  db.Category(in.Category),
			CreatedBy: in.CreatedBy,
		})
		if err != nil {
			return store.Note{}, mapErr(err)
		}
		if err := attachTags(ctx, q, n.ID, in.Tags); err != nil {
			return store.Note{}, err
		}
		return loadNote(ctx, q, n.ID)
	})
}

func (r *Repo) SuggestEdit(ctx context.Context, in store.SuggestEditInput) (store.Note, error) {
	return r.inTx(ctx, func(q *db.Queries) (store.Note, error) {
		target, err := q.GetNote(ctx, in.TargetID)
		if err != nil {
			return store.Note{}, mapErr(err)
		}
		if target.Category != db.CategoryCanon {
			// Edits can only be proposed against canon notes.
			return store.Note{}, store.ErrInvalid
		}

		title := target.Title
		if in.Title != nil {
			title = *in.Title
		}
		// Default tags to the target's so applying the suggestion never drops them.
		tags := in.Tags
		if len(tags) == 0 {
			tags = target.Tags
		}

		targetID := in.TargetID
		n, err := q.CreateNote(ctx, db.CreateNoteParams{
			Title:     title,
			Body:      in.Body,
			Category:  db.CategoryAiSuggestedEdit,
			TargetID:  &targetID,
			CreatedBy: in.CreatedBy,
		})
		if err != nil {
			return store.Note{}, mapErr(err)
		}
		if err := attachTags(ctx, q, n.ID, tags); err != nil {
			return store.Note{}, err
		}
		return loadNote(ctx, q, n.ID)
	})
}

func (r *Repo) Canonize(ctx context.Context, noteID uuid.UUID, actor string) (store.Note, error) {
	return r.inTx(ctx, func(q *db.Queries) (store.Note, error) {
		cur, err := q.GetNote(ctx, noteID)
		if err != nil {
			return store.Note{}, mapErr(err)
		}

		switch cur.Category {
		case db.CategoryAiDraft:
			if _, err := q.CanonizeDraft(ctx, db.CanonizeDraftParams{Actor: &actor, ID: noteID}); err != nil {
				return store.Note{}, mapErr(err)
			}
			if err := q.LogCanonization(ctx, db.LogCanonizationParams{NoteID: noteID, Actor: actor}); err != nil {
				return store.Note{}, mapErr(err)
			}
			return loadNote(ctx, q, noteID)

		case db.CategoryAiSuggestedEdit:
			tgt, err := q.ApplySuggestedEdit(ctx, db.ApplySuggestedEditParams{Actor: &actor, SuggestionID: noteID})
			if err != nil {
				return store.Note{}, mapErr(err)
			}
			// Move the suggestion's tags onto the target, then drop the suggestion.
			if err := q.ClearNoteTags(ctx, tgt.ID); err != nil {
				return store.Note{}, mapErr(err)
			}
			if err := attachTags(ctx, q, tgt.ID, cur.Tags); err != nil {
				return store.Note{}, err
			}
			if _, err := q.DeleteNote(ctx, noteID); err != nil {
				return store.Note{}, mapErr(err)
			}
			if err := q.LogCanonization(ctx, db.LogCanonizationParams{NoteID: tgt.ID, Actor: actor}); err != nil {
				return store.Note{}, mapErr(err)
			}
			return loadNote(ctx, q, tgt.ID)

		default: // already canon — nothing to commit
			return store.Note{}, store.ErrConflict
		}
	})
}

func (r *Repo) Archive(ctx context.Context, noteID uuid.UUID, actor string) (store.Note, error) {
	return r.inTx(ctx, func(q *db.Queries) (store.Note, error) {
		cur, err := q.GetNote(ctx, noteID)
		if err != nil {
			return store.Note{}, mapErr(err)
		}
		if cur.Category != db.CategoryCanon || cur.Archived {
			return store.Note{}, store.ErrConflict
		}
		if _, err := q.ArchiveNote(ctx, db.ArchiveNoteParams{Actor: &actor, ID: noteID}); err != nil {
			return store.Note{}, mapErr(err)
		}
		return loadNote(ctx, q, noteID)
	})
}

func (r *Repo) Restore(ctx context.Context, noteID uuid.UUID, _ string) (store.Note, error) {
	return r.inTx(ctx, func(q *db.Queries) (store.Note, error) {
		cur, err := q.GetNote(ctx, noteID)
		if err != nil {
			return store.Note{}, mapErr(err)
		}
		if !cur.Archived {
			return store.Note{}, store.ErrConflict
		}
		if _, err := q.RestoreNote(ctx, noteID); err != nil {
			return store.Note{}, mapErr(err)
		}
		return loadNote(ctx, q, noteID)
	})
}

func (r *Repo) GetNote(ctx context.Context, id uuid.UUID) (store.Note, error) {
	row, err := r.q.GetNote(ctx, id)
	if err != nil {
		return store.Note{}, mapErr(err)
	}
	return noteFromGet(row), nil
}

func (r *Repo) UpdateNote(ctx context.Context, in store.UpdateNoteInput) (store.Note, error) {
	return r.inTx(ctx, func(q *db.Queries) (store.Note, error) {
		// UpdateNote returns no row when the note does not exist.
		if _, err := q.UpdateNote(ctx, db.UpdateNoteParams{Title: in.Title, Body: in.Body, ID: in.ID}); err != nil {
			return store.Note{}, mapErr(err)
		}
		if in.Tags != nil {
			if err := q.ClearNoteTags(ctx, in.ID); err != nil {
				return store.Note{}, mapErr(err)
			}
			if err := attachTags(ctx, q, in.ID, *in.Tags); err != nil {
				return store.Note{}, err
			}
		}
		return loadNote(ctx, q, in.ID)
	})
}

func (r *Repo) DeleteNote(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.DeleteNote(ctx, id)
	if err != nil {
		return mapErr(err)
	}
	if n == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (r *Repo) ListNotes(ctx context.Context, f store.NoteFilter) (store.NotesPage, error) {
	params := db.ListNotesParams{
		Category:  catPtr(f.Category),
		Archived:  f.Archived,
		LinkedTo:  f.LinkedTo,
		Q:         f.Query,
		Tags:      f.Tags,
		PageLimit: int32(f.Limit) + 1, // fetch one extra to detect a next page
	}
	if f.Cursor != nil {
		params.CursorUpdatedAt = &f.Cursor.UpdatedAt
		params.CursorID = &f.Cursor.ID
	}

	rows, err := r.q.ListNotes(ctx, params)
	if err != nil {
		return store.NotesPage{}, mapErr(err)
	}

	var next *store.Cursor
	if len(rows) > f.Limit {
		last := rows[f.Limit-1]
		next = &store.Cursor{UpdatedAt: last.UpdatedAt, ID: last.ID}
		rows = rows[:f.Limit]
	}

	items := make([]store.Note, len(rows))
	for i, row := range rows {
		items[i] = noteFromList(row)
	}
	return store.NotesPage{Items: items, Next: next}, nil
}

func (r *Repo) CountNotes(ctx context.Context) (store.Counts, error) {
	row, err := r.q.CountNotesByCategory(ctx)
	if err != nil {
		return store.Counts{}, mapErr(err)
	}
	return store.Counts{
		Canon:           int(row.Canon),
		AIDraft:         int(row.AiDraft),
		AISuggestedEdit: int(row.AiSuggestedEdit),
	}, nil
}

func (r *Repo) NoteLinks(ctx context.Context, noteID uuid.UUID) (store.NoteLinks, error) {
	if _, err := r.q.GetNote(ctx, noteID); err != nil {
		return store.NoteLinks{}, mapErr(err)
	}
	outgoing, err := r.q.ListOutgoingLinks(ctx, noteID)
	if err != nil {
		return store.NoteLinks{}, mapErr(err)
	}
	incoming, err := r.q.ListIncomingLinks(ctx, noteID)
	if err != nil {
		return store.NoteLinks{}, mapErr(err)
	}
	return store.NoteLinks{Incoming: links(incoming), Outgoing: links(outgoing)}, nil
}

func (r *Repo) CreateLink(ctx context.Context, sourceID, targetID uuid.UUID) (store.Link, error) {
	if sourceID == targetID {
		return store.Link{}, store.ErrInvalid
	}
	l, err := r.q.CreateLink(ctx, db.CreateLinkParams{SourceID: sourceID, TargetID: targetID})
	if err != nil {
		return store.Link{}, mapErr(err)
	}
	return link(l), nil
}

func (r *Repo) DeleteLink(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.DeleteLink(ctx, id)
	if err != nil {
		return mapErr(err)
	}
	if n == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (r *Repo) ListTags(ctx context.Context) ([]store.Tag, error) {
	rows, err := r.q.ListTags(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	tags := make([]store.Tag, len(rows))
	for i, row := range rows {
		tags[i] = store.Tag{ID: row.ID, Name: row.Name, NoteCount: int(row.NoteCount)}
	}
	return tags, nil
}

func attachTags(ctx context.Context, q *db.Queries, noteID uuid.UUID, names []string) error {
	if len(names) == 0 {
		return nil
	}
	if err := q.AttachTags(ctx, db.AttachTagsParams{NoteID: noteID, Names: names}); err != nil {
		return mapErr(err)
	}
	return nil
}

// loadNote re-reads a note with its tags after a mutation.
func loadNote(ctx context.Context, q *db.Queries, id uuid.UUID) (store.Note, error) {
	row, err := q.GetNote(ctx, id)
	if err != nil {
		return store.Note{}, mapErr(err)
	}
	return noteFromGet(row), nil
}

// mapErr translates pgx / Postgres errors into store sentinels.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return store.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return store.ErrConflict
		case "23503": // foreign_key_violation
			return store.ErrNotFound
		case "23514", "23502", "22001": // check / not-null / string-too-long
			return store.ErrInvalid
		}
	}
	return err
}

func catPtr(c *store.Category) *db.Category {
	if c == nil {
		return nil
	}
	d := db.Category(*c)
	return &d
}

func noteFromGet(r db.GetNoteRow) store.Note {
	return store.Note{
		ID:          r.ID,
		Title:       r.Title,
		Body:        r.Body,
		Category:    store.Category(r.Category),
		Tags:        r.Tags,
		TargetID:    r.TargetID,
		CreatedBy:   r.CreatedBy,
		ExpiresAt:   r.ExpiresAt,
		CanonizedAt: r.CanonizedAt,
		CanonizedBy: r.CanonizedBy,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		Archived:    r.Archived,
		ArchivedAt:  r.ArchivedAt,
		ArchivedBy:  r.ArchivedBy,
	}
}

func noteFromList(r db.ListNotesRow) store.Note {
	return store.Note{
		ID:          r.ID,
		Title:       r.Title,
		Body:        r.Body,
		Category:    store.Category(r.Category),
		Tags:        r.Tags,
		TargetID:    r.TargetID,
		CreatedBy:   r.CreatedBy,
		ExpiresAt:   r.ExpiresAt,
		CanonizedAt: r.CanonizedAt,
		CanonizedBy: r.CanonizedBy,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		Archived:    r.Archived,
		ArchivedAt:  r.ArchivedAt,
		ArchivedBy:  r.ArchivedBy,
	}
}

func link(l db.Link) store.Link {
	return store.Link{ID: l.ID, SourceID: l.SourceID, TargetID: l.TargetID, CreatedAt: l.CreatedAt}
}

func links(ls []db.Link) []store.Link {
	out := make([]store.Link, len(ls))
	for i, l := range ls {
		out[i] = link(l)
	}
	return out
}
