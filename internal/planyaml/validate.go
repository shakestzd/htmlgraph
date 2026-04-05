package planyaml

import "fmt"

// Validate checks a PlanYAML for schema errors. Returns a list of error
// strings. Empty list means the plan is valid.
func Validate(plan *PlanYAML) []string {
	var errs []string
	if plan.Meta.ID == "" {
		errs = append(errs, "meta.id is required")
	}
	if plan.Meta.Title == "" {
		errs = append(errs, "meta.title is required")
	}
	switch plan.Meta.Status {
	case "draft", "review", "finalized":
	default:
		errs = append(errs, fmt.Sprintf("meta.status %q must be draft|review|finalized", plan.Meta.Status))
	}
	if plan.Design.Problem == "" {
		errs = append(errs, "design.problem is required")
	}
	if len(plan.Design.Goals) == 0 {
		errs = append(errs, "design.goals must have at least 1 entry")
	}
	if len(plan.Design.Constraints) == 0 {
		errs = append(errs, "design.constraints must have at least 1 entry")
	}
	nums := map[int]bool{}
	for i, s := range plan.Slices {
		prefix := fmt.Sprintf("slices[%d]", i)
		if s.What == "" {
			errs = append(errs, prefix+".what is required")
		}
		if s.Why == "" {
			errs = append(errs, prefix+".why is required")
		}
		if len(s.Files) == 0 {
			errs = append(errs, prefix+".files must have at least 1 entry")
		}
		if len(s.DoneWhen) == 0 {
			errs = append(errs, prefix+".done_when must have at least 1 entry")
		}
		if s.Tests == "" {
			errs = append(errs, prefix+".tests is required")
		}
		switch s.Effort {
		case "S", "M", "L":
		default:
			errs = append(errs, fmt.Sprintf("%s.effort %q must be S|M|L", prefix, s.Effort))
		}
		switch s.Risk {
		case "Low", "Med", "High":
		default:
			errs = append(errs, fmt.Sprintf("%s.risk %q must be Low|Med|High", prefix, s.Risk))
		}
		if nums[s.Num] {
			errs = append(errs, fmt.Sprintf("%s.num %d is duplicate", prefix, s.Num))
		}
		nums[s.Num] = true
		for _, d := range s.Deps {
			if d == s.Num {
				errs = append(errs, fmt.Sprintf("%s.deps: self-reference %d", prefix, d))
			}
		}
	}
	// Check dep references after collecting all nums.
	for i, s := range plan.Slices {
		for _, d := range s.Deps {
			if !nums[d] {
				errs = append(errs, fmt.Sprintf("slices[%d].deps: references nonexistent slice %d", i, d))
			}
		}
	}
	for i, q := range plan.Questions {
		prefix := fmt.Sprintf("questions[%d]", i)
		if q.Text == "" {
			errs = append(errs, prefix+".text is required")
		}
		if q.Description == "" {
			errs = append(errs, prefix+".description is required")
		}
		if len(q.Options) < 2 {
			errs = append(errs, prefix+".options must have at least 2 entries")
		}
		if q.Recommended != "" {
			found := false
			for _, o := range q.Options {
				if o.Key == q.Recommended {
					found = true
					break
				}
			}
			if !found {
				errs = append(errs, fmt.Sprintf("%s.recommended %q not in options", prefix, q.Recommended))
			}
		}
	}
	return errs
}
