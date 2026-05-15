package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// geminiHookEntry is a single hook command entry within a hook group.
type geminiHookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// geminiHookGroup is one element of a per-event hook list in Gemini settings.
// Format mirrors packages/gemini-extension/hooks/hooks.json.
type geminiHookGroup struct {
	Matcher string            `json:"matcher"`
	Hooks   []geminiHookEntry `json:"hooks"`
}

// geminiHooksFile is the top-level structure of hooks.json (and the hooks
// section inside settings.json).
type geminiHooksFile struct {
	Hooks map[string][]geminiHookGroup `json:"hooks"`
}

// geminiSettingsJSONPath returns the target settings.json path.
//
//   - "" (empty extensionName) → normal mode: ~/.gemini/settings.json
//   - non-empty extensionName  → dev/isolate mode: ~/.gemini/extensions/<name>/settings.json
//
// The isolated path is used when the launcher redirects Gemini to a per-extension
// settings dir (e.g. wipnote gemini --dev) so we never touch the real ~/.gemini/
// in dev mode.
func geminiSettingsJSONPath(extensionName string) string {
	home, _ := os.UserHomeDir()
	if extensionName == "" {
		return filepath.Join(home, ".gemini", "settings.json")
	}
	return filepath.Join(home, ".gemini", "extensions", extensionName, "settings.json")
}

// readGeminiHooksFile reads and parses a Gemini hooks.json file from the
// bundled extension tree.
func readGeminiHooksFile(hooksPath string) (geminiHooksFile, error) {
	data, err := os.ReadFile(hooksPath)
	if err != nil {
		return geminiHooksFile{}, fmt.Errorf("reading hooks file %s: %w", hooksPath, err)
	}
	var hf geminiHooksFile
	if err := json.Unmarshal(data, &hf); err != nil {
		return geminiHooksFile{}, fmt.Errorf("parsing hooks file %s: %w", hooksPath, err)
	}
	return hf, nil
}

// mergeGeminiHooks merges wipnote hooks (bundledRaw) into an existing
// settings.json body (existingRaw).
//
// Merge rules:
//   - All keys from existingRaw other than "hooks" are preserved verbatim.
//   - For each event in bundledRaw's "hooks", entries are appended to the
//     corresponding per-event hook group in existingRaw.
//   - Idempotency: a wipnote entry is skipped if an entry with the same command
//     already exists in the group.
//   - Stale wipnote detection: if an existing entry's command starts with "wipnote "
//     and differs from the canonical bundled command, it is replaced (not kept).
//   - Non-wipnote existing entries on the same event are always preserved (appended).
//
// existingRaw may be nil or empty (treated as an empty object).
// Returns the merged JSON, indented for readability.
func mergeGeminiHooks(existingRaw, bundledRaw []byte) ([]byte, error) {
	// Parse existing settings as a generic map so we preserve unknown keys.
	existing := map[string]json.RawMessage{}
	if len(existingRaw) > 0 {
		if err := json.Unmarshal(existingRaw, &existing); err != nil {
			return nil, fmt.Errorf("parsing existing settings.json: %w", err)
		}
	}

	// Parse existing hooks section (if any).
	existingHooks := map[string][]geminiHookGroup{}
	if raw, ok := existing["hooks"]; ok {
		if err := json.Unmarshal(raw, &existingHooks); err != nil {
			return nil, fmt.Errorf("parsing hooks in existing settings.json: %w", err)
		}
	}

	// Parse bundled hooks.
	var bundled geminiHooksFile
	if err := json.Unmarshal(bundledRaw, &bundled); err != nil {
		return nil, fmt.Errorf("parsing bundled hooks.json: %w", err)
	}

	// Merge each event.
	for eventName, bundledGroups := range bundled.Hooks {
		existingGroups := existingHooks[eventName]

		for _, bg := range bundledGroups {
			existingGroups = mergeHookGroup(existingGroups, bg)
		}

		existingHooks[eventName] = existingGroups
	}

	// Re-encode the merged hooks section.
	hooksRaw, err := json.Marshal(existingHooks)
	if err != nil {
		return nil, fmt.Errorf("encoding merged hooks: %w", err)
	}
	existing["hooks"] = hooksRaw

	// Re-encode the full settings object with indentation.
	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encoding merged settings.json: %w", err)
	}
	return append(out, '\n'), nil
}

