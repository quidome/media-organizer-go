package main

import (
	"bytes"
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

	cmd := newRootCmd()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"organize", tmp, "dst"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := strings.TrimSpace(out.String())
	lines := strings.Split(output, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), output)
	}

	if !strings.Contains(lines[0], "IMG_20240102_030405.jpg\t") || !strings.Contains(lines[0], "\tfilename") {
		t.Fatalf("unexpected line: %q", lines[0])
	}
	if !strings.Contains(lines[1], "holiday.jpg\t") || !strings.Contains(lines[1], "\tmtime") {
		t.Fatalf("unexpected line: %q", lines[1])
	}
	if !strings.Contains(lines[2], "sub/VID_20240102_030405.mp4\t") || !strings.Contains(lines[2], "\tfilename") {
		t.Fatalf("unexpected line: %q", lines[2])
	}
}

func TestOrganizeCommand_ExecuteNotImplemented(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, tmp, "IMG_20240102_030405.jpg")

	cmd := newRootCmd()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"organize", tmp, "dst", "--execute"})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected error, got nil")
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
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func writeFileWithMTime(t *testing.T, dir string, relPath string, mtime time.Time) {
	t.Helper()

	path := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
}
