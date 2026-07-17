package store

import (
	"context"
	"database/sql"
	"fmt"

	ckstore "github.com/openclaw/crawlkit/store"
)

type Store struct {
	inner *ckstore.Store
}

func Open(ctx context.Context, path string) (*Store, error) {
	inner, err := ckstore.Open(ctx, ckstore.Options{
		Path:         path,
		Schema:       Schema,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return nil, err
	}
	current, err := inner.SchemaVersion(ctx)
	if err != nil {
		_ = inner.Close()
		return nil, err
	}
	if current > SchemaVersion {
		_ = inner.Close()
		return nil, fmt.Errorf("database schema version %d is newer than supported version %d", current, SchemaVersion)
	}
	if err := ensureDeletionColumns(ctx, inner.DB()); err != nil {
		_ = inner.Close()
		return nil, err
	}
	if err := inner.EnsureSchemaVersion(ctx, SchemaVersion); err != nil {
		_ = inner.Close()
		return nil, err
	}
	return &Store{inner: inner}, nil
}

func ensureDeletionColumns(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin deletion migration: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	columns := map[string][]string{
		"notes": {
			"deletion_source TEXT",
			"deletion_reason TEXT",
		},
		"transcript_chunks": {
			"deleted_at TEXT",
			"deletion_source TEXT",
			"deletion_reason TEXT",
		},
		"document_panels": {
			"deleted_at TEXT",
			"deletion_source TEXT",
			"deletion_reason TEXT",
		},
		"source_objects": {
			"deleted_at TEXT",
			"deletion_source TEXT",
			"deletion_reason TEXT",
		},
	}
	for table, definitions := range columns {
		for _, definition := range definitions {
			column := definition[:len(definition)-len(" TEXT")]
			var exists int
			query := fmt.Sprintf("SELECT count(*) FROM pragma_table_info(%q) WHERE name = ?", table)
			if err := tx.QueryRowContext(ctx, query, column).Scan(&exists); err != nil {
				return fmt.Errorf("inspect %s.%s: %w", table, column, err)
			}
			if exists != 0 {
				continue
			}
			stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", ckstore.QuoteIdent(table), definition)
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("add %s.%s: %w", table, column, err)
			}
		}
	}
	if err := ReconcileTombstones(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit deletion migration: %w", err)
	}
	committed = true
	return nil
}

func (s *Store) Close() error {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.Close()
}

func (s *Store) DB() *sql.DB {
	return s.inner.DB()
}

func (s *Store) Path() string {
	return s.inner.Path()
}
