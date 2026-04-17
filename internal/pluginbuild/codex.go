package pluginbuild

import (
	"fmt"
	"os"
	"path/filepath"
)

func init() { Register(codexAdapter{}) }

// codexAdapter emits the Codex CLI plugin tree. Layout:
//
//	<outDir>/.codex-plugin/plugin.json
//	<outDir>/hooks.json           (at plugin root, not hooks/)
//	<outDir>/.mcp.json            (optional, at plugin root)
//	<outDir>/{commands,agents,skills,templates,static,config}/
//
// Codex hook event names differ from Claude in a few places (TaskStarted,
// TaskComplete, TurnAborted) — the manifest's `targets` field controls which
// events are emitted here. Business logic stays in `htmlgraph hook <handler>`
// so the Codex plugin is a thin wrapper just like the Claude one.
type codexAdapter struct{}

func (codexAdapter) Name() string { return "codex" }

// codexOwnedSubtrees lists the subdirectory names under the codex outDir that
// build-ports fully regenerates. Hand-maintained files (README.md, etc.) live
// outside these subtrees and are never touched by stale-file cleanup.
var codexOwnedSubtrees = []string{"commands", "agents", "skills", "templates", "static", "config"}

func (c codexAdapter) Emit(m *Manifest, repoRoot, outDir string) error {
	target, ok := m.Targets[c.Name()]
	if !ok {
		return fmt.Errorf("manifest has no target %q", c.Name())
	}

	// Pre-clean owned subtrees so renamed/deleted source files don't leave
	// stale output files behind. Non-owned files (README, hooks.json, etc.)
	// at the outDir root are untouched.
	if err := cleanOwnedSubtrees(outDir, codexOwnedSubtrees); err != nil {
		return fmt.Errorf("codex pre-clean: %w", err)
	}

	if err := writeCodexManifest(m, filepath.Join(outDir, target.ManifestPath)); err != nil {
		return err
	}
	if err := writeCodexHooks(m, filepath.Join(outDir, target.HooksPath)); err != nil {
		return err
	}
	if target.MCPPath != "" {
		if err := ensureCodexMCP(filepath.Join(outDir, target.MCPPath)); err != nil {
			return err
		}
	}
	return copyAssets(m, repoRoot, outDir)
}

// codexPluginJSON mirrors the Codex plugin manifest schema. The top-level
// shape is similar to Claude's, plus an `interface` block Codex uses for
// install-surface metadata.
type codexPluginJSON struct {
	Name        string             `json:"name"`
	Version     string             `json:"version"`
	Description string             `json:"description"`
	Author      codexAuthorJSON    `json:"author"`
	Homepage    string             `json:"homepage,omitempty"`
	Repository  string             `json:"repository,omitempty"`
	License     string             `json:"license,omitempty"`
	Keywords    []string           `json:"keywords,omitempty"`
	Interface   codexInterfaceJSON `json:"interface"`
}

type codexAuthorJSON struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

type codexInterfaceJSON struct {
	DisplayName      string `json:"displayName"`
	ShortDescription string `json:"shortDescription"`
	LongDescription  string `json:"longDescription,omitempty"`
	DeveloperName    string `json:"developerName"`
	Category         string `json:"category,omitempty"`
}

func writeCodexManifest(m *Manifest, path string) error {
	return writeJSON(path, codexPluginJSON{
		Name:        m.Name,
		Version:     m.Version,
		Description: m.Description,
		Author: codexAuthorJSON{
			Name:  m.Author.Name,
			Email: m.Author.Email,
			URL:   m.Author.URL,
		},
		Homepage:   m.Homepage,
		Repository: m.Repository,
		License:    m.License,
		Keywords:   m.Keywords,
		Interface: codexInterfaceJSON{
			DisplayName:      "HtmlGraph",
			ShortDescription: m.Description,
			DeveloperName:    m.Author.Name,
			Category:         m.Category,
		},
	})
}

// Codex hooks.json schema matches Claude's structure so shared matchers work.
// Different events are supported, not a different schema.
func writeCodexHooks(m *Manifest, path string) error {
	hooks := map[string][]claudeMatcherGroup{}
	order := []string{}

	for _, e := range m.Hooks.Events {
		if !e.AppliesTo("codex") {
			continue
		}
		cmd := e.Command
		if cmd == "" {
			cmd = "htmlgraph hook " + e.Handler
		}
		group := claudeMatcherGroup{
			Matcher: e.Matcher,
			Hooks: []claudeHookEntry{{
				Type:    "command",
				Command: cmd,
				Timeout: e.Timeout,
			}},
		}
		if _, seen := hooks[e.Name]; !seen {
			order = append(order, e.Name)
		}
		hooks[e.Name] = append(hooks[e.Name], group)
	}
	return writeJSON(path, orderedHookMap{keys: order, values: hooks})
}

// ensureCodexMCP writes a stub .mcp.json if none exists. HtmlGraph doesn't
// currently expose an MCP server, but the file is part of the Codex plugin
// contract and future MCP integrations land here without schema churn.
func ensureCodexMCP(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return writeJSON(path, map[string]any{"mcpServers": map[string]any{}})
}
