package planyaml

import (
	"strings"
	"testing"
)

// validPlan returns a fully-populated valid plan for use in tests.
func validPlan() *PlanYAML {
	return &PlanYAML{
		Meta: PlanMeta{
			ID:     "plan-abc12345",
			Title:  "Test Plan",
			Status: "draft",
		},
		Design: PlanDesign{
			Problem:     "A real problem to solve.",
			Goals:       []string{"Goal 1"},
			Constraints: []string{"Constraint 1"},
		},
		Slices: []PlanSlice{
			{
				Num:      1,
				What:     "Build the thing.",
				Why:      "Because it matters.",
				Files:    []string{"internal/foo/bar.go"},
				DoneWhen: []string{"Tests pass"},
				Tests:    "Unit: it works",
				Effort:   "S",
				Risk:     "Low",
				Deps:     []int{},
			},
			{
				Num:      2,
				What:     "Integrate the thing.",
				Why:      "Because end-to-end matters.",
				Files:    []string{"internal/foo/baz.go"},
				DoneWhen: []string{"Integration test passes"},
				Tests:    "Integration: full flow works",
				Effort:   "M",
				Risk:     "Med",
				Deps:     []int{1},
			},
		},
		Questions: []PlanQuestion{
			{
				Text:        "Which approach?",
				Description: "We need to decide between A and B.",
				Recommended: "opt-a",
				Options: []QuestionOption{
					{Key: "opt-a", Label: "Option A"},
					{Key: "opt-b", Label: "Option B"},
				},
			},
		},
	}
}

