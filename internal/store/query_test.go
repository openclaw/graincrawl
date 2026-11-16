package store

import (
	"path/filepath"
	"strconv"
	"testing"
)

func TestReadOnlyQueryAcceptsSQLWhitespace(t *testing.T) {
	st, err := Open(t.Context(), filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	for name, separator := range map[string]string{"tab": "\t", "CRLF": "\r\n", "form feed": "\f"} {
		for _, query := range []string{
			"SELECT" + separator + "1",
			"WITH" + separator + "value(n) AS (SELECT 1) SELECT n FROM value",
			"PRAGMA" + separator + "user_version",
		} {
			t.Run(name+"/"+query, func(t *testing.T) {
				var want int
				if err := st.DB().QueryRowContext(t.Context(), query).Scan(&want); err != nil {
					t.Fatalf("SQLite rejected fixture query: %v", err)
				}
				cols, rows, err := st.ReadOnlyQuery(t.Context(), query)
				if err != nil {
					t.Fatal(err)
				}
				if len(cols) != 1 || len(rows) != 1 || len(rows[0]) != 1 || rows[0][0] != strconv.Itoa(want) {
					t.Fatalf("query result = %v, %v; want one value %d", cols, rows, want)
				}
			})
		}
	}
}

func TestReadOnlyQueryRejectsWritesAndKeywordPrefixes(t *testing.T) {
	st, err := Open(t.Context(), filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	var before int
	if err := st.DB().QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&before); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"SELECTIVE 1", "WITHHOLD 1", "PRAGMATIC user_version", "DELETE FROM notes", "PRAGMA\tuser_version = 99"} {
		if _, _, err := st.ReadOnlyQuery(t.Context(), query); err == nil {
			t.Errorf("accepted invalid or mutating query %q", query)
		}
	}
	var after int
	if err := st.DB().QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("query changed user_version from %d to %d", before, after)
	}
}
