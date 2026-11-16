package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	queryRowLimit = 1000
	queryTimeout  = 10 * time.Second
)

func (s *Store) ReadOnlyQuery(ctx context.Context, query string) ([]string, [][]string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil, errors.New("empty query")
	}
	if !IsReadOnlySQL(query) {
		return nil, nil, errors.New("only read-only sql is allowed")
	}
	db, closeFn, err := s.openReadOnlyDB()
	if err != nil {
		return nil, nil, err
	}
	defer closeFn()
	return queryRows(ctx, db, query)
}

func IsReadOnlySQL(query string) bool {
	query = strings.TrimLeft(strings.ToLower(strings.TrimSpace(query)), " \t\r\n(")
	for _, keyword := range []string{"select", "with", "pragma"} {
		if strings.HasPrefix(query, keyword) && len(query) > len(keyword) &&
			strings.ContainsRune(" \t\r\n\f", rune(query[len(keyword)])) {
			return true
		}
	}
	return false
}

func (s *Store) openReadOnlyDB() (*sql.DB, func(), error) {
	path := strings.TrimSpace(s.Path())
	if path == "" {
		return nil, nil, errors.New("store path is empty")
	}
	dsn := (&url.URL{
		Scheme: "file",
		Path:   path,
		RawQuery: url.Values{
			"mode":    []string{"ro"},
			"_pragma": []string{"query_only(1)", "busy_timeout(5000)", "temp_store(MEMORY)"},
		}.Encode(),
	}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, func() { _ = db.Close() }, nil
}

func queryRows(ctx context.Context, db *sql.DB, query string) ([]string, [][]string, error) {
	queryCtx, cancel := withQueryTimeout(ctx)
	defer cancel()

	rows, err := db.QueryContext(queryCtx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	if len(cols) == 0 {
		return nil, nil, errors.New("query returned no columns")
	}
	out := make([][]string, 0)
	for rows.Next() {
		if len(out) >= queryRowLimit {
			return nil, nil, fmt.Errorf("query returned more than %d rows", queryRowLimit)
		}
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, nil, err
		}
		row := make([]string, len(cols))
		for i, value := range values {
			row[i] = stringifySQLValue(value)
		}
		out = append(out, row)
	}
	return cols, out, rows.Err()
}

func withQueryTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, queryTimeout)
}

func stringifySQLValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case []byte:
		return string(v)
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}