func TestValidate_ValidPlan(t *testing.T) {
	plan := validPlan()
	errs := Validate(plan)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidate_EmptyPlan_NoSlices(t *testing.T) {
	// A plan with no slices and no questions is valid as long as meta/design are okay.
	plan := &PlanYAML{
		Meta: PlanMeta{
			ID:     "plan-empty123",
			Title:  "Empty Plan",
			Status: "draft",
		},
		Design: PlanDesign{
			Problem:     "A problem.",
			Goals:       []string{"Goal 1"},
			Constraints: []string{"Constraint 1"},
		},
		Slices:    []PlanSlice{},
		Questions: []PlanQuestion{},
	}
	errs := Validate(plan)
	if len(errs) != 0 {
		t.Errorf("expected no errors for empty plan, got: %v", errs)
	}
}

func TestValidate_MissingMetaID(t *testing.T) {
	plan := validPlan()
	plan.Meta.ID = ""
	errs := Validate(plan)
	assertContainsError(t, errs, "meta.id")
}

func TestValidate_MissingMetaTitle(t *testing.T) {
	plan := validPlan()
	plan.Meta.Title = ""
	errs := Validate(plan)
	assertContainsError(t, errs, "meta.title")
}

func TestValidate_InvalidMetaStatus(t *testing.T) {
	plan := validPlan()
	plan.Meta.Status = "pending"
	errs := Validate(plan)
	assertContainsError(t, errs, "meta.status")
}

func TestValidate_ValidMetaStatuses(t *testing.T) {
	for _, status := range []string{"draft", "review", "finalized"} {
		plan := validPlan()
		plan.Meta.Status = status
		errs := Validate(plan)
		for _, e := range errs {
			if strings.Contains(e, "meta.status") {
				t.Errorf("status %q should be valid, got error: %s", status, e)
			}
		}
	}
}

func TestValidate_MissingDesignProblem(t *testing.T) {
	plan := validPlan()
	plan.Design.Problem = ""
	errs := Validate(plan)
	assertContainsError(t, errs, "design.problem")
}

func TestValidate_MissingDesignGoals(t *testing.T) {
	plan := validPlan()
	plan.Design.Goals = []string{}
	errs := Validate(plan)
	assertContainsError(t, errs, "design.goals")
}

func TestValidate_MissingDesignConstraints(t *testing.T) {
	plan := validPlan()
	plan.Design.Constraints = []string{}
	errs := Validate(plan)
	assertContainsError(t, errs, "design.constraints")
}

func TestValidate_SliceMissingWhat(t *testing.T) {
	plan := validPlan()
	plan.Slices[0].What = ""
	errs := Validate(plan)
	assertContainsError(t, errs, "slices[0].what")
}

func TestValidate_SliceMissingWhy(t *testing.T) {
	plan := validPlan()
	plan.Slices[0].Why = ""
	errs := Validate(plan)
	assertContainsError(t, errs, "slices[0].why")
}

func TestValidate_SliceMissingFiles(t *testing.T) {
	plan := validPlan()
	plan.Slices[0].Files = []string{}
	errs := Validate(plan)
	assertContainsError(t, errs, "slices[0].files")
}

func TestValidate_SliceMissingDoneWhen(t *testing.T) {
	plan := validPlan()
	plan.Slices[0].DoneWhen = []string{}
	errs := Validate(plan)
	assertContainsError(t, errs, "slices[0].done_when")
}

func TestValidate_SliceMissingTests(t *testing.T) {
	plan := validPlan()
	plan.Slices[0].Tests = ""
	errs := Validate(plan)
	assertContainsError(t, errs, "slices[0].tests")
}

func TestValidate_SliceInvalidEffort(t *testing.T) {
	plan := validPlan()
	plan.Slices[0].Effort = "XL"
	errs := Validate(plan)
	assertContainsError(t, errs, "slices[0].effort")
}

func TestValidate_SliceValidEfforts(t *testing.T) {
	for _, effort := range []string{"S", "M", "L"} {
		plan := validPlan()
		plan.Slices[0].Effort = effort
		errs := Validate(plan)
		for _, e := range errs {
			if strings.Contains(e, "slices[0].effort") {
				t.Errorf("effort %q should be valid, got error: %s", effort, e)
			}
		}
	}
}

func TestValidate_SliceInvalidRisk(t *testing.T) {
	plan := validPlan()
	plan.Slices[0].Risk = "Critical"
	errs := Validate(plan)
	assertContainsError(t, errs, "slices[0].risk")
}

func TestValidate_SliceValidRisks(t *testing.T) {
	for _, risk := range []string{"Low", "Med", "High"} {
		plan := validPlan()
		plan.Slices[0].Risk = risk
		errs := Validate(plan)
		for _, e := range errs {
			if strings.Contains(e, "slices[0].risk") {
				t.Errorf("risk %q should be valid, got error: %s", risk, e)
			}
		}
	}
}

func TestValidate_DuplicateSliceNums(t *testing.T) {
	plan := validPlan()
	plan.Slices[1].Num = 1 // duplicate of slice[0].Num
	errs := Validate(plan)
	assertContainsError(t, errs, "duplicate")
}

func TestValidate_SliceDepsNonexistentNum(t *testing.T) {
	plan := validPlan()
	plan.Slices[1].Deps = []int{99} // num 99 doesn't exist
	errs := Validate(plan)
	assertContainsError(t, errs, "slices[1].deps")
}

func TestValidate_SliceSelfReferencingDep(t *testing.T) {
	plan := validPlan()
	plan.Slices[0].Deps = []int{1} // slice num=1 referencing itself
	errs := Validate(plan)
	assertContainsError(t, errs, "slices[0].deps")
}

func TestValidate_QuestionMissingText(t *testing.T) {
	plan := validPlan()
	plan.Questions[0].Text = ""
	errs := Validate(plan)
	assertContainsError(t, errs, "questions[0].text")
}

func TestValidate_QuestionMissingDescription(t *testing.T) {
	plan := validPlan()
	plan.Questions[0].Description = ""
	errs := Validate(plan)
	assertContainsError(t, errs, "questions[0].description")
}

func TestValidate_QuestionFewerThanTwoOptions(t *testing.T) {
	plan := validPlan()
	plan.Questions[0].Options = []QuestionOption{
		{Key: "opt-a", Label: "Option A"},
	}
	errs := Validate(plan)
	assertContainsError(t, errs, "questions[0].options")
}

func TestValidate_QuestionNoOptions(t *testing.T) {
	plan := validPlan()
	plan.Questions[0].Options = []QuestionOption{}
	errs := Validate(plan)
	assertContainsError(t, errs, "questions[0].options")
}

func TestValidate_QuestionInvalidRecommended(t *testing.T) {
	plan := validPlan()
	plan.Questions[0].Recommended = "nonexistent-key"
	errs := Validate(plan)
	assertContainsError(t, errs, "questions[0].recommended")
}

func TestValidate_QuestionEmptyRecommendedIsValid(t *testing.T) {
	plan := validPlan()
	plan.Questions[0].Recommended = ""
	errs := Validate(plan)
	for _, e := range errs {
		if strings.Contains(e, "questions[0].recommended") {
			t.Errorf("empty recommended should be valid, got error: %s", e)
		}
	}
}

// assertContainsError checks that at least one error message contains the given substring.
func assertContainsError(t *testing.T, errs []string, substr string) {
	t.Helper()
	for _, e := range errs {
		if strings.Contains(e, substr) {
			return
		}
	}
	t.Errorf("expected an error containing %q, got: %v", substr, errs)
}

// ---- v2 slice-card tests ----

// validV2Plan returns a fully-populated valid v2 plan with slice-local
// questions, critic_revisions, and lifecycle states.
func validV2Plan() *PlanYAML {
	return &PlanYAML{
		Meta: PlanMeta{
			ID:     "plan-v2test01",
			Title:  "V2 Slice-Card Test Plan",
			Status: "active",
		},
		Design: PlanDesign{
			Problem:     "A real problem to solve.",
			Goals:       []string{"Goal 1"},
			Constraints: []string{"Constraint 1"},
		},
		Slices: []PlanSlice{
			{
				Num:             1,
				What:            "Build the thing.",
				Why:             "Because it matters.",
				Files:           []string{"internal/foo/bar.go"},
				DoneWhen:        []string{"Tests pass"},
				Tests:           "Unit: it works",
				Effort:          "S",
				Risk:            "Low",
				Deps:            []int{},
				ApprovalStatus:  "approved",
				ExecutionStatus: "done",
				Questions: []SliceQuestion{
					{
						ID:   "sq-1",
						Text: "Should we use interface{}?",
					},
				},
				CriticRevisions: []CriticRevision{
					{
						Source:   "haiku",
						Severity: "LOW",
						Summary:  "Minor style nit.",
					},
				},
			},
			{
				Num:             2,
				What:            "Integrate the thing.",
				Why:             "Because end-to-end matters.",
				Files:           []string{"internal/foo/baz.go"},
				DoneWhen:        []string{"Integration test passes"},
				Tests:           "Integration: full flow works",
				Effort:          "M",
				Risk:            "Med",
				Deps:            []int{1},
				ApprovalStatus:  "pending",
				ExecutionStatus: "not_started",
				Questions:       []SliceQuestion{},
				CriticRevisions: []CriticRevision{},
			},
		},
		Questions: []PlanQuestion{
			{
				Text:        "Which approach?",
				Description: "We need to decide between A and B.",
				Recommended: "opt-a",
				Options: []QuestionOption{
					{Key: "opt-a", Label: "Option A"},
					{Key: "opt-b", Label: "Option B"},
				},
			},
		},
	}
}

func TestValidate_V2Plan_Valid(t *testing.T) {
	plan := validV2Plan()
	errs := Validate(plan)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid v2 plan, got: %v", errs)
	}
}

