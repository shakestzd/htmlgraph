// Package planyaml defines the YAML data model for HtmlGraph plans.
// It provides Go structs with YAML tags matching the canonical schema
// defined in prototypes/sample_plan.yaml, plus Load/Save/NewPlan helpers.
package planyaml

// PlanYAML is the top-level plan document.
type PlanYAML struct {
	Meta      PlanMeta       `yaml:"meta"`
	Design    PlanDesign     `yaml:"design"`
	Slices    []PlanSlice    `yaml:"slices"`
	Questions []PlanQuestion `yaml:"questions"`
	Critique  *PlanCritique  `yaml:"critique,omitempty"`
}

// PlanMeta holds plan identity and lifecycle metadata.
type PlanMeta struct {
	ID          string `yaml:"id"`
	TrackID     string `yaml:"track_id,omitempty"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	CreatedAt   string `yaml:"created_at"`
	Status      string `yaml:"status"` // draft | review | finalized
	CreatedBy   string `yaml:"created_by,omitempty"`
}

// PlanDesign captures the problem statement, goals, constraints, and
// human approval for the design section.
type PlanDesign struct {
	Problem     string   `yaml:"problem"`
	Goals       []string `yaml:"goals"`
	Constraints []string `yaml:"constraints"`
	Approved    bool     `yaml:"approved"`
	Comment     string   `yaml:"comment"`
}

// PlanSlice is a vertical delivery slice with metadata for effort,
// risk, dependencies, and human approval.
type PlanSlice struct {
	ID       string   `yaml:"id"`
	Num      int      `yaml:"num"`
	Title    string   `yaml:"title"`
	What     string   `yaml:"what"`
	Why      string   `yaml:"why"`
	Files    []string `yaml:"files"`
	Deps     []int    `yaml:"deps"`
	DoneWhen []string `yaml:"done_when"`
	Effort   string   `yaml:"effort"` // S | M | L
	Risk     string   `yaml:"risk"`   // Low | Med | High
	Tests    string   `yaml:"tests"`
	Approved bool     `yaml:"approved"`
	Comment  string   `yaml:"comment"`
}

// PlanQuestion is an open design question with options and an optional answer.
type PlanQuestion struct {
	ID          string           `yaml:"id"`
	Text        string           `yaml:"text"`
	Description string           `yaml:"description"`
	Recommended string           `yaml:"recommended,omitempty"`
	Options     []QuestionOption `yaml:"options"`
	Answer      *string          `yaml:"answer"` // nil = unanswered → "answer: null"
}

// QuestionOption is a selectable choice for a PlanQuestion.
type QuestionOption struct {
	Key   string `yaml:"key"`
	Label string `yaml:"label"`
}

// PlanCritique holds the multi-reviewer critique section.
type PlanCritique struct {
	ReviewedAt  string               `yaml:"reviewed_at"`
	Reviewers   []string             `yaml:"reviewers"`
	Assumptions []CritiqueAssumption `yaml:"assumptions"`
	Critics     []CriticSection      `yaml:"critics"`
	Risks       []CritiqueRisk       `yaml:"risks"`
	Synthesis   string               `yaml:"synthesis"`
}

// CritiqueAssumption is a single assumption with verification status.
type CritiqueAssumption struct {
	ID       string `yaml:"id"`
	Status   string `yaml:"status"` // verified|plausible|unverified|questionable|falsified
	Text     string `yaml:"text"`
	Evidence string `yaml:"evidence"`
}

// CriticSection groups critic feedback under a titled reviewer.
type CriticSection struct {
	Title    string             `yaml:"title"`
	Sections []CriticSubsection `yaml:"sections"`
}

// CriticSubsection is a heading with a list of critic items.
type CriticSubsection struct {
	Heading string       `yaml:"heading"`
	Items   []CriticItem `yaml:"items"`
}

// CriticItem is a single badged feedback entry.
type CriticItem struct {
	Badge string `yaml:"badge"`
	Kind  string `yaml:"kind"` // success|warn|danger|info
	Text  string `yaml:"text"`
}

// CritiqueRisk records a risk with severity and mitigation strategy.
type CritiqueRisk struct {
	Risk       string `yaml:"risk"`
	Severity   string `yaml:"severity"` // High|Medium|Low
	Mitigation string `yaml:"mitigation"`
}
