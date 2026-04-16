package main

import (
	"database/sql"
	"testing"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
)

func setupAgentTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := dbpkg.Open(":memory:")
	if err != nil {
		t.Fatalf("open agent test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestLoadGraphNodesIncludesAgents verifies that loadGraphNodes returns agent
// nodes derived from the agent_lineage_trace table.
func TestLoadGraphNodesIncludesAgents(t *testing.T) {
	db := setupAgentTestDB(t)

	// Insert lineage rows with two distinct agent names.
	_, err := db.Exec(`
		INSERT INTO agent_lineage_trace (trace_id, root_session_id, session_id, agent_name, feature_id)
		VALUES
			('t1', 'root-1', 'sess-1', 'htmlgraph:researcher', 'feat-aaa'),
			('t2', 'root-1', 'sess-2', 'htmlgraph:sonnet-coder', 'feat-bbb'),
			('t3', 'root-2', 'sess-3', 'htmlgraph:researcher', 'feat-aaa')`)
	if err != nil {
		t.Fatalf("seed lineage: %v", err)
	}

	nodes, _, err := loadGraphNodes(db)
	if err != nil {
		t.Fatalf("loadGraphNodes: %v", err)
	}

	agentNodes := make(map[string]graphNode)
	for _, n := range nodes {
		if n.Type == "agent" {
			agentNodes[n.ID] = n
		}
	}

	if len(agentNodes) != 2 {
		t.Errorf("want 2 agent nodes, got %d: %v", len(agentNodes), agentNodes)
	}

	for _, name := range []string{"htmlgraph:researcher", "htmlgraph:sonnet-coder"} {
		n, ok := agentNodes[name]
		if !ok {
			t.Errorf("agent node %q not found", name)
			continue
		}
		if n.Title != name {
			t.Errorf("agent node %q title = %q, want %q", name, n.Title, name)
		}
		if n.Activity <= 0 {
			t.Errorf("agent node %q activity = %d, want > 0", name, n.Activity)
		}
	}
}

// TestLoadGraphNodesAgentDeduplication verifies that agent nodes are
// deduplicated by agent_name even when appearing in multiple rows.
func TestLoadGraphNodesAgentDeduplication(t *testing.T) {
	db := setupAgentTestDB(t)

	// Same agent name appearing 5 times across rows.
	_, err := db.Exec(`
		INSERT INTO agent_lineage_trace (trace_id, root_session_id, session_id, agent_name)
		VALUES
			('t1', 'root-1', 'sess-1', 'htmlgraph:researcher'),
			('t2', 'root-1', 'sess-2', 'htmlgraph:researcher'),
			('t3', 'root-2', 'sess-3', 'htmlgraph:researcher'),
			('t4', 'root-3', 'sess-4', 'htmlgraph:researcher'),
			('t5', 'root-3', 'sess-5', 'htmlgraph:researcher')`)
	if err != nil {
		t.Fatalf("seed lineage: %v", err)
	}

	nodes, _, err := loadGraphNodes(db)
	if err != nil {
		t.Fatalf("loadGraphNodes: %v", err)
	}

	count := 0
	for _, n := range nodes {
		if n.Type == "agent" && n.ID == "htmlgraph:researcher" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("want exactly 1 agent node for htmlgraph:researcher, got %d", count)
	}
}

// TestLoadAgentEdgesRanAs verifies that loadAgentEdges produces ran_as edges
// from agent to session.
func TestLoadAgentEdgesRanAs(t *testing.T) {
	db := setupAgentTestDB(t)

	_, err := db.Exec(`
		INSERT INTO agent_lineage_trace (trace_id, root_session_id, session_id, agent_name, feature_id)
		VALUES
			('t1', 'root-1', 'sess-a', 'htmlgraph:researcher', ''),
			('t2', 'root-1', 'sess-b', 'htmlgraph:sonnet-coder', '')`)
	if err != nil {
		t.Fatalf("seed lineage: %v", err)
	}

	edges := loadAgentEdges(db)

	ranAs := make(map[string]string) // agent -> session
	for _, e := range edges {
		if e.Type == "ran_as" {
			ranAs[e.Source] = e.Target
		}
	}

	if ranAs["htmlgraph:researcher"] != "sess-a" {
		t.Errorf("ran_as edge: want htmlgraph:researcher -> sess-a, got %v", ranAs["htmlgraph:researcher"])
	}
	if ranAs["htmlgraph:sonnet-coder"] != "sess-b" {
		t.Errorf("ran_as edge: want htmlgraph:sonnet-coder -> sess-b, got %v", ranAs["htmlgraph:sonnet-coder"])
	}
}

// TestLoadAgentEdgesWorkedOn verifies that loadAgentEdges produces worked_on
// edges from agent to feature when feature_id is set.
func TestLoadAgentEdgesWorkedOn(t *testing.T) {
	db := setupAgentTestDB(t)

	_, err := db.Exec(`
		INSERT INTO agent_lineage_trace (trace_id, root_session_id, session_id, agent_name, feature_id)
		VALUES
			('t1', 'root-1', 'sess-a', 'htmlgraph:researcher', 'feat-111'),
			('t2', 'root-1', 'sess-b', 'htmlgraph:sonnet-coder', ''),
			('t3', 'root-2', 'sess-c', 'htmlgraph:researcher', 'feat-222')`)
	if err != nil {
		t.Fatalf("seed lineage: %v", err)
	}

	edges := loadAgentEdges(db)

	workedOn := make(map[string][]string) // agent -> []feature
	for _, e := range edges {
		if e.Type == "worked_on" {
			workedOn[e.Source] = append(workedOn[e.Source], e.Target)
		}
	}

	// htmlgraph:researcher should have worked_on edges for feat-111 and feat-222.
	researcherFeatures := workedOn["htmlgraph:researcher"]
	if len(researcherFeatures) != 2 {
		t.Errorf("htmlgraph:researcher worked_on: want 2 features, got %d: %v", len(researcherFeatures), researcherFeatures)
	}

	// htmlgraph:sonnet-coder has no feature_id, so no worked_on edge.
	if len(workedOn["htmlgraph:sonnet-coder"]) != 0 {
		t.Errorf("htmlgraph:sonnet-coder worked_on: want 0, got %v", workedOn["htmlgraph:sonnet-coder"])
	}
}
