-- name: GetNote :one
SELECT
    n.id, n.title, n.body, n.category, n.target_id, n.created_by,
    n.expires_at, n.canonized_at, n.canonized_by, n.created_at, n.updated_at,
    COALESCE(ARRAY(
        SELECT t.name FROM note_tags nt
        JOIN tags t ON t.id = nt.tag_id
        WHERE nt.note_id = n.id
        ORDER BY t.name
    ), ARRAY[]::text[])::text[] AS tags
FROM notes n
WHERE n.id = @id;

-- name: ListNotes :many
-- Every filter is optional; a NULL argument disables that clause. Pagination is
-- keyset on (updated_at, id) descending — pass the last row of the previous page
-- as the cursor.
SELECT
    n.id, n.title, n.body, n.category, n.target_id, n.created_by,
    n.expires_at, n.canonized_at, n.canonized_by, n.created_at, n.updated_at,
    COALESCE(ARRAY(
        SELECT t.name FROM note_tags nt
        JOIN tags t ON t.id = nt.tag_id
        WHERE nt.note_id = n.id
        ORDER BY t.name
    ), ARRAY[]::text[])::text[] AS tags
FROM notes n
WHERE
    (sqlc.narg('category')::category IS NULL OR n.category = sqlc.narg('category')::category)
    AND (sqlc.narg('linked_to')::uuid IS NULL OR EXISTS (
        SELECT 1 FROM links l
        WHERE (l.source_id = n.id AND l.target_id = sqlc.narg('linked_to')::uuid)
           OR (l.target_id = n.id AND l.source_id = sqlc.narg('linked_to')::uuid)
    ))
    AND (sqlc.narg('q')::text IS NULL
         OR to_tsvector('english', n.title || ' ' || n.body) @@ plainto_tsquery('english', sqlc.narg('q')::text))
    AND (sqlc.narg('tags')::text[] IS NULL OR (
        SELECT COUNT(DISTINCT t.name)
        FROM note_tags nt JOIN tags t ON t.id = nt.tag_id
        WHERE nt.note_id = n.id AND t.name = ANY(sqlc.narg('tags')::text[])
    ) = cardinality(sqlc.narg('tags')::text[]))
    AND (sqlc.narg('cursor_updated_at')::timestamptz IS NULL
         OR (n.updated_at, n.id) < (sqlc.narg('cursor_updated_at')::timestamptz, sqlc.narg('cursor_id')::uuid))
ORDER BY n.updated_at DESC, n.id DESC
LIMIT @page_limit;

-- name: CreateNote :one
INSERT INTO notes (title, body, category, target_id, created_by, expires_at)
VALUES (@title, @body, @category, sqlc.narg('target_id'), sqlc.narg('created_by'), sqlc.narg('expires_at'))
RETURNING id, title, body, category, target_id, created_by,
          expires_at, canonized_at, canonized_by, created_at, updated_at;

-- name: UpdateNote :one
UPDATE notes
SET title = COALESCE(sqlc.narg('title'), title),
    body  = COALESCE(sqlc.narg('body'), body),
    updated_at = now()
WHERE id = @id
RETURNING id, title, body, category, target_id, created_by,
          expires_at, canonized_at, canonized_by, created_at, updated_at;

-- name: DeleteNote :execrows
DELETE FROM notes WHERE id = @id;

-- name: CanonizeDraft :one
-- Promote an ai_draft to canon. Returns no row if the note isn't a draft.
UPDATE notes
SET category = 'canon',
    expires_at = NULL,
    canonized_at = now(),
    canonized_by = @actor,
    updated_at = now()
WHERE id = @id AND category = 'ai_draft'
RETURNING id, title, body, category, target_id, created_by,
          expires_at, canonized_at, canonized_by, created_at, updated_at;

-- name: ApplySuggestedEdit :one
-- Apply a suggested edit onto its canon target and stamp the canonization.
-- Tags are copied separately; the suggestion note is removed by the caller.
UPDATE notes AS target
SET title = s.title,
    body = s.body,
    canonized_at = now(),
    canonized_by = @actor,
    updated_at = now()
FROM notes AS s
WHERE s.id = @suggestion_id
  AND s.category = 'ai_suggested_edit'
  AND target.id = s.target_id
RETURNING target.id, target.title, target.body, target.category, target.target_id,
          target.created_by, target.expires_at, target.canonized_at, target.canonized_by,
          target.created_at, target.updated_at;

-- name: LogCanonization :exec
INSERT INTO canonization_log (note_id, actor) VALUES (@note_id, @actor);

-- name: GetNoteTags :many
SELECT t.name
FROM note_tags nt
JOIN tags t ON t.id = nt.tag_id
WHERE nt.note_id = @note_id
ORDER BY t.name;

-- name: ClearNoteTags :exec
DELETE FROM note_tags WHERE note_id = @note_id;

-- name: AttachTags :exec
-- Upsert tags by name and link them to the note. Call after ClearNoteTags within
-- the same transaction to replace a note's tag set.
WITH upserted AS (
    INSERT INTO tags (name)
    SELECT DISTINCT unnest(@names::text[])
    ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
    RETURNING id
)
INSERT INTO note_tags (note_id, tag_id)
SELECT @note_id, id FROM upserted
ON CONFLICT (note_id, tag_id) DO NOTHING;
