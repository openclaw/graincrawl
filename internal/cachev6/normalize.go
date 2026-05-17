package cachev6

import (
	"time"

	"github.com/openclaw/graincrawl/internal/hashutil"
	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/timeutil"
)

func NoteFromDocument(doc Document, now time.Time) (model.Note, error) {
	created, err := timeutil.Parse(doc.CreatedAt)
	if err != nil {
		return model.Note{}, err
	}
	updated, err := timeutil.Parse(doc.UpdatedAt)
	if err != nil {
		return model.Note{}, err
	}
	deleted, err := timeutil.ParsePtr(doc.DeletedAt)
	if err != nil {
		return model.Note{}, err
	}
	noteType := doc.Type
	if noteType == "" {
		noteType = "meeting"
	}
	return model.Note{
		ID:            doc.ID,
		Title:         doc.Title,
		Type:          noteType,
		Status:        doc.Status,
		CreatedAt:     created,
		UpdatedAt:     updated,
		DeletedAt:     deleted,
		WorkspaceID:   doc.WorkspaceID,
		NotesPlain:    doc.NotesPlain,
		NotesMarkdown: doc.NotesMarkdown,
		Source:        model.SourceDesktopCache,
		PayloadHash:   hashutil.JSON(doc),
		LastSeenAt:    now,
	}, nil
}

func TranscriptFromCache(chunk TranscriptChunk) (model.TranscriptChunk, error) {
	start, err := timeutil.Parse(chunk.StartTimestamp)
	if err != nil {
		return model.TranscriptChunk{}, err
	}
	end, err := timeutil.Parse(chunk.EndTimestamp)
	if err != nil {
		return model.TranscriptChunk{}, err
	}
	return model.TranscriptChunk{
		ID:                chunk.ID,
		DocumentID:        chunk.DocumentID,
		StartTimestamp:    start,
		EndTimestamp:      end,
		Source:            chunk.Source,
		IsFinal:           chunk.IsFinal,
		TranscriberUserID: chunk.TranscriberUserID,
		Text:              chunk.Text,
		PayloadHash:       hashutil.JSON(chunk),
	}, nil
}
