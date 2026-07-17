package portable

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	cksnapshot "github.com/openclaw/crawlkit/snapshot"
	ckstore "github.com/openclaw/crawlkit/store"
	"github.com/openclaw/graincrawl/internal/store"
)

var Tables = []string{
	"source_objects",
	"notes",
	"transcript_chunks",
	"document_panels",
	"sync_runs",
	"source_state",
}

type Options struct {
	RootDir string
	Replace bool
}

type Manifest = cksnapshot.Manifest

func Export(ctx context.Context, st *store.Store, opts Options) (Manifest, error) {
	return cksnapshot.Export(ctx, cksnapshot.ExportOptions{
		DB:      st.DB(),
		RootDir: opts.RootDir,
		Tables:  Tables,
	})
}

func Import(ctx context.Context, st *store.Store, opts Options) (Manifest, error) {
	importOpts := cksnapshot.ImportOptions{
		DB:      st.DB(),
		RootDir: opts.RootDir,
		AfterImport: func(ctx context.Context, tx *sql.Tx) error {
			return store.ReconcileTombstones(ctx, tx)
		},
	}
	if opts.Replace {
		importOpts.DeleteTables = Tables
	} else {
		importOpts.DeleteTable = func(context.Context, *sql.Tx, string) error { return nil }
		importOpts.ImportRow = mergeRow
	}
	return cksnapshot.Import(ctx, importOpts)
}

var conflictColumns = map[string][]string{
	"source_objects":    {"source", "kind", "source_id"},
	"notes":             {"id"},
	"transcript_chunks": {"id"},
	"document_panels":   {"id"},
	"sync_runs":         {"id"},
	"source_state":      {"source", "entity_type", "entity_id"},
}

func mergeRow(ctx context.Context, tx *sql.Tx, table string, row map[string]any) error {
	keys, ok := conflictColumns[table]
	if !ok {
		return fmt.Errorf("snapshot contains unsupported table %q", table)
	}
	cols := make([]string, 0, len(row))
	for col := range row {
		cols = append(cols, col)
	}
	sort.Strings(cols)
	quoted := make([]string, 0, len(cols))
	holders := make([]string, 0, len(cols))
	args := make([]any, 0, len(cols))
	updates := make([]string, 0, 3)
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[key] = struct{}{}
	}
	for _, col := range cols {
		quotedCol := ckstore.QuoteIdent(col)
		quoted = append(quoted, quotedCol)
		holders = append(holders, "?")
		args = append(args, row[col])
		if _, isKey := keySet[col]; isKey {
			continue
		}
		switch col {
		case "deleted_at":
			updates = append(updates, fmt.Sprintf("%s=COALESCE(%s.%s, excluded.%s)", quotedCol, ckstore.QuoteIdent(table), quotedCol, quotedCol))
		case "deletion_source", "deletion_reason":
			updates = append(updates, fmt.Sprintf("%s=COALESCE(NULLIF(%s.%s, ''), excluded.%s)", quotedCol, ckstore.QuoteIdent(table), quotedCol, quotedCol))
		}
	}
	conflict := make([]string, 0, len(keys))
	for _, key := range keys {
		conflict = append(conflict, ckstore.QuoteIdent(key))
	}
	action := "DO NOTHING"
	if len(updates) > 0 {
		action = "DO UPDATE SET " + strings.Join(updates, ",")
	}
	stmt := fmt.Sprintf(
		"INSERT INTO %s(%s) VALUES(%s) ON CONFLICT(%s) %s",
		ckstore.QuoteIdent(table), strings.Join(quoted, ","), strings.Join(holders, ","),
		strings.Join(conflict, ","), action,
	)
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return fmt.Errorf("merge %s row: %w", table, err)
	}
	return nil
}

func DefaultDir(root string, now time.Time) string {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return filepath.Join(root, "snapshot-"+now.UTC().Format("20060102T150405Z"))
}
