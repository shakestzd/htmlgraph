package plantmpl

import "fmt"

// BuildFromTopic creates a PlanPage for a new plan created from a
// free-text topic title. This is the plan-first workflow where no
// work item exists yet.
func BuildFromTopic(planID, title, description, date string) *PlanPage {
	return &PlanPage{
		PlanID:      planID,
		Title:       title,
		Description: description,
		Date:        date,
		Status:      "draft",
		Assets:      &AssetRegistry{},
	}
}

// BuildFromWorkItem creates a PlanPage for a retroactive plan generated
// from an existing work item (track, feature, bug, or spike).
func BuildFromWorkItem(planID, featureID, title, description, date string) *PlanPage {
	return &PlanPage{
		PlanID:      planID,
		FeatureID:   featureID,
		Title:       title,
		Description: description,
		Date:        date,
		Status:      "draft",
		Assets:      &AssetRegistry{},
	}
}

// SectionsJSON returns the JavaScript array literal of section IDs used
// by the CRISPI interactive plan. Always includes "design" and "outline",
// plus one entry per slice.
func (p *PlanPage) SectionsJSON() string {
	sections := []string{`"design"`, `"outline"`}
	for _, sc := range p.Slices {
		sections = append(sections, fmt.Sprintf(`"slice-%d"`, sc.Num))
	}
	result := "["
	for i, s := range sections {
		if i > 0 {
			result += ","
		}
		result += s
	}
	result += "]"
	return result
}

// SliceCount returns the number of slices in the plan.
func (p *PlanPage) SliceCount() int {
	return len(p.Slices)
}

// TotalSections returns the total number of approvable sections
// (design + outline + each slice).
func (p *PlanPage) TotalSections() int {
	return 2 + len(p.Slices)
}

// PlanMeta returns the human-readable metadata string shown in the
// plan header (e.g. "3 slices - Created 2026-04-04").
func (p *PlanPage) PlanMeta() string {
	return fmt.Sprintf("%d slices \u00b7 Created %s", p.SliceCount(), p.Date)
}
