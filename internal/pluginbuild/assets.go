package pluginbuild

import (
	"path/filepath"
)

// copyAssets is the shared asset-copy helper used by every target adapter.
// Markdown surfaces (commands, agents, skills) and static files are copied
// verbatim — the formats are compatible across Claude Code and Codex CLI.
func copyAssets(m *Manifest, repoRoot, outDir string) error {
	pairs := []struct{ src, dst string }{
		{m.AssetSources.Commands, "commands"},
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
			return err
		}
	}
	return nil
}
