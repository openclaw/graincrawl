package privateapi

import (
	"encoding/json"
	"time"

	"github.com/vincentkoc/graincrawl/internal/hashutil"
	"github.com/vincentkoc/graincrawl/internal/model"
	"github.com/vincentkoc/graincrawl/internal/timeutil"
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
		Source:        model.SourcePrivateAPI,
		PayloadHash:   hashutil.JSON(doc),
		LastSeenAt:    now,
	}, nil
}

func TranscriptToModel(chunk TranscriptChunk) (model.TranscriptChunk, error) {
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

func PanelToModel(panel Panel) (model.Panel, error) {
	created, err := timeutil.Parse(panel.CreatedAt)
	if err != nil {
		return model.Panel{}, err
	}
	updated, err := timeutil.ParsePtr(panel.UpdatedAt)
	if err != nil {
		return model.Panel{}, err
	}
	viewed, err := timeutil.ParsePtr(panel.LastViewedAt)
	if err != nil {
		return model.Panel{}, err
	}
	content := ""
	if len(panel.Content) > 0 {
		content = string(panel.Content)
	}
	plain := panelText(panel.Content)
	return model.Panel{
		ID:           panel.ID,
		DocumentID:   panel.DocumentID,
		Title:        panel.Title,
		TemplateSlug: panel.TemplateSlug,
		ContentPlain: plain,
		ContentJSON:  content,
		CreatedAt:    created,
		UpdatedAt:    updated,
		LastViewedAt: viewed,
		YdocVersion:  panel.YdocVersion,
		Source:       model.SourcePrivateAPI,
	}, nil
}

func panelText(raw json.RawMessage) *string {
	if len(raw) == 0 {
		return nil
	}
	var walk func(any, *[]string)
	var parts []string
	walk = func(v any, out *[]string) {
		switch x := v.(type) {
		case map[string]any:
			if text, ok := x["text"].(string); ok && text != "" {
				*out = append(*out, text)
			}
			if children, ok := x["content"].([]any); ok {
				for _, child := range children {
					walk(child, out)
				}
			}
		case []any:
			for _, child := range x {
				walk(child, out)
			}
		}
	}
	var decoded any
	if json.Unmarshal(raw, &decoded) == nil {
		walk(decoded, &parts)
	}
	if len(parts) == 0 {
		return nil
	}
	text := ""
	for i, part := range parts {
		if i > 0 {
			text += "\n"
		}
		text += part
	}
	return &text
}
