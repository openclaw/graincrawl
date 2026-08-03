package publicapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	BaseURL          = "https://public-api.granola.ai"
	KeyEnv           = "GRANOLA_PUBLIC_API_KEY"
	pageLimit        = 30
	maxResponseBytes = 64 << 20
	maxAttempts      = 4
)

type Client struct {
	HTTP            *http.Client
	BaseURL         string
	APIKey          string
	RequestInterval time.Duration

	mu          sync.Mutex
	lastRequest time.Time
}

func (c *Client) ListNotes(ctx context.Context, cursor string, pageSize int) (NotesResponse, error) {
	if pageSize <= 0 || pageSize > pageLimit {
		pageSize = pageLimit
	}
	query := url.Values{"page_size": {strconv.Itoa(pageSize)}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	var response NotesResponse
	err := c.get(ctx, "/v1/notes?"+query.Encode(), &response)
	return response, err
}

func (c *Client) GetNote(ctx context.Context, noteID string, includeTranscript bool) (Note, error) {
	endpoint := "/v1/notes/" + url.PathEscape(noteID)
	if includeTranscript {
		endpoint += "?include=transcript"
	}
	var note Note
	err := c.get(ctx, endpoint, &note)
	return note, err
}

func (c *Client) get(ctx context.Context, endpoint string, output any) error {
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := c.wait(ctx); err != nil {
			return err
		}
		req, err := c.request(ctx, endpoint)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
		closeErr := resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		if len(body) > maxResponseBytes {
			return errors.New("granola public API response exceeds size limit")
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt+1 < maxAttempts {
			if err := waitForRetry(ctx, retryDelay(resp.Header.Get("Retry-After"), attempt)); err != nil {
				return err
			}
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return APIError{StatusCode: resp.StatusCode}
		}
		if err := json.Unmarshal(body, output); err != nil {
			return fmt.Errorf("decode public API response: %w", err)
		}
		return nil
	}
	return APIError{StatusCode: http.StatusTooManyRequests}
}

func (c *Client) request(ctx context.Context, endpoint string) (*http.Request, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = BaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func (c *Client) wait(ctx context.Context) error {
	interval := c.RequestInterval
	if interval == 0 {
		interval = 200 * time.Millisecond
	}
	if interval < 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if remaining := interval - time.Since(c.lastRequest); !c.lastRequest.IsZero() && remaining > 0 {
		timer := time.NewTimer(remaining)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	c.lastRequest = time.Now()
	return nil
}

type APIError struct {
	StatusCode int
}

func (e APIError) Error() string {
	return fmt.Sprintf("granola public API returned %d", e.StatusCode)
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		if delay := time.Until(when); delay > 0 {
			return delay
		}
		return 0
	}
	return time.Second << attempt
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
