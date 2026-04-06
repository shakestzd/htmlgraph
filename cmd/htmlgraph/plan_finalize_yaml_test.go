package main

import (
	"testing"

	"github.com/shakestzd/htmlgraph/internal/planyaml"
)

func TestApplyAmendments(t *testing.T) {
	plan := &planyaml.PlanYAML{
		Slices: []planyaml.PlanSlice{
			{Num: 1, Title: "Original", DoneWhen: []string{"existing"}, Files: []string{"a.go"}, Effort: "S", Risk: "Low"},
			{Num: 2, Title: "Slice 2", DoneWhen: []string{"criterion"}, Effort: "M"},
		},
	}

	amendments := []planAmendment{
		{SliceNum: 1, Field: "done_when", Operation: "add", Content: "new criterion"},
		{SliceNum: 1, Field: "title", Operation: "set", Content: "Updated Title"},
		{SliceNum: 1, Field: "files", Operation: "add", Content: "b.go"},
		{SliceNum: 2, Field: "effort", Operation: "set", Content: "L"},
		{SliceNum: 2, Field: "done_when", Operation: "remove", Content: "criterion"},
		{SliceNum: 99, Field: "title", Operation: "set", Content: "Missing"}, // should be skipped
	}

	applyAmendments(plan, amendments)

	// Verify slice 1
	if plan.Slices[0].Title != "Updated Title" {
		t.Errorf("slice 1 title: got %q, want %q", plan.Slices[0].Title, "Updated Title")
	}
	if len(plan.Slices[0].DoneWhen) != 2 {
		t.Errorf("slice 1 done_when len: got %d, want 2", len(plan.Slices[0].DoneWhen))
	} else if plan.Slices[0].DoneWhen[1] != "new criterion" {
		t.Errorf("slice 1 done_when[1]: got %q, want %q", plan.Slices[0].DoneWhen[1], "new criterion")
	}
	if len(plan.Slices[0].Files) != 2 {
		t.Errorf("slice 1 files len: got %d, want 2", len(plan.Slices[0].Files))
	} else if plan.Slices[0].Files[1] != "b.go" {
		t.Errorf("slice 1 files[1]: got %q, want %q", plan.Slices[0].Files[1], "b.go")
	}

	// Verify slice 2
	if plan.Slices[1].Effort != "L" {
		t.Errorf("slice 2 effort: got %q, want %q", plan.Slices[1].Effort, "L")
	}
	if len(plan.Slices[1].DoneWhen) != 0 {
		t.Errorf("slice 2 done_when len: got %d, want 0 (criterion removed)", len(plan.Slices[1].DoneWhen))
	}
}

func TestApplyAmendments_SetScalars(t *testing.T) {
	plan := &planyaml.PlanYAML{
		Slices: []planyaml.PlanSlice{
			{Num: 3, Title: "Old", What: "old what", Why: "old why", Risk: "Low"},
		},
	}

	amendments := []planAmendment{
		{SliceNum: 3, Field: "what", Operation: "set", Content: "new what"},
		{SliceNum: 3, Field: "why", Operation: "set", Content: "new why"},
		{SliceNum: 3, Field: "risk", Operation: "set", Content: "High"},
	}

	applyAmendments(plan, amendments)

	if plan.Slices[0].What != "new what" {
		t.Errorf("what: got %q, want %q", plan.Slices[0].What, "new what")
	}
	if plan.Slices[0].Why != "new why" {
		t.Errorf("why: got %q, want %q", plan.Slices[0].Why, "new why")
	}
	if plan.Slices[0].Risk != "High" {
		t.Errorf("risk: got %q, want %q", plan.Slices[0].Risk, "High")
	}
}

func TestRemoveStr(t *testing.T) {
	result := removeStr([]string{"a", "b", "c"}, "b")
	if len(result) != 2 || result[0] != "a" || result[1] != "c" {
		t.Errorf("removeStr: got %v, want [a c]", result)
	}

	// Remove non-existent element — slice unchanged
	result2 := removeStr([]string{"x"}, "y")
	if len(result2) != 1 || result2[0] != "x" {
		t.Errorf("removeStr non-existent: got %v, want [x]", result2)
	}

	// Remove from empty slice
	result3 := removeStr([]string{}, "z")
	if len(result3) != 0 {
		t.Errorf("removeStr empty: got %v, want []", result3)
	}

	// Remove duplicate occurrences
	result4 := removeStr([]string{"a", "b", "a"}, "a")
	if len(result4) != 1 || result4[0] != "b" {
		t.Errorf("removeStr duplicates: got %v, want [b]", result4)
	}
}
