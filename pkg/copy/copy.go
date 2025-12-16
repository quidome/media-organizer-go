package copy

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/quidome/media-organizer-go/pkg/plan"
)

var (
	// ErrDestinationExists is returned when attempting to copy to an existing file
	ErrDestinationExists = errors.New("destination file already exists")
)

// Result contains the outcome of a copy operation.
type Result struct {
	Operation plan.Operation
	Success   bool
	Error     error
}

// Options configures the copy behavior.
type Options struct {
	// Overwrite allows overwriting existing files.
	// Default should be false for safety.
	Overwrite bool
}

// Execute performs copy operations for the given plans.
//
// It will:
// - Create destination directories if they don't exist
// - Never overwrite existing files (unless Overwrite is true)
// - Copy files preserving content
func Execute(operations []plan.Operation, opts Options) ([]Result, error) {
	results := make([]Result, 0, len(operations))

	for _, op := range operations {
		result := Result{Operation: op, Success: false}

		// Create destination directory
		destDir := filepath.Dir(op.DestinationPath)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			result.Error = fmt.Errorf("create directory: %w", err)
			results = append(results, result)
			continue
		}

		// Copy the file (destination path is assumed finalized by planning/reconcile stages).
		if err := copyFile(op.SourcePath, op.DestinationPath, opts.Overwrite); err != nil {
			result.Error = fmt.Errorf("copy file: %w", err)
			results = append(results, result)
			continue
		}

		result.Success = true
		results = append(results, result)
	}

	return results, nil
}

// copyFile copies a single file from src to dst.
// If allowOverwrite is true, existing files will be overwritten.
func copyFile(src, dst string, allowOverwrite bool) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	// Create destination file
	flags := os.O_WRONLY | os.O_CREATE
	if !allowOverwrite {
		flags |= os.O_EXCL
	} else {
		flags |= os.O_TRUNC
	}

	dstFile, err := os.OpenFile(dst, flags, srcInfo.Mode())
	if err != nil {
		if os.IsExist(err) {
			return ErrDestinationExists
		}
		return fmt.Errorf("create destination: %w", err)
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		// Try to clean up partial file on error (only if we created it)
		if !allowOverwrite {
			_ = os.Remove(dst)
		}
		return fmt.Errorf("copy content: %w", err)
	}

	// Ensure data is written to disk
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	return nil
}
