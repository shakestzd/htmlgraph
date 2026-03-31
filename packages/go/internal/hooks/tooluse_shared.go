package hooks

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"github.com/shakestzd/htmlgraph/internal/db"
)

// featureIDCacheEntry holds the single cached result of GetActiveFeatureID for
// the lifetime of this process invocation. Each hook binary invocation handles
// exactly one CloudEvent, so the (sessionID → featureID) mapping is constant.
// No sync needed — hook handlers run in a single goroutine.
type featureIDCacheEntry struct {
	sessionID string
	featureID string
	populated bool
}

var featureIDCache featureIDCacheEntry

// cachedGetActiveFeatureID returns the active feature ID for sessionID,
// querying the database at most once per process invocation.
func cachedGetActiveFeatureID(database *sql.DB, sessionID string) string {
	if featureIDCache.populated && featureIDCache.sessionID == sessionID {
		return featureIDCache.featureID
	}
	featureID := GetActiveFeatureID(database, sessionID)
	featureIDCache = featureIDCacheEntry{
		sessionID: sessionID,
		featureID: featureID,
		populated: true,
	}
	return featureID
}

// toolUseContext holds resolved identifiers shared by PreToolUse and PostToolUse.
type toolUseContext struct {
	SessionID     string
	FeatureID     string
	AgentID       string
	AgentType     string
	IsSubagent    bool
	ProjectDir    string
	HgDir         string
	IsYoloMode    bool
	ParentEventID string
}

// resolveToolUseContext resolves session, feature, agent identifiers, project
// directory, YOLO mode, and parent event ID from a CloudEvent and database.
// Returns nil when no active session is found, indicating the caller should
// skip all DB operations.
func resolveToolUseContext(event *CloudEvent, database *sql.DB) *toolUseContext {
	start := time.Now()

	sessionID := EnvSessionID(event.SessionID)
	if sessionID == "" {
		return nil
	}

	featureID := cachedGetActiveFeatureID(database, sessionID)
	agentID := resolveAgentID(event)
	isSubagent := isSubagentEvent(event)

	agentType := event.AgentType
	if agentType == "" {
		agentType = os.Getenv("HTMLGRAPH_AGENT_TYPE")
	}

	projectDir := ResolveProjectDir(event.CWD)
	hgDir := filepath.Join(projectDir, ".htmlgraph")
	yolo := isYoloFromEvent(event, hgDir)
	parentEventID := resolveParentEventID(database, sessionID, agentID, isSubagent)

	LogTimed(projectDir, "pretooluse", map[string]string{
		"phase":   "resolve-context",
		"session": sessionID[:minSessionLen(sessionID)],
		"tool":    event.ToolName,
	}, start, "context resolved")

	return &toolUseContext{
		SessionID:     sessionID,
		FeatureID:     featureID,
		AgentID:       agentID,
		AgentType:     agentType,
		IsSubagent:    isSubagent,
		ProjectDir:    projectDir,
		HgDir:         hgDir,
		IsYoloMode:    yolo,
		ParentEventID: parentEventID,
	}
}

// isSubagentEvent returns true when the event originates from a subagent.
// Claude Code sets a non-empty agent_id (not "claude-code") for subagent hooks.
func isSubagentEvent(event *CloudEvent) bool {
	return event.AgentID != "" && event.AgentID != "claude-code"
}

// resolveAgentID returns the effective agent ID: the CloudEvent agent_id when
// present (subagent case), falling back to the env-var-based agent identity.
func resolveAgentID(event *CloudEvent) string {
	if event.AgentID != "" {
		return event.AgentID
	}
	return agentIDFromEnv()
}

// resolveEventAgentID returns the agent ID from the CloudEvent, falling back
// to the env-var-based agent identity. Use this for non-tooluse handlers
// (Stop, TrackEvent, etc.) that receive a raw CloudEvent.
func resolveEventAgentID(event *CloudEvent) string {
	if event.AgentID != "" {
		return event.AgentID
	}
	return agentIDFromEnv()
}

// resolveEventAgentType returns the agent type from the CloudEvent, falling
// back to the HTMLGRAPH_AGENT_TYPE env var.
func resolveEventAgentType(event *CloudEvent) string {
	if event.AgentType != "" {
		return event.AgentType
	}
	return os.Getenv("HTMLGRAPH_AGENT_TYPE")
}

// resolveParentEventID finds the parent event using a multi-step fallback that
// mirrors the Python event_tracker.py logic:
//  1. Env var HTMLGRAPH_PARENT_EVENT (written by SubagentStart)
//  2. For subagents: task_delegation row matching our agent_id (Method 0.5)
//  3. Most recent UserQuery in this session (orchestrator default)
func resolveParentEventID(database *sql.DB, sessionID, agentID string, isSubagent bool) string {
	parentEventID := os.Getenv("HTMLGRAPH_PARENT_EVENT")

	if parentEventID == "" && isSubagent {
		parentEventID, _ = db.FindDelegationByAgent(database, sessionID, agentID)
	}

	if parentEventID == "" {
		parentEventID, _ = db.LatestEventByTool(database, sessionID, "UserQuery")
	}

	return parentEventID
}
