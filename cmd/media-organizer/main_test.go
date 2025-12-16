package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRootCommand_PrintsVersion(t *testing.T) {
	cmd := newRootCmd()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Media Organizer CLI") {
		t.Fatalf("expected output to include CLI header, got %q", output)
	}
	if !strings.Contains(output, "Version: "+version) {
		t.Fatalf("expected output to include version, got %q", output)
	}
}

func TestRootCommand_VerboseFlag(t *testing.T) {
	cmd := newRootCmd()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--verbose"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Verbose mode: enabled") {
		t.Fatalf("expected verbose line, got %q", output)
	}
}

func TestOrganizeCommand_RequiresTwoArgs(t *testing.T) {
	cmd := newRootCmd()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"organize", "only-source"})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestOrganizeCommand_DryRunPrintsCreatedAtRecords(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "IMG_20240102_030405.jpg")
	writeFileWithMTime(t, tmp, "holiday.jpg", time.Date(2020, 6, 7, 8, 9, 10, 0, time.UTC))
	writeFile(t, tmp, "sub/VID_20240102_030405.mp4")
	writeFile(t, tmp, "ignore.txt")

	dest := filepath.Join(tmp, "dst")

	cmd := newRootCmd()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"organize", tmp, dest})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := strings.TrimSpace(out.String())
	lines := strings.Split(output, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), output)
	}

	// Output format is now: <source> -> <destination>
	if !strings.Contains(lines[0], "IMG_20240102_030405.jpg -> "+dest) || !strings.Contains(lines[0], filepath.Join(dest, "2024", "01", "02", "IMG_20240102_030405.jpg")) {
		t.Fatalf("unexpected line: %q", lines[0])
	}
	if !strings.Contains(lines[1], "holiday.jpg -> "+dest) || !strings.Contains(lines[1], filepath.Join(dest, "2020", "06", "07", "holiday.jpg")) {
		t.Fatalf("unexpected line: %q", lines[1])
	}
	if !strings.Contains(lines[2], "VID_20240102_030405.mp4 -> "+dest) || !strings.Contains(lines[2], filepath.Join(dest, "2024", "01", "02", "VID_20240102_030405.mp4")) {
		t.Fatalf("unexpected line: %q", lines[2])
	}
}

func TestOrganizeCommand_Execute(t *testing.T) {
	tmpSrc := t.TempDir()
	tmpDst := t.TempDir()

	writeFile(t, tmpSrc, "IMG_20240102_030405.jpg")

	cmd := newRootCmd()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"organize", tmpSrc, tmpDst, "--execute"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify file was copied
	destPath := filepath.Join(tmpDst, "2024", "01", "02", "IMG_20240102_030405.jpg")
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("file was not copied to expected destination: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "copied") {
		t.Errorf("expected 'copied' in output, got: %s", output)
	}
}

func TestOrganizeCommand_JSONOutput(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "IMG_20240102_030405.jpg")
	writeFileWithMTime(t, tmp, "vacation.jpg", time.Date(2020, 8, 15, 14, 30, 0, 0, time.UTC))

	cmd := newRootCmd()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	dest := filepath.Join(tmp, "dst")

	cmd.SetArgs([]string{"organize", tmp, dest, "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var operations []jsonOperation
	if err := json.Unmarshal(out.Bytes(), &operations); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(operations) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(operations))
	}

	// Check first operation (filename-based)
	if !strings.Contains(operations[0].SourcePath, "IMG_20240102_030405.jpg") {
		t.Errorf("expected source path to contain IMG_20240102_030405.jpg, got %s", operations[0].SourcePath)
	}
	if operations[0].CreatedAt.Filename == "" {
		t.Errorf("expected created_at.filename to be set")
	}
	if operations[0].CreatedAt.Filestat == "" {
		t.Errorf("expected created_at.filestat to be set")
	}
	if operations[0].FileSizeBytes <= 0 {
		t.Errorf("expected file_size_bytes to be > 0")
	}
	if !strings.Contains(operations[0].DestinationPath, filepath.Join(dest, "2024", "01", "02")) {
		t.Errorf("expected destination to contain 2024/01/02, got %s", operations[0].DestinationPath)
	}

	// Check second operation (mtime-based, filename doesn't match pattern)
	if !strings.Contains(operations[1].SourcePath, "vacation.jpg") {
		t.Errorf("expected source path to contain vacation.jpg, got %s", operations[1].SourcePath)
	}
	if operations[1].CreatedAt.Filename != "" {
		t.Errorf("expected created_at.filename to be empty for vacation.jpg")
	}
	if operations[1].CreatedAt.Filestat == "" {
		t.Errorf("expected created_at.filestat to be set")
	}
	if operations[1].FileSizeBytes <= 0 {
		t.Errorf("expected file_size_bytes to be > 0")
	}
	if !strings.Contains(operations[1].DestinationPath, filepath.Join(dest, "2020", "08", "15")) {
		t.Errorf("expected destination to contain 2020/08/15, got %s", operations[1].DestinationPath)
	}
}

func TestScanCommand_RequiresOneArg(t *testing.T) {
	cmd := newRootCmd()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"scan"})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScanCommand_JSONOutput(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "a.jpg")
	writeFile(t, tmp, "b.txt")

	cmd := newRootCmd()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"scan", tmp, "--max-depth", "0", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var records []struct {
		SourcePath string `json:"source_path"`
		CreatedAt  struct {
			Metadata string `json:"metadata,omitempty"`
			Filename string `json:"filename,omitempty"`
			Filestat string `json:"filestat,omitempty"`
		} `json:"created_at"`
		FileSizeBytes int64     `json:"file_size_bytes"`
		ModTime       time.Time `json:"mod_time"`
	}
	if err := json.Unmarshal(out.Bytes(), &records); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 media record, got %d", len(records))
	}
	if !strings.HasSuffix(records[0].SourcePath, "a.jpg") {
		t.Fatalf("expected source_path to end with a.jpg, got %s", records[0].SourcePath)
	}
	if records[0].FileSizeBytes <= 0 {
		t.Fatalf("expected file_size_bytes > 0")
	}
	if records[0].ModTime.IsZero() {
		t.Fatalf("expected mod_time to be set")
	}
	if records[0].CreatedAt.Filestat == "" {
		t.Fatalf("expected created_at.filestat to be set")
	}
}

func TestScanCommand_PrintsMediaFiles(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "a.jpg")
	writeFile(t, tmp, "b.txt")
	writeFile(t, tmp, "sub/c.mp4")

	cmd := newRootCmd()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"scan", tmp, "--max-depth", "0"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := out.String()
	if strings.TrimSpace(output) != "a.jpg" {
		t.Fatalf("expected only top-level media file, got %q", output)
	}
}

func writeFile(t *testing.T, dir string, relPath string) {
	t.Helper()

	path := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(relPath), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func writeFileWithMTime(t *testing.T, dir string, relPath string, mtime time.Time) {
	t.Helper()

	path := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(relPath), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
}
