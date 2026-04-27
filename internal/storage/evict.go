package storage

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// DefaultMaxAge / DefaultMaxSize are the policy used by opportunistic prune
// and as the default flag values for `htmlgraph cache prune`. They match
// the spike recommendation in spk-dfb051a3 (90-day age, 1 GiB cap).
const (
	DefaultMaxAge  = 90 * 24 * time.Hour
	DefaultMaxSize = int64(1) << 30
)

// EvictResult summarises what Evict removed.
type EvictResult struct {
	Removed        []string // absolute paths of removed project-cache dirs
	BytesFreed     int64
	RemainingBytes int64
	RemainingDirs  int
	DryRun         bool
}

// CacheEntry describes one project's cache directory for stats reporting.
type CacheEntry struct {
	Hash    string
	Path    string
	Size    int64
	ModTime time.Time
}

// CacheRoot returns the directory that holds per-project cache subdirs:
// <UserCacheDir>/htmlgraph. The HTMLGRAPH_DB_PATH override is intentionally
// ignored — it points at a single DB file, not a project-keyed cache root.
func CacheRoot() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate user cache dir: %w", err)
	}
	return filepath.Join(cache, "htmlgraph"), nil
}

// CacheStats lists every project-cache subdir under cacheRoot, newest first.
// A missing cacheRoot returns an empty slice without error.
func CacheStats(cacheRoot string) ([]CacheEntry, error) {
	entries, err := readProjectEntries(cacheRoot)
	if err != nil {
		return nil, err
	}
	out := make([]CacheEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, CacheEntry{
			Hash:    filepath.Base(e.path),
			Path:    e.path,
			Size:    e.size,
			ModTime: e.mtime,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModTime.After(out[j].ModTime) })
	return out, nil
}

// Evict reclaims disk in two passes:
//  1. age — remove every project-cache dir whose newest mtime is older than maxAge.
//  2. size — if the surviving total still exceeds maxSize, remove the
//     least-recently-used dirs until it fits.
//
// dryRun reports candidates without touching the disk. A missing cacheRoot
// is not an error — Evict returns a zero result. Non-hex entries (the
// .last-prune marker, stray files) are ignored.
func Evict(cacheRoot string, maxAge time.Duration, maxSize int64, dryRun bool) (EvictResult, error) {
	res := EvictResult{DryRun: dryRun}
	entries, err := readProjectEntries(cacheRoot)
	if err != nil {
		return res, err
	}

	now := time.Now()
	keep := make([]projectEntry, 0, len(entries))
	for _, e := range entries {
		if maxAge > 0 && now.Sub(e.mtime) > maxAge {
			if !dryRun {
				if rmErr := os.RemoveAll(e.path); rmErr != nil {
					return res, fmt.Errorf("remove %s: %w", e.path, rmErr)
				}
			}
			res.Removed = append(res.Removed, e.path)
			res.BytesFreed += e.size
			continue
		}
		keep = append(keep, e)
	}

	if maxSize > 0 {
		sort.Slice(keep, func(i, j int) bool { return keep[i].mtime.Before(keep[j].mtime) })
		var total int64
		for _, e := range keep {
			total += e.size
		}
		for total > maxSize && len(keep) > 0 {
			victim := keep[0]
			if !dryRun {
				if rmErr := os.RemoveAll(victim.path); rmErr != nil {
					return res, fmt.Errorf("remove %s: %w", victim.path, rmErr)
				}
			}
			res.Removed = append(res.Removed, victim.path)
			res.BytesFreed += victim.size
			total -= victim.size
			keep = keep[1:]
		}
	}

	for _, e := range keep {
		res.RemainingBytes += e.size
	}
	res.RemainingDirs = len(keep)
	return res, nil
}

// MaybePruneOpportunistic evicts the cache iff the .last-prune marker is
// older than minInterval. On first call (no marker) it creates the marker
// without pruning, so a brand-new install is never surprised by deletions.
// The bool return reports whether a prune ran. Errors are returned but
// callers may ignore them — opportunistic prune is best-effort.
func MaybePruneOpportunistic(cacheRoot string, minInterval, maxAge time.Duration, maxSize int64) (EvictResult, bool, error) {
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return EvictResult{}, false, fmt.Errorf("ensure cache root: %w", err)
	}
	marker := filepath.Join(cacheRoot, ".last-prune")
	info, err := os.Stat(marker)
	if errors.Is(err, fs.ErrNotExist) {
		return EvictResult{}, false, touchMarker(marker)
	}
	if err != nil {
		return EvictResult{}, false, err
	}
	if time.Since(info.ModTime()) < minInterval {
		return EvictResult{}, false, nil
	}
	res, evictErr := Evict(cacheRoot, maxAge, maxSize, false)
	if evictErr != nil {
		return res, true, evictErr
	}
	return res, true, touchMarker(marker)
}

// OpportunisticPrune is the default-policy wrapper invoked from the CLI's
// PersistentPreRunE: 7-day prune cadence, 90-day max age, 1 GiB max size.
// Errors are swallowed — eviction is best-effort and must not block any
// command. When a prune actually removes something, an advisory line is
// written to w (callers typically pass os.Stderr).
func OpportunisticPrune(cacheRoot string, w io.Writer) {
	res, ran, err := MaybePruneOpportunistic(cacheRoot, 7*24*time.Hour, DefaultMaxAge, DefaultMaxSize)
	if err != nil || !ran || len(res.Removed) == 0 || w == nil {
		return
	}
	fmt.Fprintf(w, "[htmlgraph] cache: pruned %d stale cache dir(s), freed %d bytes\n",
		len(res.Removed), res.BytesFreed)
}

type projectEntry struct {
	path  string
	size  int64
	mtime time.Time
}

func readProjectEntries(cacheRoot string) ([]projectEntry, error) {
	dirEntries, err := os.ReadDir(cacheRoot)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", cacheRoot, err)
	}
	out := make([]projectEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		if !de.IsDir() {
			continue
		}
		name := de.Name()
		if !isHexHash(name) {
			continue
		}
		p := filepath.Join(cacheRoot, name)
		size, mtime, ferr := walkSize(p)
		if ferr != nil {
			// Skip dirs we can't read; don't fail the whole sweep.
			continue
		}
		out = append(out, projectEntry{path: p, size: size, mtime: mtime})
	}
	return out, nil
}

func walkSize(dir string) (int64, time.Time, error) {
	var size int64
	var mt time.Time
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		size += info.Size()
		if info.ModTime().After(mt) {
			mt = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return 0, time.Time{}, err
	}
	if mt.IsZero() {
		info, ierr := os.Stat(dir)
		if ierr != nil {
			return 0, time.Time{}, ierr
		}
		mt = info.ModTime()
	}
	return size, mt, nil
}

func isHexHash(s string) bool {
	if len(s) != 16 {
		return false
	}
	for _, ch := range s {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			return false
		}
	}
	return true
}

func touchMarker(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, werr := f.WriteString(time.Now().UTC().Format(time.RFC3339)); werr != nil {
		f.Close()
		return werr
	}
	return f.Close()
}
