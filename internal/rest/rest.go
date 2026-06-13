// Package rest implements the generated api.StrictServerInterface on top of a
// store.Repository. Handlers translate between the generated api types and the
// store domain types and map store sentinel errors to HTTP responses; they
// never import the generated db package directly.
package rest

import (
	"encoding/json"
	"net/http"

	"github.com/Kerblif/Library/internal/api"
	"github.com/Kerblif/Library/internal/store"
)

// Server implements api.StrictServerInterface backed by a repository.
type Server struct {
	repo store.Repository
}

// New builds the REST server over the given repository.
func New(repo store.Repository) *Server {
	return &Server{repo: repo}
}

// ServerInterface adapts the strict handlers into the chi-compatible
// api.ServerInterface and maps framework-level errors to JSON responses.
func (s *Server) ServerInterface() api.ServerInterface {
	return api.NewStrictHandlerWithOptions(s, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			writeError(w, http.StatusInternalServerError, "internal", "internal error")
		},
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.Error{Code: code, Message: message})
}

// toAPINote maps a store note to the wire representation.
func toAPINote(n store.Note) api.Note {
	id := n.ID
	created := n.CreatedAt
	updated := n.UpdatedAt
	tags := n.Tags
	if tags == nil {
		tags = []string{}
	}
	return api.Note{
		Id:          &id,
		Title:       n.Title,
		Body:        n.Body,
		Category:    api.Category(n.Category),
		Tags:        tags,
		TargetId:    n.TargetID,
		CreatedBy:   n.CreatedBy,
		ExpiresAt:   n.ExpiresAt,
		CanonizedAt: n.CanonizedAt,
		CanonizedBy: n.CanonizedBy,
		CreatedAt:   &created,
		UpdatedAt:   &updated,
		Archived:    n.Archived,
		ArchivedAt:  n.ArchivedAt,
		ArchivedBy:  n.ArchivedBy,
	}
}

// toAPILink maps a store link to the wire representation.
func toAPILink(l store.Link) api.Link {
	id := l.ID
	created := l.CreatedAt
	return api.Link{Id: &id, SourceId: l.SourceID, TargetId: l.TargetID, CreatedAt: &created}
}

// derefTags flattens an optional tag list to a plain slice (nil stays nil).
func derefTags(tags *[]api.TagName) []string {
	if tags == nil {
		return nil
	}
	return *tags
}

func clamp(v, lo, hi int) int {
	switch {
	case v < lo:
		return lo
	case v > hi:
		return hi
	default:
		return v
	}
}
