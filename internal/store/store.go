package store

import (
	"context"
	"database/sql"

	ckstore "github.com/openclaw/crawlkit/store"
)

type Store struct {
	inner *ckstore.Store
}

func Open(ctx context.Context, path string) (*Store, error) {
	inner, err := ckstore.Open(ctx, ckstore.Options{
		Path:          path,
		Schema:        Schema,
		SchemaVersion: SchemaVersion,
		MaxOpenConns:  1,
		MaxIdleConns:  1,
	})
	if err != nil {
		return nil, err
	}
	return &Store{inner: inner}, nil
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
