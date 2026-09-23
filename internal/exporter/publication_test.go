package exporter

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/store"
)

func TestMarkdownReplacesOutputWithoutFollowingLinks(t *testing.T) {
	for _, kind := range []string{"regular", "symlink", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			st, err := store.Open(ctx, filepath.Join(t.TempDir(), "archive.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			body := "private meeting notes"
			note := model.Note{ID: "doc-1", Type: "meeting", NotesPlain: &body}
			if err := st.UpsertNote(ctx, note); err != nil {
				t.Fatal(err)
			}
			out := t.TempDir()
			path := filepath.Join(out, noteFilename(note))
			external := filepath.Join(t.TempDir(), "unrelated.txt")
			if err := os.WriteFile(external, []byte("keep me"), 0o644); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "regular":
				err = os.WriteFile(path, []byte("previous export"), 0o644)
			case "symlink":
				err = os.Symlink(external, path)
			case "hardlink":
				err = os.Link(external, path)
			}
			if err != nil {
				t.Fatal(err)
			}
			result, err := Markdown(ctx, st, out, 10)
			if err != nil || result.Count != 1 {
				t.Fatalf("export = %+v, error = %v", result, err)
			}
			got, err := os.ReadFile(external)
			if err != nil || string(got) != "keep me" {
				t.Errorf("export overwrote unrelated file: %q, %v", got, err)
			}
			info, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
				t.Errorf("export mode = %v, want regular file with 0600 permissions", info.Mode())
			}
			got, err = os.ReadFile(path)
			if err != nil || !strings.Contains(string(got), body) {
				t.Fatalf("export missing note: %q, %v", got, err)
			}
			entries, err := os.ReadDir(out)
			if err != nil || len(entries) != 1 {
				t.Fatalf("export left temporary files: %v, %v", entries, err)
			}
		})
	}
}

func TestMarkdownPublicationFailureCleansTemporaryFile(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	note := model.Note{ID: "doc-1", Type: "meeting"}
	if err := st.UpsertNote(ctx, note); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	path := filepath.Join(out, noteFilename(note))
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(path, "unrelated.txt")
	if err := os.WriteFile(sentinel, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Markdown(ctx, st, out, 10); err == nil {
		t.Fatal("export must fail when the destination is a directory")
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "keep me" {
		t.Fatalf("failed export changed destination: %q, %v", got, err)
	}
	entries, err := os.ReadDir(out)
	if err != nil || len(entries) != 1 {
		t.Fatalf("failed export left temporary files: %v, %v", entries, err)
	}
}
