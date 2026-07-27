package granola

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPostMigrationState(t *testing.T) {
	tests := []struct {
		name          string
		createProfile bool
		files         []string
		want          bool
	}{
		{
			name:          "migrated",
			createProfile: true,
			files:         []string{"cache-v6.json.enc"},
			want:          true,
		},
		{
			name:          "legacy",
			createProfile: true,
			files:         []string{"cache-v6.json.enc", "storage.dek"},
			want:          false,
		},
		{
			name:          "plaintext only",
			createProfile: true,
			files:         []string{"cache-v6.json"},
			want:          false,
		},
		{
			name: "no profile",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := filepath.Join(t.TempDir(), "Granola")
			if tt.createProfile {
				if err := os.MkdirAll(profile, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			for _, name := range tt.files {
				if err := os.WriteFile(filepath.Join(profile, name), []byte("fixture"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if got := PostMigrationState(Paths(profile, "")); got != tt.want {
				t.Fatalf("PostMigrationState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPostMigrationDiagnosticIncludesVersionWhenAvailable(t *testing.T) {
	got := PostMigrationDiagnostic("7.441.6")
	want := PostMigrationStateMessage + " Detected Granola version 7.441.6."
	if got != want {
		t.Fatalf("PostMigrationDiagnostic() = %q, want %q", got, want)
	}
}
