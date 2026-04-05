package plantmpl

import (
	"html/template"
	"io"
)

// DependencyGraph renders the interactive dependency graph zone showing
// slice relationships and approval status.
type DependencyGraph struct {
	Nodes []GraphNode
}

// GraphNode represents a single node in the dependency graph.
type GraphNode struct {
	Num    int
	Name   string
	Status string // "pending", "approved", etc.
	Deps   string // comma-separated dep numbers
	Files  int
}

var depGraphTmpl = template.Must(
	template.ParseFS(templateFS, "templates/dependency_graph.gohtml"),
)

// Render writes the dependency graph zone HTML to w.
func (g *DependencyGraph) Render(w io.Writer) error {
	return depGraphTmpl.Execute(w, g)
}
