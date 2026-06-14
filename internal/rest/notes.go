package rest

import (
	"context"
	"errors"

	"github.com/Kerblif/Library/internal/api"
	"github.com/Kerblif/Library/internal/store"
)

func (s *Server) ListNotes(ctx context.Context, req api.ListNotesRequestObject) (api.ListNotesResponseObject, error) {
	f, err := noteFilter(req.Params)
	if err != nil {
		return api.ListNotes400JSONResponse{BadRequestJSONResponse: api.BadRequestJSONResponse{Code: "bad_request", Message: "invalid cursor"}}, nil
	}
	page, err := s.repo.ListNotes(ctx, f)
	if err != nil {
		return nil, err
	}
	return api.ListNotes200JSONResponse(toNoteList(page)), nil
}

func (s *Server) SubmitNoteOperation(ctx context.Context, req api.SubmitNoteOperationRequestObject) (api.SubmitNoteOperationResponseObject, error) {
	if req.Body == nil {
		return badSubmit("bad_request", "missing request body"), nil
	}
	op, err := req.Body.Discriminator()
	if err != nil {
		return badSubmit("bad_request", "missing or invalid op"), nil
	}

	switch op {
	case string(api.Create):
		c, err := req.Body.AsNoteCreate()
		if err != nil {
			return badSubmit("bad_request", "invalid create payload"), nil
		}
		return created(s.repo.CreateNote(ctx, store.CreateNoteInput{
			Title:     c.Title,
			Body:      c.Body,
			Category:  store.Category(c.Category),
			Tags:      derefTags(c.Tags),
			CreatedBy: c.CreatedBy,
		}))

	case string(api.SuggestEdit):
		e, err := req.Body.AsNoteSuggestEdit()
		if err != nil {
			return badSubmit("bad_request", "invalid suggest_edit payload"), nil
		}
		return created(s.repo.SuggestEdit(ctx, store.SuggestEditInput{
			TargetID:  e.TargetId,
			Title:     e.Title,
			Body:      e.Body,
			Tags:      derefTags(e.Tags),
			CreatedBy: e.CreatedBy,
		}))

	case string(api.Canonize):
		c, err := req.Body.AsNoteCanonize()
		if err != nil {
			return badSubmit("bad_request", "invalid canonize payload"), nil
		}
		return transitioned(s.repo.Canonize(ctx, c.NoteId, c.Actor))

	case string(api.Archive):
		a, err := req.Body.AsNoteArchive()
		if err != nil {
			return badSubmit("bad_request", "invalid archive payload"), nil
		}
		return transitioned(s.repo.Archive(ctx, a.NoteId, a.Actor))

	case string(api.Restore):
		r, err := req.Body.AsNoteRestore()
		if err != nil {
			return badSubmit("bad_request", "invalid restore payload"), nil
		}
		return transitioned(s.repo.Restore(ctx, r.NoteId, r.Actor))

	default:
		return badSubmit("bad_request", "unknown op: "+op), nil
	}
}

func (s *Server) GetNoteCounts(ctx context.Context, _ api.GetNoteCountsRequestObject) (api.GetNoteCountsResponseObject, error) {
	c, err := s.repo.CountNotes(ctx)
	if err != nil {
		return nil, err
	}
	return api.GetNoteCounts200JSONResponse(api.NoteCounts{
		Canon:           c.Canon,
		AiDraft:         c.AIDraft,
		AiSuggestedEdit: c.AISuggestedEdit,
	}), nil
}

func (s *Server) GetNote(ctx context.Context, req api.GetNoteRequestObject) (api.GetNoteResponseObject, error) {
	n, err := s.repo.GetNote(ctx, req.NoteId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.GetNote404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse{Code: "not_found", Message: "note not found"}}, nil
		}
		return nil, err
	}
	return api.GetNote200JSONResponse(toAPINote(n)), nil
}

