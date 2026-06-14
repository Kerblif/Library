package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listNoteLinksInput struct {
	NoteID string `json:"note_id" jsonschema:"the note's UUID"`
}

func (h *handlers) listNoteLinks(ctx context.Context, _ *mcp.CallToolRequest, in listNoteLinksInput) (*mcp.CallToolResult, noteLinksDTO, error) {
	id, err := parseUUID("note_id", in.NoteID)
	if err != nil {
		return nil, noteLinksDTO{}, err
	}
	nl, err := h.repo.NoteLinks(ctx, id)
	if err != nil {
		return nil, noteLinksDTO{}, mapErr(err)
	}
	return nil, noteLinksDTO{Incoming: toLinkDTOs(nl.Incoming), Outgoing: toLinkDTOs(nl.Outgoing)}, nil
}

type createLinkInput struct {
	SourceID string `json:"source_id" jsonschema:"UUID of the note the link starts from"`
	TargetID string `json:"target_id" jsonschema:"UUID of the note the link points to"`
}

func (h *handlers) createLink(ctx context.Context, _ *mcp.CallToolRequest, in createLinkInput) (*mcp.CallToolResult, linkDTO, error) {
	source, err := parseUUID("source_id", in.SourceID)
	if err != nil {
		return nil, linkDTO{}, err
	}
	target, err := parseUUID("target_id", in.TargetID)
	if err != nil {
		return nil, linkDTO{}, err
	}
	l, err := h.repo.CreateLink(ctx, source, target)
	if err != nil {
		return nil, linkDTO{}, mapErr(err)
	}
	return nil, toLinkDTO(l), nil
}

type deleteLinkInput struct {
	LinkID string `json:"link_id" jsonschema:"UUID of the link to delete"`
}

type deleteResult struct {
	Deleted bool `json:"deleted"`
}

func (h *handlers) deleteLink(ctx context.Context, _ *mcp.CallToolRequest, in deleteLinkInput) (*mcp.CallToolResult, deleteResult, error) {
	id, err := parseUUID("link_id", in.LinkID)
	if err != nil {
		return nil, deleteResult{}, err
	}
	if err := h.repo.DeleteLink(ctx, id); err != nil {
		return nil, deleteResult{}, mapErr(err)
	}
	return nil, deleteResult{Deleted: true}, nil
}
