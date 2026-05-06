package store

const SchemaVersion = 1

const Schema = `
CREATE TABLE IF NOT EXISTS source_objects (
  source TEXT NOT NULL,
  kind TEXT NOT NULL,
  source_id TEXT NOT NULL,
  document_id TEXT,
  payload_json TEXT NOT NULL,
  payload_hash TEXT NOT NULL,
  observed_at TEXT NOT NULL,
  PRIMARY KEY (source, kind, source_id)
);

CREATE TABLE IF NOT EXISTS notes (
  id TEXT PRIMARY KEY,
  title TEXT,
  type TEXT NOT NULL,
  status TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT,
  workspace_id TEXT,
  calendar_event_id TEXT,
  notes_plain TEXT,
  notes_markdown TEXT,
  summary_text TEXT,
  summary_markdown TEXT,
  source TEXT NOT NULL,
  payload_hash TEXT,
  last_seen_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS transcript_chunks (
  id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL,
  start_timestamp TEXT NOT NULL,
  end_timestamp TEXT NOT NULL,
  source TEXT NOT NULL,
  is_final INTEGER NOT NULL,
  transcriber_user_id TEXT,
  text TEXT NOT NULL,
  payload_hash TEXT
);

CREATE TABLE IF NOT EXISTS document_panels (
  id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL,
  title TEXT,
  template_slug TEXT,
  content_plain TEXT,
  content_markdown TEXT,
  content_json TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT,
  last_viewed_at TEXT,
  ydoc_version INTEGER,
  ydoc_cached_at TEXT,
  source TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sync_runs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source TEXT NOT NULL,
  started_at TEXT NOT NULL,
  completed_at TEXT NOT NULL,
  status TEXT NOT NULL,
  notes INTEGER NOT NULL DEFAULT 0,
  transcripts INTEGER NOT NULL DEFAULT 0,
  panels INTEGER NOT NULL DEFAULT 0,
  message TEXT
);

CREATE TABLE IF NOT EXISTS source_state (
  source TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  value TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (source, entity_type, entity_id)
);

CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
  note_id UNINDEXED,
  title,
  notes_plain,
  notes_markdown,
  summary_text,
  summary_markdown,
  transcript_text,
  panel_text
);

CREATE INDEX IF NOT EXISTS idx_notes_created_at ON notes(created_at);
CREATE INDEX IF NOT EXISTS idx_notes_updated_at ON notes(updated_at);
CREATE INDEX IF NOT EXISTS idx_notes_workspace_id ON notes(workspace_id);
CREATE INDEX IF NOT EXISTS idx_transcript_document_time ON transcript_chunks(document_id, start_timestamp);
CREATE INDEX IF NOT EXISTS idx_panels_document_id ON document_panels(document_id);
CREATE INDEX IF NOT EXISTS idx_source_objects_document_id ON source_objects(document_id);
`
