// Package mcpserver exposes the assistant-safe subset of the Library API as MCP
// tools over a store.Repository. It omits the human-only operations (canonize,
// archive, restore, hard delete, creating canon notes) so the assistant can only
// produce ai_draft / ai_suggested_edit — the invariant the OpenAPI spec assigns to
// the MCP layer. Depends only on the store port, never on the HTTP api package.
package mcpserver

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Kerblif/Library/internal/store"
)

// Handler returns the MCP server as a streamable-HTTP handler for the main router
// to mount. The tool set is registered once and reused across requests.
func Handler(repo store.Repository, actor string) http.Handler {
	server := newServer(repo, actor)
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
}

type handlers struct {
	repo store.Repository
	// actor is stamped as created_by on notes the assistant authors.
	actor string
}

func newServer(repo store.Repository, actor string) *mcp.Server {
	h := &handlers{repo: repo, actor: actor}
	server := mcp.NewServer(&mcp.Implementation{Name: "library", Version: "0.1.0"}, nil)

	// Read surface.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_notes",
		Description: "List notes on the knowledge board, newest first. Use to browse or search before answering from the board. Supports full-text search (query), tag and category filters, and keyset pagination via cursor. Returns a page of notes and an optional next_cursor.",
	}, h.listNotes)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_note",
		Description: "Fetch one note by id, including its body and tags. Use after list_notes to read a note in full.",
	}, h.getNote)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_note_counts",
		Description: "Active-note counts per category (canon, ai_draft, ai_suggested_edit). Use to gauge how much draft/suggested work is pending review.",
	}, h.getNoteCounts)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_note_links",
		Description: "List the links touching a note, split into incoming and outgoing edges. Use to explore how a note connects to the rest of the board.",
	}, h.listNoteLinks)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_tags",
		Description: "List every tag in use with how many notes carry it. Use to discover the board's vocabulary before tagging or filtering.",
	}, h.listTags)

	// Write surface — assistant-safe only.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_draft",
		Description: "Create a new draft note (category ai_draft) authored by the assistant. Drafts are NOT canon — a human reviews and canonizes them later. Use this to capture new knowledge; you cannot create canon notes.",
	}, h.createDraft)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "suggest_edit",
		Description: "Propose a change to an existing canon note. Creates an ai_suggested_edit note that targets it WITHOUT modifying the canon note; a human applies it later. Use this instead of editing a canon note directly. Title defaults to the target's title when omitted.",
	}, h.suggestEdit)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_note",
		Description: "Edit a note you authored in place (draft or suggested edit). Canon notes cannot be edited here — use suggest_edit for those. Provide at least one of title, body, or tags; tags replace the existing set.",
	}, h.updateNote)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_link",
		Description: "Create a directed link from one note to another. Both notes must exist, must differ, and the edge must not already exist.",
	}, h.createLink)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_link",
		Description: "Delete a link by id. Use to remove an edge you created in error.",
	}, h.deleteLink)

	return server
}

type noArgs struct{}

type noteDTO struct {
	ID          string     `json:"id" jsonschema:"note UUID"`
	Title       string     `json:"title"`
	Body        string     `json:"body" jsonschema:"markdown body"`
	Category    string     `json:"category" jsonschema:"canon, ai_draft, or ai_suggested_edit"`
	Tags        []string   `json:"tags"`
	TargetID    *string    `json:"target_id,omitempty" jsonschema:"for ai_suggested_edit, the canon note this edit targets"`
	CreatedBy   *string    `json:"created_by,omitempty"`
	CanonizedAt *time.Time `json:"canonized_at,omitempty"`
	CanonizedBy *string    `json:"canonized_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Archived    bool       `json:"archived"`
}

type linkDTO struct {
	ID        string    `json:"id"`
	SourceID  string    `json:"source_id"`
	TargetID  string    `json:"target_id"`
	CreatedAt time.Time `json:"created_at"`
}

type noteListDTO struct {
	Items      []noteDTO `json:"items"`
	NextCursor *string   `json:"next_cursor,omitempty" jsonschema:"pass as cursor to fetch the next page; absent on the last page"`
}

type noteLinksDTO struct {
	Incoming []linkDTO `json:"incoming"`
	Outgoing []linkDTO `json:"outgoing"`
}

type tagDTO struct {
	Name      string `json:"name"`
	NoteCount int    `json:"note_count"`
}

type countsDTO struct {
	Canon           int `json:"canon"`
	AIDraft         int `json:"ai_draft"`
	AISuggestedEdit int `json:"ai_suggested_edit"`
}

func toNoteDTO(n store.Note) noteDTO {
	tags := n.Tags
	if tags == nil {
		tags = []string{}
	}
	d := noteDTO{
		ID:          n.ID.String(),
		Title:       n.Title,
		Body:        n.Body,
		Category:    string(n.Category),
		Tags:        tags,
		CreatedBy:   n.CreatedBy,
		CanonizedAt: n.CanonizedAt,
		CanonizedBy: n.CanonizedBy,
		CreatedAt:   n.CreatedAt,
		UpdatedAt:   n.UpdatedAt,
		Archived:    n.Archived,
	}
	if n.TargetID != nil {
		s := n.TargetID.String()
		d.TargetID = &s
	}
	return d
}

func toLinkDTO(l store.Link) linkDTO {
	return linkDTO{ID: l.ID.String(), SourceID: l.SourceID.String(), TargetID: l.TargetID.String(), CreatedAt: l.CreatedAt}
}

func toLinkDTOs(ls []store.Link) []linkDTO {
	out := make([]linkDTO, len(ls))
	for i, l := range ls {
		out[i] = toLinkDTO(l)
	}
	return out
}

// mapErr turns a store sentinel into a model-readable tool error.
func mapErr(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return errors.New("not found")
	case errors.Is(err, store.ErrConflict):
		return errors.New("conflict: operation is not allowed in the note's current state")
	case errors.Is(err, store.ErrInvalid):
		return errors.New("invalid: the request is not acceptable for this note")
	default:
		return errors.New("internal error")
	}
}

func parseUUID(field, s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid %s: must be a UUID", field)
	}
	return id, nil
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
