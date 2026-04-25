//go:build !linux

package db

// isUnsafeForMmap returns false on non-Linux platforms. Native filesystems on
// macOS (APFS, HFS+) and Windows (NTFS) support mmap correctly, so WAL mode
// is safe to use.
func isUnsafeForMmap(_ string) bool {
	return false
}
