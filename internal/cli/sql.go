package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/openclaw/graincrawl/internal/output"
	gruntime "github.com/openclaw/graincrawl/internal/runtime"
)

func (a App) runSQL(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" || query == "-" {
		data, err := io.ReadAll(bufio.NewReader(os.Stdin))
		if err != nil {
			return err
		}
		query = strings.TrimSpace(string(data))
	}
	if query == "" {
		return fmt.Errorf("sql query required")
	}
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	cols, rows, err := rt.Store.ReadOnlyQuery(ctx, query)
	if err != nil {
		return err
	}
	if flags.JSON {
		return output.WriteEnvelope(w, map[string]any{"columns": cols, "rows": rows})
	}
	printSQLRows(w, cols, rows)
	return nil
}

func printSQLRows(w io.Writer, cols []string, rows [][]string) {
	fmt.Fprintln(w, strings.Join(cols, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
}
