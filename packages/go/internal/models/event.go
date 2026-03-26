package models

import (
	"time"
)

// EventRecord mirrors the Python EventRecord from event_log.py.
// It is the unit of append-only JSONL logging.
type EventRecord struct {
	EventID   string    `json:"event_id"`
	Timestamp time.Time `json:"timestamp"`
	SessionID string    `json:"session_id"`
	Agent     string    `json:"agent"`
	Tool      string    `json:"tool"`
	Summary   string    `json:"summary"`
	Success   bool      `json:"success"`

	FeatureID       string   `json:"feature_id,omitempty"`
	DriftScore      *float64 `json:"drift_score,omitempty"`
	StartCommit     string   `json:"start_commit,omitempty"`
	ContinuedFrom   string   `json:"continued_from,omitempty"`
	WorkType        string   `json:"work_type,omitempty"`
	SessionStatus   string   `json:"session_status,omitempty"`
	FilePaths       []string `json:"file_paths"`
	Payload         map[string]any `json:"payload,omitempty"`
	ParentSessionID string   `json:"parent_session_id,omitempty"`

	// Multi-AI delegation tracking
	DelegatedToAI          string   `json:"delegated_to_ai,omitempty"`
	TaskID                 string   `json:"task_id,omitempty"`
	TaskStatus             string   `json:"task_status,omitempty"`
	ModelSelected          string   `json:"model_selected,omitempty"`
	ComplexityLevel        string   `json:"complexity_level,omitempty"`
	BudgetMode             string   `json:"budget_mode,omitempty"`
	ExecutionDurationSecs  *float64 `json:"execution_duration_seconds,omitempty"`
	TokensEstimated        *int     `json:"tokens_estimated,omitempty"`
	TokensActual           *int     `json:"tokens_actual,omitempty"`
	CostUSD                *float64 `json:"cost_usd,omitempty"`
	TaskFindings           string   `json:"task_findings,omitempty"`
}

// AgentEvent mirrors a row in the agent_events SQLite table.
type AgentEvent struct {
	EventID     string    `json:"event_id"`
	AgentID     string    `json:"agent_id"`
	EventType   EventType `json:"event_type"`
	Timestamp   time.Time `json:"timestamp"`
	ToolName    string    `json:"tool_name,omitempty"`
	InputSummary  string  `json:"input_summary,omitempty"`
	ToolInput     string  `json:"tool_input,omitempty"`
	OutputSummary string  `json:"output_summary,omitempty"`
	Context       string  `json:"context,omitempty"`
	SessionID     string  `json:"session_id"`
	FeatureID     string  `json:"feature_id,omitempty"`
	ParentAgentID string  `json:"parent_agent_id,omitempty"`
	ParentEventID string  `json:"parent_event_id,omitempty"`
	SubagentType  string  `json:"subagent_type,omitempty"`
	ChildSpikeCount int   `json:"child_spike_count"`
	CostTokens    int     `json:"cost_tokens"`
	ExecDuration  float64 `json:"execution_duration_seconds"`
	Status        string  `json:"status"`
	Model         string  `json:"model,omitempty"`
	ClaudeTaskID  string  `json:"claude_task_id,omitempty"`
	Source        string  `json:"source"`
	StepID        string  `json:"step_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
