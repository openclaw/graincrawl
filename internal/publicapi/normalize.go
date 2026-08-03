package publicapi

import (
	"fmt"
	"time"

	"github.com/openclaw/graincrawl/internal/hashutil"
	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/timeutil"
)

func NoteToModel(note Note, now time.Time) (model.Note, error) {
	created, err := timeutil.Parse(note.CreatedAt)
	if err != nil {
		return model.Note{}, err
	}
	updated, err := timeutil.Parse(note.UpdatedAt)
	if err != nil {
		return model.Note{}, err
	}
	var calendarEventID *string
	if note.CalendarEvent.CalendarEventID != "" {
		calendarEventID = &note.CalendarEvent.CalendarEventID
	}
	var summaryText *string
	if note.SummaryText != "" {
		summaryText = &note.SummaryText
	}
	return model.Note{
		ID:              note.ID,
		Title:           note.Title,
		Type:            "meeting",
		CreatedAt:       created,
		UpdatedAt:       updated,
		CalendarEventID: calendarEventID,
		SummaryText:     summaryText,
		SummaryMarkdown: note.SummaryMarkdown,
		Source:          model.SourcePublicAPI,
		PayloadHash:     hashutil.JSON(note),
		LastSeenAt:      now,
	}, nil
}

func TranscriptToModel(noteID string, transcript Transcript, occurrence int) (model.TranscriptChunk, error) {
	start, err := timeutil.Parse(transcript.StartTime)
	if err != nil {
		return model.TranscriptChunk{}, err
	}
	end, err := timeutil.Parse(transcript.EndTime)
	if err != nil {
		return model.TranscriptChunk{}, err
	}
	identityHash := TranscriptIdentityHash(transcript)
	return model.TranscriptChunk{
		ID:             fmt.Sprintf("public-api:%s:%s:%d", noteID, identityHash, occurrence),
		DocumentID:     noteID,
		StartTimestamp: start,
		EndTimestamp:   end,
		Source:         transcript.Speaker.Source,
		IsFinal:        true,
		Text:           transcript.Text,
		PayloadHash:    TranscriptHash(transcript),
	}, nil
}

func TranscriptHash(transcript Transcript) string {
	return hashutil.JSON(transcript)
}

func TranscriptIdentityHash(transcript Transcript) string {
	return hashutil.JSON(struct {
		StartTime        string `json:"start_time"`
		EndTime          string `json:"end_time"`
		Source           string `json:"source"`
		DiarizationLabel string `json:"diarization_label"`
	}{
		StartTime:        transcript.StartTime,
		EndTime:          transcript.EndTime,
		Source:           transcript.Speaker.Source,
		DiarizationLabel: transcript.Speaker.DiarizationLabel,
	})
}
