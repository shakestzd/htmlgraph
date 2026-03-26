// Package sdk provides the public HtmlGraph SDK for Go consumers.
//
// It mirrors the Python SDK from htmlgraph.sdk, offering collections
// for features, bugs, spikes, tracks, and sessions with functional
// options for creation and a dual-write strategy (HTML canonical,
// SQLite read-index).
//
// Usage:
//
//	s, err := sdk.New("/path/to/.htmlgraph", "my-agent")
//	if err != nil { log.Fatal(err) }
//	defer s.Close()
//
//	feat, err := s.Features.Create("My Feature",
//	    sdk.FeatWithPriority("high"),
//	    sdk.FeatWithTrack("trk-abc"),
//	    sdk.FeatWithSteps("Step 1", "Step 2"),
//	)
package sdk

import (
	"database/sql"
	"fmt"
	"path/filepath"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
)

// SDK is the main entry point for interacting with an HtmlGraph project.
type SDK struct {
	// ProjectDir is the path to the .htmlgraph/ directory.
	ProjectDir string

	// Agent is the identifier of the agent using this SDK.
	Agent string

	// db is the optional SQLite database (read index).
	db *sql.DB

	// Collection accessors
	Features *FeatureCollection
	Bugs     *BugCollection
	Spikes   *SpikeCollection
	Tracks   *TrackCollection
	Sessions *SessionCollection
}

// New creates a new SDK instance, opens the SQLite database, and
// initialises all collection accessors.
//
// projectDir must point to a .htmlgraph/ directory.
// agent identifies the calling agent for work attribution.
func New(projectDir, agent string) (*SDK, error) {
	if projectDir == "" {
		return nil, fmt.Errorf("projectDir must not be empty")
	}
	if agent == "" {
		return nil, fmt.Errorf("agent must not be empty")
	}

	dbPath := filepath.Join(projectDir, "htmlgraph.db")
	database, err := dbpkg.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	s := &SDK{
		ProjectDir: projectDir,
		Agent:      agent,
		db:         database,
	}

	s.Features = NewFeatureCollection(s)
	s.Bugs = NewBugCollection(s)
	s.Spikes = NewSpikeCollection(s)
	s.Tracks = NewTrackCollection(s)
	s.Sessions = NewSessionCollection(s)

	return s, nil
}

// Close releases the SQLite database connection.
func (s *SDK) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// FeaturesDir returns the path to the features subdirectory.
func (s *SDK) FeaturesDir() string {
	return filepath.Join(s.ProjectDir, "features")
}

// BugsDir returns the path to the bugs subdirectory.
func (s *SDK) BugsDir() string {
	return filepath.Join(s.ProjectDir, "bugs")
}

// SpikesDir returns the path to the spikes subdirectory.
func (s *SDK) SpikesDir() string {
	return filepath.Join(s.ProjectDir, "spikes")
}

// TracksDir returns the path to the tracks subdirectory.
func (s *SDK) TracksDir() string {
	return filepath.Join(s.ProjectDir, "tracks")
}
