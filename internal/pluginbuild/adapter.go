package pluginbuild

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Adapter emits a target-specific plugin tree from the shared manifest. Each
// target (Claude Code, Codex CLI, …) registers one.
type Adapter interface {
	// Name identifies the target in CLI flags and manifest.targets keys.
	Name() string

	// Emit writes the generated plugin tree under outDir. repoRoot is the
	// absolute path to the repository root — adapters resolve manifest asset
	// sources relative to it.
	Emit(m *Manifest, repoRoot, outDir string) error
}

// Registry holds all known adapters by name.
var registry = map[string]Adapter{}

// Register adds an adapter to the registry. Duplicate registrations for the
// same name panic at init time so conflicts surface immediately.
func Register(a Adapter) {
	if _, dup := registry[a.Name()]; dup {
		panic(fmt.Sprintf("pluginbuild: adapter %q already registered", a.Name()))
	}
	registry[a.Name()] = a
}

// Get returns the adapter for name or an error if none is registered.
func Get(name string) (Adapter, error) {
	a, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown plugin-build target %q (registered: %v)", name, Names())
	}
	return a, nil
}

// Names lists all registered target names in stable order.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	// sort for determinism — stdlib sort avoids importing another dep
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j-1] > names[j]; j-- {
			names[j-1], names[j] = names[j], names[j-1]
		}
	}
	return names
}

// copyAssetTree copies srcDir into dstDir recursively. Missing sources are
// silently skipped — every target is expected to accept a superset of assets
// and missing ones simply mean that the parent chose not to ship that surface.
func copyAssetTree(srcDir, dstDir string) error {
	info, err := os.Stat(srcDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat %s: %w", srcDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("asset source %s is not a directory", srcDir)
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// writeJSON marshals v to path with stable two-space indent and a trailing newline.
func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return writeJSONTo(f, v)
}