// mergeHookGroup merges a single bundled geminiHookGroup into the existing
// groups for one event. It handles the "wildcard group" pattern used by
// wipnote's hooks.json where each event has exactly one group with matcher "*".
//
// Strategy:
//  1. Find an existing group with the same matcher as the bundled group.
//  2. Within that group, append each bundled hook entry if not already present
//     (same command). If an old wipnote entry exists for the same command prefix,
//     replace it.
//  3. If no matching group exists, append the bundled group as a new group.
func mergeHookGroup(existing []geminiHookGroup, bundled geminiHookGroup) []geminiHookGroup {
	// Find an existing group with the same matcher.
	for i, eg := range existing {
		if eg.Matcher != bundled.Matcher {
			continue
		}
		// Merge entries into this group.
		existing[i].Hooks = mergeHookEntries(eg.Hooks, bundled.Hooks)
		return existing
	}
	// No matching group found — append the entire bundled group.
	return append(existing, bundled)
}

// mergeHookEntries merges bundled hook entries into existing ones.
// Rules per entry:
//   - If an entry with the exact same command already exists → skip (idempotent).
//   - If an existing entry whose command starts with "wipnote " differs from the
//     bundled entry's command → replace it (stale wipnote install).
//   - Otherwise → append the bundled entry.
func mergeHookEntries(existing, bundled []geminiHookEntry) []geminiHookEntry {
	result := make([]geminiHookEntry, len(existing))
	copy(result, existing)

	for _, be := range bundled {
		// Check for exact match (idempotent).
		if hasExactEntry(result, be.Command) {
			continue
		}

		// Check for stale wipnote entry (same wipnote handler, different command).
		replaced := false
		if strings.HasPrefix(be.Command, "wipnote ") {
			for i, ee := range result {
				if strings.HasPrefix(ee.Command, "wipnote ") && ee.Command != be.Command {
					fmt.Printf("  replacing stale wipnote hook: %q → %q\n", ee.Command, be.Command)
					result[i] = be
					replaced = true
					break
				}
			}
		}

		if !replaced {
			fmt.Printf("  adding hook: %q\n", be.Command)
			result = append(result, be)
		}
	}

	return result
}

// hasExactEntry reports whether entries contains an entry with the given command.
func hasExactEntry(entries []geminiHookEntry, command string) bool {
	for _, e := range entries {
		if e.Command == command {
			return true
		}
	}
	return false
}

// installGeminiHooks reads the canonical hook list from the bundled Gemini
// extension tree and merges them into the target settings.json.
//
// extensionName controls the target path (see geminiSettingsJSONPath).
// When dryRun is true, the function prints what would change without writing.
func installGeminiHooks(extensionName string, dryRun bool) error {
	settingsPath := geminiSettingsJSONPath(extensionName)

	// Resolve the bundled Gemini extension tree.
	bundledExtPath, err := resolveSharedTreePath("gemini-extension")
	if err != nil {
		if dryRun {
			fmt.Printf("[dry-run] installGeminiHooks: could not resolve bundled extension (%v); skipping\n", err)
			return nil
		}
		return fmt.Errorf("resolving bundled Gemini extension: %w", err)
	}

	hooksPath := filepath.Join(bundledExtPath, "hooks", "hooks.json")

	if dryRun {
		fmt.Printf("[dry-run] installGeminiHooks: would merge hooks from %s into %s\n", hooksPath, settingsPath)
		return nil
	}

	return installGeminiHooksAt(settingsPath, hooksPath)
}

// installGeminiHooksAt merges the wipnote hooks from hooksPath into settingsPath.
// It is the testable core of installGeminiHooks: tests can call it directly with
// paths under t.TempDir() to avoid touching the real ~/.gemini/.
func installGeminiHooksAt(settingsPath, hooksPath string) error {
	bundledRaw, err := os.ReadFile(hooksPath)
	if err != nil {
		return fmt.Errorf("reading bundled hooks.json at %s: %w", hooksPath, err)
	}

	// Read existing settings.json.
	var existingRaw []byte
	var existingMode os.FileMode = 0o644
	if info, statErr := os.Stat(settingsPath); statErr == nil {
		existingMode = info.Mode()
		existingRaw, err = os.ReadFile(settingsPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", settingsPath, err)
		}
		// Reject malformed JSON early with a clear error.
		if len(existingRaw) > 0 {
			var probe any
			if err := json.Unmarshal(existingRaw, &probe); err != nil {
				return fmt.Errorf("settings.json at %s contains malformed JSON: %w\nFix or remove the file and retry.", settingsPath, err)
			}
		}
	}

	fmt.Printf("Installing wipnote hooks into %s\n", settingsPath)

	merged, err := mergeGeminiHooks(existingRaw, bundledRaw)
	if err != nil {
		return fmt.Errorf("merging hooks: %w", err)
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", filepath.Dir(settingsPath), err)
	}

	// Write atomically via temp file + rename, preserving file mode.
	tmp := settingsPath + ".wipnote-tmp"
	if err := os.WriteFile(tmp, merged, existingMode); err != nil {
		return fmt.Errorf("writing temp file %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, settingsPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming %s → %s: %w", tmp, settingsPath, err)
	}

	fmt.Printf("Hooks installed successfully into %s\n", settingsPath)
	return nil
}
