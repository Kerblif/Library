package mcpserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Kerblif/Library/internal/store"
)

type listNotesInput struct {
	Query    *string  `json:"query,omitempty" jsonschema:"full-text search over title and body"`
	Category *string  `json:"category,omitempty" jsonschema:"filter by category: canon, ai_draft, or ai_suggested_edit"`
	Tags     []string `json:"tags,omitempty" jsonschema:"notes must carry all of these tags"`
	LinkedTo *string  `json:"linked_to,omitempty" jsonschema:"restrict to notes linked to or from this note UUID"`
	Archived *string  `json:"archived,omitempty" jsonschema:"visibility: false (default, active only), true (archived only), or all"`
	Limit    *int     `json:"limit,omitempty" jsonschema:"page size, 1-100 (default 20)"`
	Cursor   *string  `json:"cursor,omitempty" jsonschema:"next_cursor from a previous page"`
}

func (h *handlers) listNotes(ctx context.Context, _ *mcp.CallToolRequest, in listNotesInput) (*mcp.CallToolResult, noteListDTO, error) {
	f, err := h.buildFilter(in)
	if err != nil {
		return nil, noteListDTO{}, err
	}
	page, err := h.repo.ListNotes(ctx, f)
	if err != nil {
		return nil, noteListDTO{}, mapErr(err)
	}

	items := make([]noteDTO, len(page.Items))
	for i, n := range page.Items {
		items[i] = toNoteDTO(n)
	}
	out := noteListDTO{Items: items}
	if page.Next != nil {
		token := page.Next.Encode()
		out.NextCursor = &token
	}
	return nil, out, nil
}

func (h *handlers) buildFilter(in listNotesInput) (store.NoteFilter, error) {
	f := store.NoteFilter{Limit: 20}
	if in.Limit != nil {
		f.Limit = clamp(*in.Limit, 1, 100)
	}
	if in.Category != nil {
		c := store.Category(*in.Category)
		switch c {
		case store.CategoryCanon, store.CategoryAIDraft, store.CategoryAISuggestedEdit:
			f.Category = &c
		default:
			return store.NoteFilter{}, fmt.Errorf("invalid category %q (want canon, ai_draft, or ai_suggested_edit)", *in.Category)
		}
	}

	archived := "false"
	if in.Archived != nil {
		archived = *in.Archived
	}
	switch archived {
	case "false":
		v := false
		f.Archived = &v
	case "true":
		v := true
		f.Archived = &v
	case "all":
		f.Archived = nil
	default:
		return store.NoteFilter{}, fmt.Errorf("invalid archived %q (want false, true, or all)", archived)
	}

	f.Tags = in.Tags
	f.Query = in.Query
	if in.LinkedTo != nil {
		id, err := parseUUID("linked_to", *in.LinkedTo)
		if err != nil {
			return store.NoteFilter{}, err
		}
		f.LinkedTo = &id
	}
	if in.Cursor != nil && *in.Cursor != "" {
		c, err := store.DecodeCursor(*in.Cursor)
		if err != nil {
			return store.NoteFilter{}, errors.New("invalid cursor")
		}
		f.Cursor = &c
	}
	return f, nil
}

type getNoteInput struct {
	NoteID string `json:"note_id" jsonschema:"the note's UUID"`
}

func (h *handlers) getNote(ctx context.Context, _ *mcp.CallToolRequest, in getNoteInput) (*mcp.CallToolResult, noteDTO, error) {
	id, err := parseUUID("note_id", in.NoteID)
	if err != nil {
		return nil, noteDTO{}, err
	}
	n, err := h.repo.GetNote(ctx, id)
	if err != nil {
		return nil, noteDTO{}, mapErr(err)
	}
	return nil, toNoteDTO(n), nil
}

