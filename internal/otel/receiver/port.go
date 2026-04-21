package receiver

import (
	"hash/fnv"
	"path/filepath"
)

// PortForProject returns the deterministic OTLP HTTP port for a project
// directory. Range: 4318..5317 (1000 slots). Collisions are rare; the
// caller should handle a bind-failure gracefully by probing nearby.
func PortForProject(projectDir string) int {
	if projectDir == "" {
		return 4318
	}
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		abs = projectDir
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(abs))
	return 4318 + int(h.Sum32()%1000)
}