func TestValidate_MetaStatusActive(t *testing.T) {
	plan := validPlan()
	plan.Meta.Status = "active"
	errs := Validate(plan)
	for _, e := range errs {
		if strings.Contains(e, "meta.status") {
			t.Errorf("status 'active' should be valid, got error: %s", e)
		}
	}
}

func TestValidate_MetaStatusCompleted(t *testing.T) {
	plan := validPlan()
	plan.Meta.Status = "completed"
	errs := Validate(plan)
	for _, e := range errs {
		if strings.Contains(e, "meta.status") {
			t.Errorf("status 'completed' should be valid, got error: %s", e)
		}
	}
}

func TestValidate_AllMetaStatuses(t *testing.T) {
	for _, status := range []string{"draft", "review", "finalized", "active", "completed"} {
		plan := validPlan()
		plan.Meta.Status = status
		errs := Validate(plan)
		for _, e := range errs {
			if strings.Contains(e, "meta.status") {
				t.Errorf("status %q should be valid, got error: %s", status, e)
			}
		}
	}
}

func TestValidate_DuplicateSliceIDs(t *testing.T) {
	plan := validPlan()
	plan.Slices[0].ID = "feat-duplicate"
	plan.Slices[1].ID = "feat-duplicate"
	errs := Validate(plan)
	assertContainsError(t, errs, "duplicate")
}

