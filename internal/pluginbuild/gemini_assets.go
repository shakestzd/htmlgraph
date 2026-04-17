package pluginbuild

import (
	"fmt"
	"path/filepath"
)

// Phase 1 registration: append the asset sub-emitter at init time so
// geminiAdapter.Emit walks it without needing edits to gemini.go. Phases 2 and
// 3 register their own sub-emitters from separate files for the same reason.
func init() {
	geminiSubEmitters = append(geminiSubEmitters, emitGeminiAssets)
}

// emitGeminiAssets copies the verbatim-reusable asset trees (agents, skills,
// templates, static, config) plus the repo-root context file (GEMINI.md) into
// the generated Gemini extension tree.
//
// Commands are deliberately NOT copied here — Gemini uses TOML command
// definitions and a different on-disk layout (commands/<namespace>/*.toml).
// Phase 2 owns the .md → .toml translation and lives in its own file.
//
// Missing sources are tolerated (copyAssetTree treats them as no-ops), which
// mirrors how copyAssets behaves for the Claude and Codex targets.
func emitGeminiAssets(m *Manifest, repoRoot, outDir string, t Target) error {
	pairs := []struct{ src, dst string }{
		{m.AssetSources.Agents, "agents"},
		{m.AssetSources.Skills, "skills"},
		{m.AssetSources.Templates, "templates"},
		{m.AssetSources.Static, "static"},
		{m.AssetSources.Config, "config"},
	}
	for _, p := range pairs {
		if p.src == "" {
			continue
		}
		src := filepath.Join(repoRoot, p.src)
		dst := filepath.Join(outDir, p.dst)
		if err := copyAssetTree(src, dst); err != nil {
			return fmt.Errorf("gemini copy %s -> %s: %w", p.src, p.dst, err)
		}
	}
	// Gemini picks up the extension's "context file" (e.g. GEMINI.md) from the
	// extension root. The manifest target declares the repo-relative source via
	// ContextFile; we copy it verbatim to <outDir>/<basename>. When ContextFile
	// is empty, skip — targets that don't declare one (Claude, Codex) opt out.
	if t.ContextFile != "" {
		src := filepath.Join(repoRoot, t.ContextFile)
		dst := filepath.Join(outDir, filepath.Base(t.ContextFile))
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("gemini copy contextFile %s: %w", t.ContextFile, err)
		}
	}
	return nil
}
