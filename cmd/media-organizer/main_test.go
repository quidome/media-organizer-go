package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestOrganizeCommand_PrintsSourceAndDestination(t *testing.T) {
	cmd := newRootCmd()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"organize", "src", "dst"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Source: src") {
		t.Fatalf("expected source line, got %q", output)
	}
	if !strings.Contains(output, "Destination: dst") {
		t.Fatalf("expected destination line, got %q", output)
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
