package rest

import (
	"context"

	"github.com/Kerblif/Library/internal/api"
)

func (s *Server) ListTags(_ context.Context, _ api.ListTagsRequestObject) (api.ListTagsResponseObject, error) {
	return nil, ErrNotImplemented
}