func TestValidate_DuplicateQuestionIDsWithinSlice(t *testing.T) {
	plan := validV2Plan()
	plan.Slices[0].Questions = []SliceQuestion{
		{ID: "sq-dup", Text: "First question"},
		{ID: "sq-dup", Text: "Second question"},
	}
	errs := Validate(plan)
	assertContainsError(t, errs, "duplicate")
}

func TestValidate_CriticRevisionMissingSource(t *testing.T) {
	plan := validV2Plan()
	plan.Slices[0].CriticRevisions = []CriticRevision{
		{Source: "", Severity: "HIGH", Summary: "A summary"},
	}
	errs := Validate(plan)
	assertContainsError(t, errs, "source")
}

func TestValidate_CriticRevisionMissingSeverity(t *testing.T) {
	plan := validV2Plan()
	plan.Slices[0].CriticRevisions = []CriticRevision{
		{Source: "haiku", Severity: "", Summary: "A summary"},
	}
	errs := Validate(plan)
	assertContainsError(t, errs, "severity")
}

func TestValidate_CriticRevisionMissingSummary(t *testing.T) {
	plan := validV2Plan()
	plan.Slices[0].CriticRevisions = []CriticRevision{
		{Source: "haiku", Severity: "HIGH", Summary: ""},
	}
	errs := Validate(plan)
	assertContainsError(t, errs, "summary")
}

func TestValidate_InvalidApprovalStatus(t *testing.T) {
	plan := validV2Plan()
	plan.Slices[0].ApprovalStatus = "unknown-status"
	errs := Validate(plan)
	assertContainsError(t, errs, "approval_status")
}

func TestValidate_ValidApprovalStatuses(t *testing.T) {
	for _, status := range []string{"", "pending", "approved", "rejected", "changes_requested"} {
		plan := validV2Plan()
		plan.Slices[0].ApprovalStatus = status
		errs := Validate(plan)
		for _, e := range errs {
			if strings.Contains(e, "approval_status") {
				t.Errorf("approval_status %q should be valid, got error: %s", status, e)
			}
		}
	}
}

func TestValidate_InvalidExecutionStatus(t *testing.T) {
	plan := validV2Plan()
	plan.Slices[0].ExecutionStatus = "running"
	errs := Validate(plan)
	assertContainsError(t, errs, "execution_status")
}

func TestValidate_ValidExecutionStatuses(t *testing.T) {
	for _, status := range []string{"", "not_started", "promoted", "in_progress", "done", "blocked", "superseded"} {
		plan := validV2Plan()
		plan.Slices[0].ExecutionStatus = status
		errs := Validate(plan)
		for _, e := range errs {
			if strings.Contains(e, "execution_status") {
				t.Errorf("execution_status %q should be valid, got error: %s", status, e)
			}
		}
	}
}

func TestValidate_LegacyPlanRegression(t *testing.T) {
	// A legacy plan (no v2 fields) should still validate without errors.
	plan := &PlanYAML{
		Meta: PlanMeta{
			ID:     "plan-legacy01",
			Title:  "Legacy Plan",
			Status: "draft",
		},
		Design: PlanDesign{
			Problem:     "An old problem.",
			Goals:       []string{"Legacy goal"},
			Constraints: []string{"Legacy constraint"},
		},
		Slices: []PlanSlice{
			{
				Num:      1,
				What:     "Do the legacy thing.",
				Why:      "Because legacy.",
				Files:    []string{"internal/legacy/foo.go"},
				DoneWhen: []string{"It works"},
				Tests:    "Manual: smoke test",
				Effort:   "M",
				Risk:     "High",
				Deps:     []int{},
			},
		},
		Questions: []PlanQuestion{},
	}
	errs := Validate(plan)
	if len(errs) != 0 {
		t.Errorf("legacy plan should validate without errors, got: %v", errs)
	}
}
