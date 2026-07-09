package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: codesign-hook TARGET BINARY")
		os.Exit(2)
	}

	target := os.Args[1]
	if !strings.HasPrefix(target, "darwin_") || os.Getenv("GRAINCRAWL_REQUIRE_CODESIGN") != "1" {
		return
	}
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "official graincrawl macOS signing must run on macOS")
		os.Exit(1)
	}

	cmd := exec.Command("./scripts/codesign-graincrawl.sh", os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "codesign hook failed: %v\n", err)
		os.Exit(1)
	}
}
