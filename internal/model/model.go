package model

import "time"

type Source string

const (
	SourcePrivateAPI    Source = "private-api"
	SourcePublicAPI     Source = "public-api"
	SourceDesktopCache  Source = "desktop-cache"
	SourceEncryptedJSON Source = "encrypted-json"
	SourceOPFS          Source = "opfs"
	SourceCompanionCLI  Source = "companion-cli"
)

type Note struct {
	ID              string     `json:"id"`
	Title           *string    `json:"title"`
	Type            string     `json:"type"`
	Status          *string    `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	DeletionSource  string     `json:"deletion_source,omitempty"`
	DeletionReason  string     `json:"deletion_reason,omitempty"`
	WorkspaceID     *string    `json:"workspace_id,omitempty"`
	CalendarEventID *string    `json:"calendar_event_id,omitempty"`
	NotesPlain      *string    `json:"notes_plain,omitempty"`
	NotesMarkdown   *string    `json:"notes_markdown,omitempty"`
	SummaryText     *string    `json:"summary_text,omitempty"`
	SummaryMarkdown *string    `json:"summary_markdown,omitempty"`
	Source          Source     `json:"source"`
	PayloadHash     string     `json:"payload_hash,omitempty"`
	LastSeenAt      time.Time  `json:"last_seen_at"`
}

type TranscriptChunk struct {
	ID                string     `json:"id"`
	DocumentID        string     `json:"document_id"`
	StartTimestamp    time.Time  `json:"start_timestamp"`
	EndTimestamp      time.Time  `json:"end_timestamp"`
	Source            string     `json:"source"`
	IsFinal           bool       `json:"is_final"`
	TranscriberUserID *string    `json:"transcriber_user_id,omitempty"`
	Text              string     `json:"text"`
	PayloadHash       string     `json:"payload_hash,omitempty"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
	DeletionSource    string     `json:"deletion_source,omitempty"`
	DeletionReason    string     `json:"deletion_reason,omitempty"`
}

type Panel struct {
	ID              string     `json:"id"`
	DocumentID      string     `json:"document_id"`
	Title           *string    `json:"title,omitempty"`
	TemplateSlug    *string    `json:"template_slug,omitempty"`
	ContentPlain    *string    `json:"content_plain,omitempty"`
	ContentMarkdown *string    `json:"content_markdown,omitempty"`
	ContentJSON     string     `json:"content_json,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
	LastViewedAt    *time.Time `json:"last_viewed_at,omitempty"`
	YdocVersion     *int64     `json:"ydoc_version,omitempty"`
	YdocCachedAt    *time.Time `json:"ydoc_cached_at,omitempty"`
	Source          Source     `json:"source"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	DeletionSource  string     `json:"deletion_source,omitempty"`
	DeletionReason  string     `json:"deletion_reason,omitempty"`
}

type SyncRun struct {
	ID          int64     `json:"id"`
	Source      Source    `json:"source"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Status      string    `json:"status"`
	Notes       int       `json:"notes"`
	Transcripts int       `json:"transcripts"`
	Panels      int       `json:"panels"`
	Message     string    `json:"message,omitempty"`
}
