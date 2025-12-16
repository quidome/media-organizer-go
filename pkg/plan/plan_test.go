package plan

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDestination(t *testing.T) {
	destRoot := "/dest"
	createdAt := time.Date(2023, 11, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name          string
		filename      string
		existingFiles map[string]bool
		want          string
	}{
		{
			name:          "no collision",
			filename:      "photo.jpg",
			existingFiles: make(map[string]bool),
			want:          filepath.Join("/dest", "2023", "11", "15", "photo.jpg"),
		},
		{
			name:     "first collision gets _1",
			filename: "photo.jpg",
			existingFiles: map[string]bool{
				filepath.Join("/dest", "2023", "11", "15", "photo.jpg"): true,
			},
			want: filepath.Join("/dest", "2023", "11", "15", "photo_1.jpg"),
		},
		{
			name:     "second collision gets _2",
			filename: "photo.jpg",
			existingFiles: map[string]bool{
				filepath.Join("/dest", "2023", "11", "15", "photo.jpg"):   true,
				filepath.Join("/dest", "2023", "11", "15", "photo_1.jpg"): true,
			},
			want: filepath.Join("/dest", "2023", "11", "15", "photo_2.jpg"),
		},
		{
			name:     "multiple extensions handled correctly",
			filename: "archive.tar.gz",
			existingFiles: map[string]bool{
				filepath.Join("/dest", "2023", "11", "15", "archive.tar.gz"): true,
			},
			want: filepath.Join("/dest", "2023", "11", "15", "archive.tar_1.gz"),
		},
		{
			name:          "file without extension",
			filename:      "README",
			existingFiles: make(map[string]bool),
			want:          filepath.Join("/dest", "2023", "11", "15", "README"),
		},
		{
			name:     "file without extension with collision",
			filename: "README",
			existingFiles: map[string]bool{
				filepath.Join("/dest", "2023", "11", "15", "README"): true,
			},
			want: filepath.Join("/dest", "2023", "11", "15", "README_1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Destination(destRoot, tt.filename, createdAt, tt.existingFiles)
			if got != tt.want {
				t.Errorf("Destination() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDestination_DeterministicSuffixes(t *testing.T) {
	destRoot := "/dest"
	createdAt := time.Date(2023, 11, 15, 10, 30, 0, 0, time.UTC)
	existingFiles := make(map[string]bool)

	// Add same filename multiple times
	filenames := []string{
		"photo.jpg",
		"photo.jpg",
		"photo.jpg",
		"photo.jpg",
	}

	expected := []string{
		filepath.Join("/dest", "2023", "11", "15", "photo.jpg"),
		filepath.Join("/dest", "2023", "11", "15", "photo_1.jpg"),
		filepath.Join("/dest", "2023", "11", "15", "photo_2.jpg"),
		filepath.Join("/dest", "2023", "11", "15", "photo_3.jpg"),
	}

	for i, filename := range filenames {
		got := Destination(destRoot, filename, createdAt, existingFiles)
		if got != expected[i] {
			t.Errorf("iteration %d: Destination() = %v, want %v", i, got, expected[i])
		}
	}
}

func TestPlan(t *testing.T) {
	destRoot := "/dest"

	sources := []string{
		"photos/IMG_20231115_103000.jpg",
		"photos/IMG_20231115_103001.jpg",
		"photos/IMG_20231116_120000.jpg",
	}

	createdAtMap := map[string]time.Time{
		"photos/IMG_20231115_103000.jpg": time.Date(2023, 11, 15, 10, 30, 0, 0, time.UTC),
		"photos/IMG_20231115_103001.jpg": time.Date(2023, 11, 15, 10, 30, 1, 0, time.UTC),
		"photos/IMG_20231116_120000.jpg": time.Date(2023, 11, 16, 12, 0, 0, 0, time.UTC),
	}

	operations := Plan(destRoot, sources, createdAtMap)

	if len(operations) != 3 {
		t.Fatalf("Plan() returned %d operations, want 3", len(operations))
	}

	expected := []Operation{
		{
			SourcePath:      "photos/IMG_20231115_103000.jpg",
			DestinationPath: filepath.Join("/dest", "2023", "11", "15", "IMG_20231115_103000.jpg"),
		},
		{
			SourcePath:      "photos/IMG_20231115_103001.jpg",
			DestinationPath: filepath.Join("/dest", "2023", "11", "15", "IMG_20231115_103001.jpg"),
		},
		{
			SourcePath:      "photos/IMG_20231116_120000.jpg",
			DestinationPath: filepath.Join("/dest", "2023", "11", "16", "IMG_20231116_120000.jpg"),
		},
	}

	for i, op := range operations {
		if op.SourcePath != expected[i].SourcePath {
			t.Errorf("operation %d: SourcePath = %v, want %v", i, op.SourcePath, expected[i].SourcePath)
		}
		if op.DestinationPath != expected[i].DestinationPath {
			t.Errorf("operation %d: DestinationPath = %v, want %v", i, op.DestinationPath, expected[i].DestinationPath)
		}
	}
}

func TestPlan_WithCollisions(t *testing.T) {
	destRoot := "/dest"

	// Same filename, same date - should get collision suffixes
	sources := []string{
		"photos/photo.jpg",
		"backup/photo.jpg",
		"archive/photo.jpg",
	}

	createdAt := time.Date(2023, 11, 15, 10, 30, 0, 0, time.UTC)
	createdAtMap := map[string]time.Time{
		"photos/photo.jpg":  createdAt,
		"backup/photo.jpg":  createdAt,
		"archive/photo.jpg": createdAt,
	}

	operations := Plan(destRoot, sources, createdAtMap)

	if len(operations) != 3 {
		t.Fatalf("Plan() returned %d operations, want 3", len(operations))
	}

	expected := []string{
		filepath.Join("/dest", "2023", "11", "15", "photo.jpg"),
		filepath.Join("/dest", "2023", "11", "15", "photo_1.jpg"),
		filepath.Join("/dest", "2023", "11", "15", "photo_2.jpg"),
	}

	for i, op := range operations {
		if op.DestinationPath != expected[i] {
			t.Errorf("operation %d: DestinationPath = %v, want %v", i, op.DestinationPath, expected[i])
		}
	}
}

func TestPlan_SkipsMissingCreatedAt(t *testing.T) {
	destRoot := "/dest"

	sources := []string{
		"photos/photo1.jpg",
		"photos/photo2.jpg",
	}

	// Only one file has a creation date
	createdAtMap := map[string]time.Time{
		"photos/photo1.jpg": time.Date(2023, 11, 15, 10, 30, 0, 0, time.UTC),
	}

	operations := Plan(destRoot, sources, createdAtMap)

	if len(operations) != 1 {
		t.Fatalf("Plan() returned %d operations, want 1", len(operations))
	}

	if operations[0].SourcePath != "photos/photo1.jpg" {
		t.Errorf("SourcePath = %v, want photos/photo1.jpg", operations[0].SourcePath)
	}
}
