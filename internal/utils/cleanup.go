package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CleanupResult reports the outcome of a deep cleanup operation.
type CleanupResult struct {
	DeletedPaths []string
	FreedBytes   int64
}

// DeepCleanup performs a two-phase cleanup of auth state, browser profiles,
// and temporary files. In preview mode (dryRun=true), it only reports what
// would be deleted. In delete mode, it removes the files and reports results.
//
// If preserveLibrary is true, library.json files are never deleted.
func DeepCleanup(dataDir string, preserveLibrary bool) (*CleanupResult, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("dataDir is required")
	}

	// Phase 1: preview — collect all paths that would be deleted
	paths, err := collectCleanupPaths(dataDir, preserveLibrary)
	if err != nil {
		return nil, fmt.Errorf("collect cleanup paths: %w", err)
	}

	// Phase 2: delete
	result := &CleanupResult{}
	for _, p := range paths {
		info, statErr := os.Stat(p)
		if statErr != nil {
			continue // already gone
		}
		if info.IsDir() {
			if err := os.RemoveAll(p); err != nil {
				return result, fmt.Errorf("remove dir %s: %w", p, err)
			}
			result.DeletedPaths = append(result.DeletedPaths, p)
		} else {
			size := info.Size()
			if err := os.Remove(p); err != nil {
				return result, fmt.Errorf("remove file %s: %w", p, err)
			}
			result.DeletedPaths = append(result.DeletedPaths, p)
			result.FreedBytes += size
		}
	}

	return result, nil
}

// collectCleanupPaths returns all paths that should be cleaned up.
func collectCleanupPaths(dataDir string, preserveLibrary bool) ([]string, error) {
	var paths []string

	// Walk the data directory and collect deletable paths
	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors (e.g., permission denied)
		}

		// Skip the root dataDir itself
		if path == dataDir {
			return nil
		}

		// Preserve library.json if requested
		if preserveLibrary && strings.HasSuffix(path, "library.json") {
			return nil
		}

		// Collect: auth state files, chrome profiles, instance profiles, tmp files
		if isCleanupTarget(path, info) {
			paths = append(paths, path)
			// If it's a directory, skip its contents (RemoveAll handles it)
			if info.IsDir() {
				return filepath.SkipDir
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort: directories first (deepest first for RemoveAll), then files
	// Simple approach: reverse the walk order (deepest first)
	for i, j := 0, len(paths)-1; i < j; i, j = i+1, j-1 {
		paths[i], paths[j] = paths[j], paths[i]
	}

	return paths, nil
}

// isCleanupTarget returns true if the path should be cleaned up.
func isCleanupTarget(path string, info os.FileInfo) bool {
	name := info.Name()
	base := filepath.Base(filepath.Dir(path))

	// Auth state files
	if name == "state.json" || name == "session.json" {
		return true
	}

	// Chrome profile directories
	if strings.HasPrefix(name, "chrome_profile") {
		return true
	}

	// Instance profile directories
	if base == "chrome_profile_instances" || strings.HasPrefix(name, "isolated-") {
		return true
	}

	// Browser state directory
	if base == "browser_state" {
		return true
	}

	// Lock files
	if name == ".browser.lock" {
		return true
	}

	// Temp files
	if strings.HasSuffix(name, ".tmp") || strings.HasPrefix(name, ".tmp") {
		return true
	}

	return false
}
