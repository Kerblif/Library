package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type tagListDTO struct {
	Items []tagDTO `json:"items"`
}

func (h *handlers) listTags(ctx context.Context, _ *mcp.CallToolRequest, _ noArgs) (*mcp.CallToolResult, tagListDTO, error) {
	tags, err := h.repo.ListTags(ctx)
	if err != nil {
		return nil, tagListDTO{}, mapErr(err)
	}
	items := make([]tagDTO, len(tags))
	for i, t := range tags {
		items[i] = tagDTO{Name: t.Name, NoteCount: t.NoteCount}
	}
	return nil, tagListDTO{Items: items}, nil
}
