CREATE TABLE IF NOT EXISTS notes (
 id BIGSERIAL PRIMARY KEY,
 title TEXT NOT NULL CHECK (length(btrim(title)) > 0),
 content TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notes_title ON notes(title);
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = clock_timestamp(); RETURN NEW; END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS trg_notes_updated ON notes;
CREATE TRIGGER trg_notes_updated BEFORE UPDATE ON notes
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
