// Package models defines the core data structures for HtmlGraph.
//
// These types mirror the Python Pydantic models in models.py and event_log.py,
// ensuring JSON-compatible serialization so both runtimes can read/write the
// same .htmlgraph/ files and SQLite databases.
package models

// RelationshipType enumerates typed relationships between graph nodes.
type RelationshipType string

const (
	RelBlocks      RelationshipType = "blocks"
	RelBlockedBy   RelationshipType = "blocked_by"
	RelRelatesTo   RelationshipType = "relates_to"
	RelImplements  RelationshipType = "implements"
	RelCausedBy    RelationshipType = "caused_by"
	RelSpawnedFrom RelationshipType = "spawned_from"
	RelImplementedIn RelationshipType = "implemented-in"
)

// WorkType classifies work/activity type for events and sessions.
type WorkType string

const (
	WorkFeature       WorkType = "feature-implementation"
	WorkSpike         WorkType = "spike-investigation"
	WorkBugFix        WorkType = "bug-fix"
	WorkMaintenance   WorkType = "maintenance"
	WorkDocumentation WorkType = "documentation"
	WorkPlanning      WorkType = "planning"
	WorkReview        WorkType = "review"
	WorkAdmin         WorkType = "admin"
)

// SpikeType categorises spike investigations.
type SpikeType string

const (
	SpikeTechnical     SpikeType = "technical"
	SpikeArchitectural SpikeType = "architectural"
	SpikeRisk          SpikeType = "risk"
	SpikeGeneral       SpikeType = "general"
)

// MaintenanceType categorises software maintenance per IEEE standards.
type MaintenanceType string

const (
	MaintCorrective MaintenanceType = "corrective"
	MaintAdaptive   MaintenanceType = "adaptive"
	MaintPerfective MaintenanceType = "perfective"
	MaintPreventive MaintenanceType = "preventive"
)

// NodeStatus represents the lifecycle state of a work item.
type NodeStatus string

const (
	StatusTodo       NodeStatus = "todo"
	StatusInProgress NodeStatus = "in-progress"
	StatusBlocked    NodeStatus = "blocked"
	StatusDone       NodeStatus = "done"
	StatusActive     NodeStatus = "active"
	StatusEnded      NodeStatus = "ended"
	StatusStale      NodeStatus = "stale"
)

// Priority represents the priority level of a work item.
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// EventType enumerates agent event types stored in SQLite.
type EventType string

const (
	EventToolCall       EventType = "tool_call"
	EventToolResult     EventType = "tool_result"
	EventError          EventType = "error"
	EventDelegation     EventType = "delegation"
	EventCompletion     EventType = "completion"
	EventStart          EventType = "start"
	EventEnd            EventType = "end"
	EventCheckPoint     EventType = "check_point"
	EventTaskDelegation EventType = "task_delegation"
	EventTeammateIdle   EventType = "teammate_idle"
	EventTaskCompleted  EventType = "task_completed"
)
