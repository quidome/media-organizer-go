package plan

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Operation represents a planned copy from source to destination.
type Operation struct {
	SourcePath      string
	DestinationPath string
}

// Destination computes the destination path for a file based on its creation date.
//
// The path follows the pattern: <destRoot>/YYYY/MM/DD/<filename>
// If a file with that name already exists in the existingFiles map,
// a suffix _N is appended before the extension, where N starts at 1.
func Destination(destRoot string, filename string, createdAt time.Time, existingFiles map[string]bool) string {
	year := fmt.Sprintf("%04d", createdAt.Year())
	month := fmt.Sprintf("%02d", createdAt.Month())
	day := fmt.Sprintf("%02d", createdAt.Day())

	dir := filepath.Join(destRoot, year, month, day)

	return resolveCollision(dir, filename, existingFiles)
}

// resolveCollision returns a unique destination path by appending _N before the extension if needed.
func resolveCollision(dir string, filename string, existingFiles map[string]bool) string {
	basePath := filepath.Join(dir, filename)

	if existingFiles == nil {
		existingFiles = make(map[string]bool)
	}

	// If no collision, return as-is
	if !existingFiles[basePath] {
		existingFiles[basePath] = true
		return basePath
	}

	// Split filename into name and extension
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	// Try suffixes starting from _1
	for i := 1; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s_%d%s", nameWithoutExt, i, ext))
		if !existingFiles[candidate] {
			existingFiles[candidate] = true
			return candidate
		}
	}
}

// Plan computes destination paths for a list of source files.
//
// Returns a slice of Operations with resolved collision handling.
func Plan(destRoot string, sources []string, createdAtMap map[string]time.Time) []Operation {
	existingFiles := make(map[string]bool)
	operations := make([]Operation, 0, len(sources))

	for _, src := range sources {
		createdAt, ok := createdAtMap[src]
		if !ok {
			// If no creation date, skip this file
			continue
		}

		filename := filepath.Base(src)
		dest := Destination(destRoot, filename, createdAt, existingFiles)

		operations = append(operations, Operation{
			SourcePath:      src,
			DestinationPath: dest,
		})
	}

	return operations
}
