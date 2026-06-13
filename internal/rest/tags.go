package rest

import (
	"context"

	"github.com/Kerblif/Library/internal/api"
)

func (s *Server) ListTags(ctx context.Context, _ api.ListTagsRequestObject) (api.ListTagsResponseObject, error) {
	tags, err := s.repo.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]api.Tag, len(tags))
	for i, t := range tags {
		id := t.ID
		count := t.NoteCount
		items[i] = api.Tag{Id: &id, Name: t.Name, NoteCount: &count}
	}
	return api.ListTags200JSONResponse(api.TagList{Items: items}), nil
}
