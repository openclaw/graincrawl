package publicapi

type NotesResponse struct {
	Notes   []NoteSummary `json:"notes"`
	HasMore bool          `json:"hasMore"`
	Cursor  *string       `json:"cursor"`
}

type NoteSummary struct {
	ID        string  `json:"id"`
	Object    string  `json:"object"`
	Title     *string `json:"title"`
	Owner     Person  `json:"owner"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type Note struct {
	ID               string        `json:"id"`
	Object           string        `json:"object"`
	Title            *string       `json:"title"`
	Owner            Person        `json:"owner"`
	CreatedAt        string        `json:"created_at"`
	UpdatedAt        string        `json:"updated_at"`
	WebURL           string        `json:"web_url"`
	CalendarEvent    CalendarEvent `json:"calendar_event"`
	Attendees        []Person      `json:"attendees"`
	FolderMembership []Folder      `json:"folder_membership"`
	SummaryText      string        `json:"summary_text"`
	SummaryMarkdown  *string       `json:"summary_markdown"`
	Transcript       []Transcript  `json:"transcript"`
}

type Person struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CalendarEvent struct {
	EventTitle         string   `json:"event_title"`
	Invitees           []Person `json:"invitees"`
	Organiser          string   `json:"organiser"`
	CalendarEventID    string   `json:"calendar_event_id"`
	ScheduledStartTime string   `json:"scheduled_start_time"`
	ScheduledEndTime   string   `json:"scheduled_end_time"`
}

type Folder struct {
	ID             string  `json:"id"`
	Object         string  `json:"object"`
	Name           string  `json:"name"`
	ParentFolderID *string `json:"parent_folder_id"`
}

type Transcript struct {
	Speaker   Speaker `json:"speaker"`
	Text      string  `json:"text"`
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
}

type Speaker struct {
	Source           string `json:"source"`
	Attribution      string `json:"attribution,omitempty"`
	DiarizationLabel string `json:"diarization_label,omitempty"`
}
