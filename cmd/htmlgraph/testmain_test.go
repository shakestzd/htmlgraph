//go:build !integration

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain is the test suite entry point for the non-integration (unit) test
// suite. It redirects HTMLGRAPH_DB_PATH to a process-scoped temp directory so
// that no test inadvertently creates entries under the user's real
// ~/.cache/htmlgraph. Tests that need a per-test isolated DB should set their
// own HTMLGRAPH_DB_PATH via t.Setenv, which will override this process-wide
// default for the duration of that test and restore it afterwards.
//
// See bug-8c34e1f5 for context: without this redirect, each t.TempDir()-rooted
// test project produced a unique hash-keyed subdir under the real user cache,
// creating thousands of entries (6,022 subdirs / 2.5 GB) after a single test run.
func TestMain(m *testing.M) {
	// Redirect DB to a process-scoped temp dir before any test runs.
	// os.MkdirTemp is used (not t.TempDir) because TestMain has no *testing.T.
	// Note: defer does not execute before os.Exit, so we capture the dir and
	// clean up manually after m.Run() returns the exit code.
	var dbTmp string
	if tmp, err := os.MkdirTemp("", "htmlgraph-test-db-*"); err == nil {
		dbTmp = tmp
		os.Setenv("HTMLGRAPH_DB_PATH", filepath.Join(dbTmp, "htmlgraph.db"))
	}

	code := m.Run()

	// Cleanup: remove the process-scoped temp DB dir.
	if dbTmp != "" {
		_ = os.RemoveAll(dbTmp)
	}

	os.Exit(code)
}
