package portable

import (
	"context"
	"path/filepath"
	"time"

	cksnapshot "github.com/openclaw/crawlkit/snapshot"
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
	return cksnapshot.Import(ctx, cksnapshot.ImportOptions{
		DB:           st.DB(),
		RootDir:      opts.RootDir,
		DeleteTables: Tables,
	})
}

func DefaultDir(root string, now time.Time) string {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return filepath.Join(root, "snapshot-"+now.UTC().Format("20060102T150405Z"))
}
