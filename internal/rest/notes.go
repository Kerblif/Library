package rest

import (
	"context"

	"github.com/Kerblif/Library/internal/api"
)

func (s *Server) ListNotes(_ context.Context, _ api.ListNotesRequestObject) (api.ListNotesResponseObject, error) {
	return nil, ErrNotImplemented
}

func (s *Server) SubmitNoteOperation(_ context.Context, _ api.SubmitNoteOperationRequestObject) (api.SubmitNoteOperationResponseObject, error) {
	return nil, ErrNotImplemented
}

func (s *Server) GetNoteCounts(_ context.Context, _ api.GetNoteCountsRequestObject) (api.GetNoteCountsResponseObject, error) {
	return nil, ErrNotImplemented
}

func (s *Server) GetNote(_ context.Context, _ api.GetNoteRequestObject) (api.GetNoteResponseObject, error) {
	return nil, ErrNotImplemented
}

func (s *Server) UpdateNote(_ context.Context, _ api.UpdateNoteRequestObject) (api.UpdateNoteResponseObject, error) {
	return nil, ErrNotImplemented
}

func (s *Server) DeleteNote(_ context.Context, _ api.DeleteNoteRequestObject) (api.DeleteNoteResponseObject, error) {
	return nil, ErrNotImplemented
}
