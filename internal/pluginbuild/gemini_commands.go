package pluginbuild

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// init registers this phase's sub-emitter. Order within geminiSubEmitters is
// deterministic by filename collation across gemini_*.go files, so assets/
// commands/hooks never race even though each lives in its own file.
func init() {
	geminiSubEmitters = append(geminiSubEmitters, emitGeminiCommands)
}

// emitGeminiCommands translates every plugin/commands/*.md file into a Gemini
// TOML command under <outDir>/commands/<namespace>/<name>.toml. Gemini loads
// .toml commands where the `prompt` key is the full markdown body; the
// namespace segment means the slash-command resolves to /<namespace>:<name>
// (e.g. /htmlgraph:feature-start). When the target declares no namespace the
// files land directly under commands/ — a degenerate case kept for symmetry.
func emitGeminiCommands(m *Manifest, repoRoot, outDir string, t Target) error {
	if m.AssetSources.Commands == "" {
		return nil
	}
	srcDir := filepath.Join(repoRoot, m.AssetSources.Commands)
	info, err := os.Stat(srcDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat commands source %s: %w", srcDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("commands source %s is not a directory", srcDir)
	}

	dstDir := filepath.Join(outDir, "commands")
	if t.CommandNamespace != "" {
		dstDir = filepath.Join(dstDir, t.CommandNamespace)
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("read commands source %s: %w", srcDir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(srcDir, e.Name()))
		if err != nil {
			return fmt.Errorf("read command %s: %w", e.Name(), err)
		}
		name := strings.TrimSuffix(e.Name(), ".md") + ".toml"
		dst := filepath.Join(dstDir, name)
		if err := os.WriteFile(dst, []byte(toGeminiCommandTOML(string(body))), 0o644); err != nil {
			return fmt.Errorf("write gemini command %s: %w", dst, err)
		}
	}
	return nil
}

// toGeminiCommandTOML wraps a markdown body as a TOML `prompt` value using a
// triple-quoted multi-line string. Any literal `"""` inside the body is broken
// by inserting a backslash (`""\"`) — TOML treats this as three quote chars in
// the output but does NOT terminate the string, so we preserve the body
// byte-for-byte while keeping TOML parseable.
func toGeminiCommandTOML(mdBody string) string {
	escaped := strings.ReplaceAll(mdBody, `"""`, `""\"`)
	return "prompt = \"\"\"\n" + escaped + "\n\"\"\"\n"
}
