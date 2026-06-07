package rest

import (
	"context"

	"github.com/Kerblif/Library/internal/api"
)

func (s *Server) CreateLink(_ context.Context, _ api.CreateLinkRequestObject) (api.CreateLinkResponseObject, error) {
	return nil, ErrNotImplemented
}

func (s *Server) DeleteLink(_ context.Context, _ api.DeleteLinkRequestObject) (api.DeleteLinkResponseObject, error) {
	return nil, ErrNotImplemented
}

func (s *Server) ListNoteLinks(_ context.Context, _ api.ListNoteLinksRequestObject) (api.ListNoteLinksResponseObject, error) {
	return nil, ErrNotImplemented
}
