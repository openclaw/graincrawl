package privateapi

import "context"

func (c Client) GetDocuments(ctx context.Context, req DocumentsRequest) (DocumentsResponse, error) {
	var out DocumentsResponse
	err := c.Do(ctx, "/v2/get-documents", req, &out)
	return out, err
}

func (c Client) GetDocumentsBatch(ctx context.Context, ids []string) (BatchResponse, error) {
	var out BatchResponse
	const batchSize = 50
	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		var batch BatchResponse
		if err := c.Do(ctx, "/v1/get-documents-batch", BatchRequest{DocumentIDs: ids[start:end]}, &batch); err != nil {
			return out, err
		}
		out.Docs = append(out.Docs, batch.Docs...)
	}
	return out, nil
}

func (c Client) GetDocumentTranscript(ctx context.Context, id string) ([]TranscriptChunk, error) {
	var out []TranscriptChunk
	err := c.Do(ctx, "/v1/get-document-transcript", TranscriptRequest{DocumentID: id}, &out)
	return out, err
}

func (c Client) GetDocumentPanels(ctx context.Context, id string) ([]Panel, error) {
	var out []Panel
	err := c.Do(ctx, "/v1/get-document-panels", PanelsRequest{DocumentID: id}, &out)
	return out, err
}
