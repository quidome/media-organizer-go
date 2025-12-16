package reconcile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quidome/media-organizer-go/pkg/createdat"
)

func TestDedupeSources_ChoosesOldest(t *testing.T) {
	tmp := t.TempDir()
	p1 := filepath.Join(tmp, "a.jpg")
	p2 := filepath.Join(tmp, "b.jpg")

	content := []byte("same")
	if err := os.WriteFile(p1, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p2, content, 0o644); err != nil {
		t.Fatal(err)
	}

	details := map[string]createdat.DetailedResult{
		p1: {Best: createdat.Result{CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}},
		p2: {Best: createdat.Result{CreatedAt: time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)}},
	}

	sizes := map[string]int64{p1: int64(len(content)), p2: int64(len(content))}

	kept, decisions, err := DedupeSources([]string{p1, p2}, details, sizes)
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 1 || kept[0] != p2 {
		t.Fatalf("expected to keep oldest %s, got %v", p2, kept)
	}

	sawSkip := false
	for _, d := range decisions {
		if d.SourcePath == p1 && d.Action == ActionSkippedDuplicateSrc {
			sawSkip = true
		}
	}
	if !sawSkip {
		t.Fatalf("expected %s to be skipped duplicate", p1)
	}
}
