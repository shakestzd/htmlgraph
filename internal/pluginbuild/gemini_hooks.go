package pluginbuild

import (
	"path/filepath"
)

// Register hooks.json emission as a Gemini sub-emitter. Gemini's hook schema
// mirrors Claude's (same `{"hooks": {"<Event>": [...]}}` shape), so we reuse
// the orderedHookMap + claudeMatcherGroup + claudeHookEntry types declared in
// claude.go and only change the target filter passed to AppliesTo.
func init() { geminiSubEmitters = append(geminiSubEmitters, emitGeminiHooks) }

// emitGeminiHooks writes <outDir>/<t.HooksPath> using the Gemini-filtered
// subset of m.Hooks.Events. Declaration order from the manifest is preserved
// so matcher groups for the same event name stay adjacent. When no events
// are tagged for Gemini, the file is not emitted — this keeps skeleton-only
// builds (pre-Phase-3 fixtures, tests) from writing stub hooks.json files.
func emitGeminiHooks(m *Manifest, repoRoot, outDir string, t Target) error {
	hooks := map[string][]claudeMatcherGroup{}
	order := []string{}

	for _, e := range m.Hooks.Events {
		if !e.AppliesTo("gemini") {
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

	if len(order) == 0 {
		return nil
	}
	return writeJSON(filepath.Join(outDir, t.HooksPath), orderedHookMap{keys: order, values: hooks})
}
