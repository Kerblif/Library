-- +goose Up

ALTER TABLE notes
    ADD COLUMN archived    boolean     NOT NULL DEFAULT false,
    ADD COLUMN archived_at timestamptz,
    ADD COLUMN archived_by text,
    -- archived flag, timestamp and actor move together.
    ADD CONSTRAINT notes_archived_stamp CHECK (archived = (archived_at IS NOT NULL)),
    ADD CONSTRAINT notes_archived_pair  CHECK ((archived_at IS NULL) = (archived_by IS NULL)),
    -- Only canon notes can be archived.
    ADD CONSTRAINT notes_archive_only_canon CHECK (NOT archived OR category = 'canon');

-- Active notes by category: serves the default visibility filter and badge counts.
CREATE INDEX notes_active_idx ON notes (category) WHERE NOT archived;

-- +goose Down

DROP INDEX IF EXISTS notes_active_idx;

ALTER TABLE notes
    DROP COLUMN IF EXISTS archived_by,
    DROP COLUMN IF EXISTS archived_at,
    DROP COLUMN IF EXISTS archived;