func (h *handlers) getNoteCounts(ctx context.Context, _ *mcp.CallToolRequest, _ noArgs) (*mcp.CallToolResult, countsDTO, error) {
	c, err := h.repo.CountNotes(ctx)
	if err != nil {
		return nil, countsDTO{}, mapErr(err)
	}
	return nil, countsDTO{Canon: c.Canon, AIDraft: c.AIDraft, AISuggestedEdit: c.AISuggestedEdit}, nil
}

type createDraftInput struct {
	Title string   `json:"title" jsonschema:"note title, 1-200 characters"`
	Body  string   `json:"body" jsonschema:"markdown body"`
	Tags  []string `json:"tags,omitempty" jsonschema:"lowercase-slug tags, e.g. [\"go\", \"postgres\"]"`
}

func (h *handlers) createDraft(ctx context.Context, _ *mcp.CallToolRequest, in createDraftInput) (*mcp.CallToolResult, noteDTO, error) {
	n, err := h.repo.CreateNote(ctx, store.CreateNoteInput{
		Title:     in.Title,
		Body:      in.Body,
		Category:  store.CategoryAIDraft, // assistant can only create drafts
		Tags:      in.Tags,
		CreatedBy: &h.actor,
	})
	if err != nil {
		return nil, noteDTO{}, mapErr(err)
	}
	return nil, toNoteDTO(n), nil
}

type suggestEditInput struct {
	TargetID string   `json:"target_id" jsonschema:"UUID of the canon note to amend"`
	Body     string   `json:"body" jsonschema:"proposed markdown body"`
	Title    *string  `json:"title,omitempty" jsonschema:"proposed title; defaults to the target's current title"`
	Tags     []string `json:"tags,omitempty" jsonschema:"proposed tags; defaults to the target's current tags"`
}

func (h *handlers) suggestEdit(ctx context.Context, _ *mcp.CallToolRequest, in suggestEditInput) (*mcp.CallToolResult, noteDTO, error) {
	target, err := parseUUID("target_id", in.TargetID)
	if err != nil {
		return nil, noteDTO{}, err
	}
	n, err := h.repo.SuggestEdit(ctx, store.SuggestEditInput{
		TargetID:  target,
		Body:      in.Body,
		Title:     in.Title,
		Tags:      in.Tags,
		CreatedBy: &h.actor,
	})
	if err != nil {
		return nil, noteDTO{}, mapErr(err)
	}
	return nil, toNoteDTO(n), nil
}

type updateNoteInput struct {
	NoteID string    `json:"note_id" jsonschema:"UUID of the draft or suggested-edit note to update"`
	Title  *string   `json:"title,omitempty" jsonschema:"new title, 1-200 characters"`
	Body   *string   `json:"body,omitempty" jsonschema:"new markdown body"`
	Tags   *[]string `json:"tags,omitempty" jsonschema:"replacement tag set (replaces all existing tags)"`
}

func (h *handlers) updateNote(ctx context.Context, _ *mcp.CallToolRequest, in updateNoteInput) (*mcp.CallToolResult, noteDTO, error) {
	id, err := parseUUID("note_id", in.NoteID)
	if err != nil {
		return nil, noteDTO{}, err
	}
	if in.Title == nil && in.Body == nil && in.Tags == nil {
		return nil, noteDTO{}, errors.New("provide at least one of title, body, or tags")
	}

	// Canon notes are off-limits to the assistant; it must propose edits instead.
	cur, err := h.repo.GetNote(ctx, id)
	if err != nil {
		return nil, noteDTO{}, mapErr(err)
	}
	if cur.Category == store.CategoryCanon {
		return nil, noteDTO{}, errors.New("canon notes cannot be edited via MCP; use suggest_edit to propose a change")
	}

	n, err := h.repo.UpdateNote(ctx, store.UpdateNoteInput{ID: id, Title: in.Title, Body: in.Body, Tags: in.Tags})
	if err != nil {
		return nil, noteDTO{}, mapErr(err)
	}
	return nil, toNoteDTO(n), nil
}
