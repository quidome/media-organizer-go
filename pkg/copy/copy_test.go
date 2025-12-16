package copy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/quidome/media-organizer-go/pkg/plan"
)

func TestExecute_CopiesFileAndCreatesDirs(t *testing.T) {
	tmpSrc := t.TempDir()
	tmpDst := t.TempDir()

	srcPath := filepath.Join(tmpSrc, "test.jpg")
	content := []byte("test content")
	if err := os.WriteFile(srcPath, content, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	destPath := filepath.Join(tmpDst, "2023", "11", "15", "test.jpg")
	ops := []plan.Operation{{SourcePath: srcPath, DestinationPath: destPath}}

	results, err := Execute(ops, Options{Overwrite: false})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Fatalf("expected success, got %v", results[0].Error)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("content mismatch: got %q want %q", got, content)
	}
}

func TestExecute_DoesNotOverwrite(t *testing.T) {
	tmpSrc := t.TempDir()
	tmpDst := t.TempDir()

	srcPath := filepath.Join(tmpSrc, "test.jpg")
	if err := os.WriteFile(srcPath, []byte("new"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	destPath := filepath.Join(tmpDst, "test.jpg")
	if err := os.WriteFile(destPath, []byte("old"), 0o644); err != nil {
		t.Fatalf("write destination: %v", err)
	}

	op := plan.Operation{SourcePath: srcPath, DestinationPath: destPath}
	results, err := Execute([]plan.Operation{op}, Options{Overwrite: false})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if results[0].Success {
		t.Fatalf("expected failure when destination exists")
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != "old" {
		t.Fatalf("destination was overwritten: %q", got)
	}
}

func TestExecute_OverwriteWhenEnabled(t *testing.T) {
	tmpSrc := t.TempDir()
	tmpDst := t.TempDir()

	srcPath := filepath.Join(tmpSrc, "test.jpg")
	if err := os.WriteFile(srcPath, []byte("new"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	destPath := filepath.Join(tmpDst, "test.jpg")
	if err := os.WriteFile(destPath, []byte("old"), 0o644); err != nil {
		t.Fatalf("write destination: %v", err)
	}

	op := plan.Operation{SourcePath: srcPath, DestinationPath: destPath}
	results, err := Execute([]plan.Operation{op}, Options{Overwrite: true})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !results[0].Success {
		t.Fatalf("expected success, got %v", results[0].Error)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != "new" {
		t.Fatalf("expected overwritten content, got %q", got)
	}
}

func TestExecute_MultipleOperations(t *testing.T) {
	tmpSrc := t.TempDir()
	tmpDst := t.TempDir()

	s1 := filepath.Join(tmpSrc, "a.jpg")
	s2 := filepath.Join(tmpSrc, "b.jpg")
	if err := os.WriteFile(s1, []byte("a"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(s2, []byte("b"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	ops := []plan.Operation{
		{SourcePath: s1, DestinationPath: filepath.Join(tmpDst, "2023", "11", "15", "a.jpg")},
		{SourcePath: s2, DestinationPath: filepath.Join(tmpDst, "2023", "11", "16", "b.jpg")},
	}

	results, err := Execute(ops, Options{Overwrite: false})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, r := range results {
		if !r.Success {
			t.Fatalf("result %d failed: %v", i, r.Error)
		}
	}
}
