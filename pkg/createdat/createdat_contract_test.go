//go:build createdat_contract

package createdat_test

import (
	"errors"
	"io"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/quidome/media-organizer-go/pkg/createdat"
)

// This file defines the contract for the created-at attribution stage (PIPELINE.md Stage 2).
//
// It is guarded by the build tag "createdat_contract" so the repository can land test cases
// before the implementation exists.
//
// Run with:
//   go test ./... -tags createdat_contract

func TestDetermine_Priorities_MetadataThenFilenameThenMtime(t *testing.T) {
	loc := time.FixedZone("TEST", 2*60*60)

	metadataTime := time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)
	mtime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name          string
		path          string
		modTime       time.Time
		metadataTime  time.Time
		metadataFound bool
		metadataErr   error
		wantTime      time.Time
		wantSource    createdat.Source
	}{
		{
			name:          "metadata beats filename and mtime",
			path:          "root/IMG_20240102_030405.jpg",
			modTime:       mtime,
			metadataTime:  metadataTime,
			metadataFound: true,
			wantTime:      metadataTime,
			wantSource:    createdat.SourceMetadata,
		},
		{
			name:          "filename used when metadata missing",
			path:          "root/IMG_20240102_030405.jpg",
			modTime:       mtime,
			metadataFound: false,
			wantTime:      time.Date(2024, 1, 2, 3, 4, 5, 0, loc),
			wantSource:    createdat.SourceFilename,
		},
		{
			name:          "metadata error falls back to filename",
			path:          "root/IMG_20240102_030405.jpg",
			modTime:       mtime,
			metadataFound: false,
			metadataErr:   errors.New("boom"),
			wantTime:      time.Date(2024, 1, 2, 3, 4, 5, 0, loc),
			wantSource:    createdat.SourceFilename,
		},
		{
			name:          "mtime used when filename has no date",
			path:          "root/holiday.jpg",
			modTime:       mtime,
			metadataFound: false,
			wantTime:      mtime,
			wantSource:    createdat.SourceMtime,
		},
		{
			name:          "unknown when no metadata, no filename, zero mtime",
			path:          "root/holiday.jpg",
			modTime:       time.Time{},
			metadataFound: false,
			wantTime:      time.Time{},
			wantSource:    createdat.SourceUnknown,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fsys := fstest.MapFS{
				tc.path: &fstest.MapFile{Data: []byte("x"), ModTime: tc.modTime},
			}

			metadata := &fakeMetadataExtractor{
				createdAt: tc.metadataTime,
				found:     tc.metadataFound,
				err:       tc.metadataErr,
			}

			res, err := createdat.Determine(fsys, tc.path, createdat.Options{Location: loc, Metadata: metadata})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !res.CreatedAt.Equal(tc.wantTime) {
				t.Fatalf("unexpected CreatedAt\n got: %v\nwant: %v", res.CreatedAt, tc.wantTime)
			}
			if res.Source != tc.wantSource {
				t.Fatalf("unexpected Source\n got: %q\nwant: %q", res.Source, tc.wantSource)
			}
		})
	}
}

func TestDetermine_FilenamePatterns(t *testing.T) {
	loc := time.FixedZone("TEST", -7*60*60)
	mtime := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name string
		path string
		want time.Time
	}{
		{
			name: "IMG_YYYYMMDD_HHMMSS",
			path: "root/IMG_20250102_030405.jpg",
			want: time.Date(2025, 1, 2, 3, 4, 5, 0, loc),
		},
		{
			name: "VID_YYYYMMDD_HHMMSS",
			path: "root/VID_20250102_030405.mp4",
			want: time.Date(2025, 1, 2, 3, 4, 5, 0, loc),
		},
		{
			name: "PXL_YYYYMMDD_HHMMSSfff",
			path: "root/PXL_20250102_030405123.jpg",
			want: time.Date(2025, 1, 2, 3, 4, 5, 0, loc),
		},
		{
			name: "YYYY-MM-DD HH.MM.SS",
			path: "root/2025-01-02 03.04.05.jpg",
			want: time.Date(2025, 1, 2, 3, 4, 5, 0, loc),
		},
		{
			name: "IMG-YYYYMMDD-WA0001 date only",
			path: "root/IMG-20250102-WA0001.jpg",
			want: time.Date(2025, 1, 2, 0, 0, 0, 0, loc),
		},
		{
			name: "Screenshot_YYYY-MM-DD-HH-MM-SS",
			path: "root/Screenshot_2025-01-02-03-04-05.png",
			want: time.Date(2025, 1, 2, 3, 4, 5, 0, loc),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fsys := fstest.MapFS{
				tc.path: &fstest.MapFile{Data: []byte("x"), ModTime: mtime},
			}

			res, err := createdat.Determine(fsys, tc.path, createdat.Options{Location: loc})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Source != createdat.SourceFilename {
				t.Fatalf("expected filename source, got %q", res.Source)
			}
			if !res.CreatedAt.Equal(tc.want) {
				t.Fatalf("unexpected CreatedAt\n got: %v\nwant: %v", res.CreatedAt, tc.want)
			}
		})
	}
}

func TestDetermine_MissingFileReturnsError(t *testing.T) {
	fsys := fstest.MapFS{}

	_, err := createdat.Determine(fsys, "root/missing.jpg", createdat.Options{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected fs.ErrNotExist, got %v", err)
	}
}

func TestDetermine_DirectoryReturnsError(t *testing.T) {
	fsys := fstest.MapFS{
		"root": &fstest.MapFile{Mode: fs.ModeDir},
	}

	_, err := createdat.Determine(fsys, "root", createdat.Options{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

type fakeMetadataExtractor struct {
	createdAt time.Time
	found     bool
	err       error

	calls int
}

func (f *fakeMetadataExtractor) CreatedAt(path string, r io.Reader) (time.Time, bool, error) {
	f.calls++
	_, _ = io.ReadAll(r)
	return f.createdAt, f.found, f.err
}