func (s *Server) UpdateNote(ctx context.Context, req api.UpdateNoteRequestObject) (api.UpdateNoteResponseObject, error) {
	if req.Body == nil || (req.Body.Title == nil && req.Body.Body == nil && req.Body.Tags == nil) {
		return api.UpdateNote422JSONResponse{UnprocessableEntityJSONResponse: api.UnprocessableEntityJSONResponse{Code: "validation_failed", Message: "at least one field is required"}}, nil
	}

	in := store.UpdateNoteInput{ID: req.NoteId, Title: req.Body.Title, Body: req.Body.Body}
	if req.Body.Tags != nil {
		tags := []string(*req.Body.Tags)
		in.Tags = &tags
	}

	n, err := s.repo.UpdateNote(ctx, in)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			return api.UpdateNote404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse{Code: "not_found", Message: "note not found"}}, nil
		case errors.Is(err, store.ErrInvalid):
			return api.UpdateNote422JSONResponse{UnprocessableEntityJSONResponse: api.UnprocessableEntityJSONResponse{Code: "validation_failed", Message: "invalid note fields"}}, nil
		default:
			return nil, err
		}
	}
	return api.UpdateNote200JSONResponse(toAPINote(n)), nil
}

func (s *Server) DeleteNote(ctx context.Context, req api.DeleteNoteRequestObject) (api.DeleteNoteResponseObject, error) {
	if err := s.repo.DeleteNote(ctx, req.NoteId); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.DeleteNote404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse{Code: "not_found", Message: "note not found"}}, nil
		}
		return nil, err
	}
	return api.DeleteNote204Response{}, nil
}

// noteFilter builds a store filter; LinkedTo and Query are pointer aliases that
// assign straight through.
func noteFilter(p api.ListNotesParams) (store.NoteFilter, error) {
	f := store.NoteFilter{
		Limit:    20,
		LinkedTo: p.LinkedTo,
		Query:    p.Q,
		Archived: archivedFilter(p.Archived),
	}
	if p.Limit != nil {
		f.Limit = clamp(*p.Limit, 1, 100)
	}
	if p.Category != nil {
		c := store.Category(*p.Category)
		f.Category = &c
	}
	if p.Tag != nil {
		f.Tags = *p.Tag
	}
	if p.Cursor != nil && *p.Cursor != "" {
		c, err := store.DecodeCursor(*p.Cursor)
		if err != nil {
			return store.NoteFilter{}, err
		}
		f.Cursor = &c
	}
	return f, nil
}

// archivedFilter maps the enum to a tri-state flag: active by default, archived for "true", nil for "all".
func archivedFilter(v *api.ListNotesParamsArchived) *bool {
	val := api.ListNotesParamsArchivedFalse
	if v != nil {
		val = *v
	}
	switch val {
	case api.ListNotesParamsArchivedTrue:
		t := true
		return &t
	case api.ListNotesParamsArchivedAll:
		return nil
	default:
		f := false
		return &f
	}
}

func toNoteList(page store.NotesPage) api.NoteList {
	items := make([]api.Note, len(page.Items))
	for i, n := range page.Items {
		items[i] = toAPINote(n)
	}
	list := api.NoteList{Items: items}
	if page.Next != nil {
		token := page.Next.Encode()
		list.NextCursor = &token
	}
	return list
}

// created returns the new note as HTTP 201.
func created(n store.Note, err error) (api.SubmitNoteOperationResponseObject, error) {
	if err != nil {
		return submitError(err)
	}
	return api.SubmitNoteOperation201JSONResponse(toAPINote(n)), nil
}

// transitioned returns the changed note as HTTP 200.
func transitioned(n store.Note, err error) (api.SubmitNoteOperationResponseObject, error) {
	if err != nil {
		return submitError(err)
	}
	return api.SubmitNoteOperation200JSONResponse(toAPINote(n)), nil
}

func badSubmit(code, message string) api.SubmitNoteOperationResponseObject {
	return api.SubmitNoteOperation400JSONResponse{BadRequestJSONResponse: api.BadRequestJSONResponse{Code: code, Message: message}}
}

// submitError maps a repository error to the matching POST /notes response.
func submitError(err error) (api.SubmitNoteOperationResponseObject, error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.SubmitNoteOperation404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse{Code: "not_found", Message: "note not found"}}, nil
	case errors.Is(err, store.ErrConflict):
		return api.SubmitNoteOperation409JSONResponse{ConflictJSONResponse: api.ConflictJSONResponse{Code: "conflict", Message: "operation not allowed in the note's current state"}}, nil
	case errors.Is(err, store.ErrInvalid):
		return api.SubmitNoteOperation422JSONResponse{UnprocessableEntityJSONResponse: api.UnprocessableEntityJSONResponse{Code: "validation_failed", Message: "unprocessable operation"}}, nil
	default:
		return nil, err
	}
}
