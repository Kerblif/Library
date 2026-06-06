-- name: CreateLink :one
INSERT INTO links (source_id, target_id)
VALUES (@source_id, @target_id)
RETURNING id, source_id, target_id, created_at;

-- name: DeleteLink :execrows
DELETE FROM links WHERE id = @id;

-- name: ListOutgoingLinks :many
SELECT id, source_id, target_id, created_at
FROM links
WHERE source_id = @note_id
ORDER BY created_at, id;

-- name: ListIncomingLinks :many
SELECT id, source_id, target_id, created_at
FROM links
WHERE target_id = @note_id
ORDER BY created_at, id;
