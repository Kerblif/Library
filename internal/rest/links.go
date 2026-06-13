package rest

import (
	"context"
	"errors"

	"github.com/Kerblif/Library/internal/api"
	"github.com/Kerblif/Library/internal/store"
)

func (s *Server) CreateLink(ctx context.Context, req api.CreateLinkRequestObject) (api.CreateLinkResponseObject, error) {
	if req.Body == nil {
		return api.CreateLink400JSONResponse{BadRequestJSONResponse: api.BadRequestJSONResponse{Code: "bad_request", Message: "missing request body"}}, nil
	}

	l, err := s.repo.CreateLink(ctx, req.Body.SourceId, req.Body.TargetId)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			return api.CreateLink404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse{Code: "not_found", Message: "source or target note not found"}}, nil
		case errors.Is(err, store.ErrConflict):
			return api.CreateLink409JSONResponse{ConflictJSONResponse: api.ConflictJSONResponse{Code: "conflict", Message: "link already exists"}}, nil
		case errors.Is(err, store.ErrInvalid):
			return api.CreateLink422JSONResponse{UnprocessableEntityJSONResponse: api.UnprocessableEntityJSONResponse{Code: "validation_failed", Message: "source and target must differ"}}, nil
		default:
			return nil, err
		}
	}
	return api.CreateLink201JSONResponse(toAPILink(l)), nil
}

func (s *Server) DeleteLink(ctx context.Context, req api.DeleteLinkRequestObject) (api.DeleteLinkResponseObject, error) {
	if err := s.repo.DeleteLink(ctx, req.LinkId); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.DeleteLink404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse{Code: "not_found", Message: "link not found"}}, nil
		}
		return nil, err
	}
	return api.DeleteLink204Response{}, nil
}

func (s *Server) ListNoteLinks(ctx context.Context, req api.ListNoteLinksRequestObject) (api.ListNoteLinksResponseObject, error) {
	nl, err := s.repo.NoteLinks(ctx, req.NoteId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.ListNoteLinks404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse{Code: "not_found", Message: "note not found"}}, nil
		}
		return nil, err
	}

	incoming := make([]api.Link, len(nl.Incoming))
	for i, l := range nl.Incoming {
		incoming[i] = toAPILink(l)
	}
	outgoing := make([]api.Link, len(nl.Outgoing))
	for i, l := range nl.Outgoing {
		outgoing[i] = toAPILink(l)
	}
	return api.ListNoteLinks200JSONResponse(api.NoteLinks{Incoming: incoming, Outgoing: outgoing}), nil
}
