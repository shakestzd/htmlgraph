// Package blame — areas aggregator.
//
// WalkAreas walks every tracked source file under root, runs blame.Query for
// each, and returns a grouped per-track inventory.  It reuses Query() from
// this package rather than re-implementing the SQL logic.
package blame

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// WalkOptions controls the WalkAreas traversal.
type WalkOptions struct {
	// ByFile reverses the grouping: instead of tracks→files, return files→tracks.
	ByFile bool
	// IncludeUntracked includes files with zero feature_files rows in Untracked.
	// Defaults to true when the zero value of WalkOptions is used.
	IncludeUntracked *bool
}

// includeUntracked returns the effective IncludeUntracked setting.
// When the pointer is nil the default is true.
func (o WalkOptions) includeUntracked() bool {
	if o.IncludeUntracked == nil {
		return true
	}
	return *o.IncludeUntracked
}

// FileEntry describes one file within a track group.
type FileEntry struct {
	Path     string `json:"path"`
	Features int    `json:"features"`
	Touches  int    `json:"touches"`
}

// TrackArea is a single track's file inventory.
type TrackArea struct {
	TrackID    string      `json:"track_id"`
	TrackTitle string      `json:"track_title"`
	Files      []FileEntry `json:"files"`
	// Aggregate counts
	FeatureCount int `json:"feature_count"`
	TouchCount   int `json:"touch_count"`
}

// FileArea is one file with its associated tracks (used in ByFile mode).
type FileArea struct {
	Path   string        `json:"path"`
	Tracks []TrackRollup `json:"tracks"`
}

// AreasResult holds the complete WalkAreas output.
type AreasResult struct {
	// ByTrack groups files per track (populated when WalkOptions.ByFile == false).
	ByTrack []TrackArea `json:"by_track,omitempty"`
	// ByFile lists each file with its tracks (populated when WalkOptions.ByFile == true).
	ByFile []FileArea `json:"by_file,omitempty"`
	// Untracked holds files with no feature attribution (when includeUntracked == true).
	Untracked []string `json:"untracked,omitempty"`
}

// skipDir reports whether a directory should be excluded from the walk.
func skipDir(name string) bool {
	switch name {
	case ".git", ".htmlgraph", "node_modules", "vendor", ".claude",
		"dist", "bin", "build", "out", "target":
		return true
	}
	// Hidden directories (e.g. .github, .vscode) are skipped.
	return strings.HasPrefix(name, ".")
}

// WalkAreas walks root, runs blame.Query for every source file, and groups the
// results as requested by opts.
func WalkAreas(ctx context.Context, database *sql.DB, root string, opts WalkOptions) (*AreasResult, error) {
	// trackMap accumulates per-track aggregates (ByTrack mode).
	trackMap := make(map[string]*TrackArea)
	// trackFeatures holds the set of distinct feature IDs touching each track,
	// so FeatureCount reflects unique features rather than the per-file rollup
	// summed across files (which double-counted features touching multiple files).
	trackFeatures := make(map[string]map[string]struct{})
	var byFile []FileArea
	var untracked []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// Failure to read root itself is fatal — we'd silently return an
			// empty result. Failures on individual child entries (a single
			// unreadable subdir or stale dirent) are skipped.
			if path == root {
				return walkErr
			}
			return nil
		}
		if d.IsDir() {
			if path != root && skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}

		// Convert to a root-relative path for DB matching.
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		// Normalise to forward slashes for cross-platform consistency.
		rel = filepath.ToSlash(rel)

		result, err := Query(ctx, database, rel, QueryOptions{})
		if err != nil {
			return fmt.Errorf("blame %s: %w", rel, err)
		}

		if len(result.Features) == 0 {
			if opts.includeUntracked() {
				untracked = append(untracked, rel)
			}
			return nil
		}

		if opts.ByFile {
			byFile = append(byFile, FileArea{
				Path:   rel,
				Tracks: result.Tracks,
			})
			return nil
		}

		// ByTrack grouping: fan out to each track that touched this file.
		for _, tr := range result.Tracks {
			ta, ok := trackMap[tr.ID]
			if !ok {
				ta = &TrackArea{
					TrackID:    tr.ID,
					TrackTitle: tr.Title,
				}
				trackMap[tr.ID] = ta
				trackFeatures[tr.ID] = make(map[string]struct{})
			}
			ta.Files = append(ta.Files, FileEntry{
				Path:     rel,
				Features: tr.FeatureCount,
				Touches:  tr.TouchCount,
			})
			ta.TouchCount += tr.TouchCount
		}
		// Record distinct feature IDs per track from this file's features.
		for _, fr := range result.Features {
			if fr.TrackID == "" {
				continue
			}
			if set, ok := trackFeatures[fr.TrackID]; ok {
				set[fr.ID] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Resolve distinct-feature counts per track now that the walk is complete.
	for id, ta := range trackMap {
		ta.FeatureCount = len(trackFeatures[id])
	}

	res := &AreasResult{}

	if opts.ByFile {
		// Sort by path for deterministic output.
		sort.Slice(byFile, func(i, j int) bool { return byFile[i].Path < byFile[j].Path })
		res.ByFile = byFile
	} else {
		// Flatten map, sort tracks by file count desc, then alphabetically.
		tracks := make([]TrackArea, 0, len(trackMap))
		for _, ta := range trackMap {
			// Sort files within track alphabetically.
			sort.Slice(ta.Files, func(i, j int) bool { return ta.Files[i].Path < ta.Files[j].Path })
			tracks = append(tracks, *ta)
		}
		sort.Slice(tracks, func(i, j int) bool {
			if len(tracks[i].Files) != len(tracks[j].Files) {
				return len(tracks[i].Files) > len(tracks[j].Files)
			}
			return tracks[i].TrackID < tracks[j].TrackID
		})
		res.ByTrack = tracks
	}

	if opts.includeUntracked() {
		sort.Strings(untracked)
		res.Untracked = untracked
	}

	return res, nil
}
