// Package rest implements the generated api.StrictServerInterface.
//
// Handlers depend on a repository interface (injected here in the runtime pass),
// never on the generated db package directly — only the Postgres repository
// implementation imports internal/db. For now every operation is a stub that
// returns 501.
package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Kerblif/Library/internal/api"
)

// ErrNotImplemented is returned by every stub handler until the runtime pass
// wires real behaviour.
var ErrNotImplemented = errors.New("not implemented")

// Server implements api.StrictServerInterface. It will gain a repository
// interface field (not *db.Queries) once the runtime pass lands.
type Server struct{}

// New builds the REST server.
func New() *Server {
	return &Server{}
}

// ServerInterface adapts the strict handlers into the chi-compatible
// api.ServerInterface and maps handler errors to JSON responses.
func (s *Server) ServerInterface() api.ServerInterface {
	return api.NewStrictHandlerWithOptions(s, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			if errors.Is(err, ErrNotImplemented) {
				writeError(w, http.StatusNotImplemented, "not_implemented", "operation not implemented")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal", "internal error")
		},
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.Error{Code: code, Message: message})
}
