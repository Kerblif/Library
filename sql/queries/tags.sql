-- name: ListTags :many
SELECT t.id, t.name, t.created_at, COUNT(nt.note_id) AS note_count
FROM tags t
LEFT JOIN note_tags nt ON nt.tag_id = t.id
GROUP BY t.id
ORDER BY t.name;
