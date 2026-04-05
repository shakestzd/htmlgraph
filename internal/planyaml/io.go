package planyaml

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// NewPlan creates a PlanYAML with sensible defaults: status "draft",
// empty design/slices/questions, nil critique, and CreatedAt set to today.
func NewPlan(id, title, description string) *PlanYAML {
	return &PlanYAML{
		Meta: PlanMeta{
			ID:          id,
			Title:       title,
			Description: description,
			CreatedAt:   time.Now().UTC().Format("2006-01-02"),
			Status:      "draft",
		},
		Design:    PlanDesign{},
		Slices:    []PlanSlice{},
		Questions: []PlanQuestion{},
		Critique:  nil,
	}
}

// Load reads a YAML plan file from disk and unmarshals it.
func Load(path string) (*PlanYAML, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read plan YAML: %w", err)
	}

	var plan PlanYAML
	if err := yaml.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("unmarshal plan YAML: %w", err)
	}

	return &plan, nil
}

// Save marshals the plan to YAML and writes it to the given path.
func Save(path string, plan *PlanYAML) error {
	data, err := yaml.Marshal(plan)
	if err != nil {
		return fmt.Errorf("marshal plan YAML: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write plan YAML: %w", err)
	}

	return nil
}
