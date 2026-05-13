package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func buildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Rebuild the wipnote binary",
		Long:  "Compile the wipnote Go binary with version stamping and install it to ~/.local/bin/.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild()
		},
	}
}

func runBuild() error {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return err
	}

	version := resolveBuildVersion(projectRoot)

	if err := syncNotebookFiles(projectRoot); err != nil {
		return fmt.Errorf("sync notebook files: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home dir: %w", err)
	}
	installDir := filepath.Join(home, ".local", "bin")
	metaDir := filepath.Join(home, ".local", "share", "wipnote")
	binaryPath := filepath.Join(installDir, "wipnote")
	aliasPath := filepath.Join(installDir, "wn")
	versionFile := filepath.Join(metaDir, ".binary-version")

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", installDir, err)
	}
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", metaDir, err)
	}

	// Remove existing binary first so macOS doesn't reuse a cached code signature.
	_ = os.Remove(binaryPath)

	fmt.Printf("Building wipnote (version: %s)...\n", version)
	goBuild := exec.Command("go", "build",
		"-ldflags", fmt.Sprintf("-s -w -X main.version=%s", version),
		"-o", binaryPath,
		"./cmd/wipnote/",
	)
	goBuild.Dir = projectRoot
	goBuild.Stdout = os.Stdout
	goBuild.Stderr = os.Stderr
	if err := goBuild.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}
	if err := os.Chmod(binaryPath, 0o755); err != nil {
		return fmt.Errorf("chmod %s: %w", binaryPath, err)
	}

	if err := os.Remove(aliasPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", aliasPath, err)
	}
	if err := os.Symlink(binaryPath, aliasPath); err != nil {
		return fmt.Errorf("symlink %s: %w", aliasPath, err)
	}

	if err := os.WriteFile(versionFile, []byte(version+"\n"), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", versionFile, err)
	}

	fmt.Printf("Installed: %s (v%s)\n", binaryPath, version)
	fmt.Printf("Alias:     %s -> wipnote\n", aliasPath)
	return nil
}

// findProjectRoot walks up from CWD looking for go.mod. Honors WIPNOTE_PROJECT_ROOT.
func findProjectRoot() (string, error) {
	if env := os.Getenv("WIPNOTE_PROJECT_ROOT"); env != "" {
		if _, err := os.Stat(filepath.Join(env, "go.mod")); err == nil {
			return env, nil
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf(
				"wipnote project root not found — no go.mod walking up from %s.\n"+
					"Run from within the project tree, or set WIPNOTE_PROJECT_ROOT=<repo-path>",
				cwd,
			)
		}
		dir = parent
	}
}

func resolveBuildVersion(projectRoot string) string {
	cmd := exec.Command("git", "describe", "--tags", "--always")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return "dev"
	}
	return strings.TrimPrefix(strings.TrimSpace(string(out)), "v")
}

// syncNotebookFiles mirrors prototypes/*.py into internal/notebook/files/ so
// //go:embed picks up the latest source. Idempotent; preserves destination-only
// files (plan_notebook.py, plan_persistence.py, plan_ui.py live only in the
// destination and must not be deleted).
func syncNotebookFiles(projectRoot string) error {
	srcDir := filepath.Join(projectRoot, "prototypes")
	dstDir := filepath.Join(projectRoot, "internal", "notebook", "files")

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".py") {
			continue
		}
		src := filepath.Join(srcDir, e.Name())
		dst := filepath.Join(dstDir, e.Name())
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if existing, err := os.ReadFile(dst); err == nil && string(existing) == string(data) {
			continue
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
