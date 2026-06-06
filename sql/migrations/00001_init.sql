-- +goose Up

CREATE TYPE category AS ENUM ('canon', 'ai_draft', 'ai_suggested_edit');

CREATE TABLE notes (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    title        text        NOT NULL CHECK (char_length(title) BETWEEN 1 AND 200),
    body         text        NOT NULL,
    category     category    NOT NULL,
    -- For ai_suggested_edit: the canon note this edit is proposed against.
    target_id    uuid        REFERENCES notes (id) ON DELETE CASCADE,
    created_by   text,
    -- Inactivity TTL; only ai_draft notes have one.
    expires_at   timestamptz,
    -- Set when a human removes the AI mark (canonization).
    canonized_at timestamptz,
    canonized_by text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),

    -- target_id is present exactly for suggested edits.
    CONSTRAINT notes_target_only_suggested
        CHECK ((category = 'ai_suggested_edit') = (target_id IS NOT NULL)),
    -- A TTL only makes sense for drafts.
    CONSTRAINT notes_ttl_only_draft
        CHECK (expires_at IS NULL OR category = 'ai_draft'),
    -- Canonization metadata is logged as a pair.
    CONSTRAINT notes_canonized_pair
        CHECK ((canonized_at IS NULL) = (canonized_by IS NULL))
);

CREATE INDEX notes_category_idx ON notes (category);
CREATE INDEX notes_page_idx ON notes (updated_at DESC, id DESC);
CREATE INDEX notes_target_idx ON notes (target_id) WHERE target_id IS NOT NULL;
CREATE INDEX notes_expiry_idx ON notes (expires_at) WHERE category = 'ai_draft';
CREATE INDEX notes_search_idx ON notes USING gin (to_tsvector('english', title || ' ' || body));

CREATE TABLE tags (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text        NOT NULL UNIQUE
                   CHECK (name ~ '^[a-z0-9][a-z0-9-]*$' AND char_length(name) <= 64),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE note_tags (
    note_id uuid NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    tag_id  uuid NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_id)
);

CREATE INDEX note_tags_tag_idx ON note_tags (tag_id);

CREATE TABLE links (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id  uuid        NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    target_id  uuid        NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT links_no_self CHECK (source_id <> target_id),
    CONSTRAINT links_unique UNIQUE (source_id, target_id)
);

CREATE INDEX links_target_idx ON links (target_id);

CREATE TABLE canonization_log (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id    uuid        NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    actor      text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX canonization_log_note_idx ON canonization_log (note_id);

-- +goose Down

DROP TABLE IF EXISTS canonization_log;
DROP TABLE IF EXISTS links;
DROP TABLE IF EXISTS note_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS notes;
DROP TYPE IF EXISTS category;
