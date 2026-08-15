package privateapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	BaseURL          = "https://api.granola.ai"
	maxResponseBytes = 64 << 20
)

type Client struct {
	HTTP          *http.Client
	BaseURL       string
	AccessToken   string
	ClientVersion string
	Platform      string
	WorkspaceID   string
}

func (c Client) Do(ctx context.Context, endpoint string, input any, output any) error {
	base := c.BaseURL
	if base == "" {
		base = BaseURL
	}
	var body io.Reader
	if input != nil {
		b, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+endpoint, body)
	if err != nil {
		return err
	}
	if c.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Client-Version", defaultString(c.ClientVersion, "auto"))
	req.Header.Set("X-Granola-Platform", defaultString(c.Platform, "darwin"))
	if c.WorkspaceID != "" {
		req.Header.Set("X-Granola-Workspace-Id", c.WorkspaceID)
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return err
	}
	if len(b) > maxResponseBytes {
		return errors.New("granola private API response exceeds size limit")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return APIError{StatusCode: resp.StatusCode, Body: string(b)}
	}
	if output == nil {
		return nil
	}
	if err := json.Unmarshal(b, output); err != nil {
		return fmt.Errorf("decode %s: %w", endpoint, err)
	}
	return nil
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e APIError) Error() string {
	return fmt.Sprintf("granola api returned %d", e.StatusCode)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
