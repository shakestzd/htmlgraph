package templates

// Minimal Mustache renderer using only Go stdlib.
//
// Supports the subset of the Mustache spec needed by HtmlGraph templates:
//   - {{variable}}       -- interpolation (HTML-safe, no escaping)
//   - {{#section}}...{{/section}} -- truthy conditional / list iteration
//   - {{^section}}...{{/section}} -- inverted (falsy) section
//   - {{.}}              -- current item in a list of strings
//
// This avoids an external dependency while keeping templates portable
// between Python (chevron) and Go.

import (
	"fmt"
	"strings"
)

// mustacheRender processes a Mustache template string with the given data map.
func mustacheRender(tmpl string, data map[string]any) (string, error) {
	result, _, err := renderFragment(tmpl, data)
	return result, err
}

// renderFragment processes a template fragment, returning the rendered string
// and the number of bytes consumed from tmpl.
func renderFragment(tmpl string, ctx map[string]any) (string, int, error) {
	var buf strings.Builder
	i := 0
	for i < len(tmpl) {
		// Look for opening tag
		start := strings.Index(tmpl[i:], "{{")
		if start == -1 {
			buf.WriteString(tmpl[i:])
			i = len(tmpl)
			break
		}
		// Write literal text before the tag
		buf.WriteString(tmpl[i : i+start])
		i += start

		// Find closing tag
		end := strings.Index(tmpl[i:], "}}")
		if end == -1 {
			return "", 0, fmt.Errorf("unclosed mustache tag at position %d", i)
		}
		tag := tmpl[i+2 : i+end]
		tag = strings.TrimSpace(tag)
		i += end + 2

		switch {
		case strings.HasPrefix(tag, "#"):
			// Section: {{#key}}...{{/key}}
			key := strings.TrimSpace(tag[1:])
			body, closeLen, err := findSectionBody(tmpl[i:], key)
			if err != nil {
				return "", 0, err
			}
			rendered, err := renderSection(key, body, ctx)
			if err != nil {
				return "", 0, err
			}
			buf.WriteString(rendered)
			i += closeLen

		case strings.HasPrefix(tag, "^"):
			// Inverted section: {{^key}}...{{/key}}
			key := strings.TrimSpace(tag[1:])
			body, closeLen, err := findSectionBody(tmpl[i:], key)
			if err != nil {
				return "", 0, err
			}
			if !isTruthy(ctx[key]) {
				rendered, _, err := renderFragment(body, ctx)
				if err != nil {
					return "", 0, err
				}
				buf.WriteString(rendered)
			}
			i += closeLen

		case strings.HasPrefix(tag, "/"):
			// Closing tag found outside section scan -- should not happen
			// in well-formed input at this level.
			return "", 0, fmt.Errorf("unexpected closing tag {{/%s}}", tag[1:])

		case tag == ".":
			// Current value -- used inside list iteration.
			// The caller should set "." in the context.
			if v, ok := ctx["."]; ok {
				buf.WriteString(fmt.Sprintf("%v", v))
			}

		default:
			// Variable interpolation
			if v, ok := ctx[tag]; ok {
				buf.WriteString(fmt.Sprintf("%v", v))
			}
			// If key is absent, output nothing (Mustache spec).
		}
	}
	return buf.String(), i, nil
}

// findSectionBody finds the body between {{#key}} (already consumed) and {{/key}}.
// It handles nested sections with the same key. Returns the body string and the
// total number of bytes consumed from rest (including the closing tag).
func findSectionBody(rest, key string) (body string, consumed int, err error) {
	openTag := "{{#" + key + "}}"
	closeTag := "{{/" + key + "}}"
	depth := 1
	pos := 0

	for depth > 0 {
		nextOpen := strings.Index(rest[pos:], openTag)
		nextClose := strings.Index(rest[pos:], closeTag)

		if nextClose == -1 {
			return "", 0, fmt.Errorf("unclosed section {{#%s}}", key)
		}

		// If there's a nested open before the close, increase depth
		if nextOpen != -1 && nextOpen < nextClose {
			depth++
			pos += nextOpen + len(openTag)
		} else {
			depth--
			if depth == 0 {
				body = rest[:pos+nextClose]
				consumed = pos + nextClose + len(closeTag)
				return body, consumed, nil
			}
			pos += nextClose + len(closeTag)
		}
	}
	return "", 0, fmt.Errorf("unclosed section {{#%s}}", key)
}

// renderSection handles a truthy section. If the value is a list of maps,
// it iterates. If the value is a list of strings, it iterates with {{.}}.
// If the value is truthy scalar, it renders once.
func renderSection(key, body string, ctx map[string]any) (string, error) {
	val := ctx[key]
	if !isTruthy(val) {
		return "", nil
	}

	// List of maps (e.g., edge_groups, steps)
	if items, ok := val.([]map[string]any); ok {
		var buf strings.Builder
		for _, item := range items {
			// Merge parent context with item context (item overrides)
			merged := mergeCtx(ctx, item)
			rendered, _, err := renderFragment(body, merged)
			if err != nil {
				return "", err
			}
			buf.WriteString(rendered)
		}
		return buf.String(), nil
	}

	// List of strings (e.g., blockers)
	if items, ok := val.([]string); ok {
		var buf strings.Builder
		for _, item := range items {
			itemCtx := mergeCtx(ctx, map[string]any{".": item})
			rendered, _, err := renderFragment(body, itemCtx)
			if err != nil {
				return "", err
			}
			buf.WriteString(rendered)
		}
		return buf.String(), nil
	}

	// Truthy scalar -- render once with parent context
	rendered, _, err := renderFragment(body, ctx)
	return rendered, err
}

// isTruthy returns whether a value should cause a section to render.
func isTruthy(v any) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != ""
	case int:
		return val != 0
	case float64:
		return val != 0
	case []map[string]any:
		return len(val) > 0
	case []string:
		return len(val) > 0
	default:
		return true
	}
}

// mergeCtx creates a new context by merging parent and child.
// Child values override parent values.
func mergeCtx(parent, child map[string]any) map[string]any {
	merged := make(map[string]any, len(parent)+len(child))
	for k, v := range parent {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = v
	}
	return merged
}
