package createdat

import (
	"bytes"
	"io"
	"testing"
	"testing/fstest"
	"time"
)

func TestVideoExtractor_MP4CreationTime(t *testing.T) {
	b, err := testdataFS.ReadFile("testdata/f1.mp4")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	fsys := fstest.MapFS{
		"a.mp4": &fstest.MapFile{Data: b, ModTime: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	res, err := Determine(fsys, "a.mp4", Options{Location: time.UTC})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != SourceMetadata {
		t.Fatalf("expected metadata source, got %q", res.Source)
	}

	want := time.Date(2024, 1, 15, 9, 30, 45, 0, time.UTC)
	if !res.CreatedAt.Equal(want) {
		t.Fatalf("unexpected CreatedAt\n got: %v\nwant: %v", res.CreatedAt, want)
	}
}

func TestVideoExtractor_MOVCreationTime(t *testing.T) {
	b, err := testdataFS.ReadFile("testdata/f1.mov")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	fsys := fstest.MapFS{
		"a.mov": &fstest.MapFile{Data: b, ModTime: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	res, err := Determine(fsys, "a.mov", Options{Location: time.UTC})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != SourceMetadata {
		t.Fatalf("expected metadata source, got %q", res.Source)
	}

	want := time.Date(2024, 6, 20, 12, 22, 10, 0, time.UTC)
	if !res.CreatedAt.Equal(want) {
		t.Fatalf("unexpected CreatedAt\n got: %v\nwant: %v", res.CreatedAt, want)
	}
}

func TestVideoExtractor_NonVideoDataIsNotFound(t *testing.T) {
	tm, ok, err := (videoExtractor{}).CreatedAt("a.mp4", bytes.NewReader([]byte("not a mp4")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false")
	}
	if !tm.IsZero() {
		t.Fatalf("expected zero time")
	}
}

func TestVideoExtractor_InvalidDataIsNotFound(t *testing.T) {
	tm, ok, err := (videoExtractor{}).CreatedAt("a.mp4", bytes.NewReader([]byte{0x00, 0x00, 0x00, 0x00, 0x66, 0x74, 0x79, 0x70}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false for invalid mp4 data")
	}
	if !tm.IsZero() {
		t.Fatalf("expected zero time")
	}
}

func TestCompositeExtractor_RoutesToVideoExtractor(t *testing.T) {
	extractor := &compositeExtractor{
		imageExtractor: &testImageExtractor{},
		videoExtractor: videoExtractor{},
	}

	tm, ok, err := extractor.CreatedAt("a.mp4", bytes.NewReader([]byte("not a valid mp4")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false for invalid video, got: %v", tm)
	}
}

func TestCompositeExtractor_RoutesToImageExtractor(t *testing.T) {
	extractor := &compositeExtractor{
		imageExtractor: &testImageExtractor{
			tm:    time.Date(2023, 5, 10, 12, 0, 0, 0, time.UTC),
			found: true,
		},
		videoExtractor: videoExtractor{},
	}

	tm, ok, err := extractor.CreatedAt("a.jpg", bytes.NewReader([]byte("not a jpeg")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true for image file")
	}
	want := time.Date(2023, 5, 10, 12, 0, 0, 0, time.UTC)
	if !tm.Equal(want) {
		t.Fatalf("unexpected CreatedAt\n got: %v\nwant: %v", tm, want)
	}
}

func TestCompositeExtractor_FallsBackToFilenameForVideo(t *testing.T) {
	fsys := fstest.MapFS{
		"VID_20240520_142210.mp4": &fstest.MapFile{Data: []byte("not a mp4"), ModTime: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	res, err := Determine(fsys, "VID_20240520_142210.mp4", Options{Location: time.UTC})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != SourceFilename {
		t.Fatalf("expected filename source when video metadata extraction fails, got %q", res.Source)
	}

	want := time.Date(2024, 5, 20, 14, 22, 10, 0, time.UTC)
	if !res.CreatedAt.Equal(want) {
		t.Fatalf("unexpected CreatedAt\n got: %v\nwant: %v", res.CreatedAt, want)
	}
}

func TestCompositeExtractor_ImageWithMetadataBeatsFilename(t *testing.T) {
	fsys := fstest.MapFS{
		"IMG_20240520_142210.jpg": &fstest.MapFile{Data: []byte("not a jpeg"), ModTime: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	res, err := Determine(fsys, "IMG_20240520_142210.jpg", Options{Location: time.UTC})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != SourceFilename {
		t.Fatalf("expected filename source for image without metadata, got %q", res.Source)
	}

	want := time.Date(2024, 5, 20, 14, 22, 10, 0, time.UTC)
	if !res.CreatedAt.Equal(want) {
		t.Fatalf("unexpected CreatedAt\n got: %v\nwant: %v", res.CreatedAt, want)
	}
}

func TestIsVideoExtension(t *testing.T) {
	tests := []struct {
		ext    string
		expect bool
	}{
		{".mp4", true},
		{".MP4", true},
		{".mov", true},
		{".MOV", true},
		{".m4v", true},
		{".mkv", true},
		{".avi", true},
		{".webm", true},
		{".mts", true},
		{".3gp", true},
		{".jpg", false},
		{".png", false},
		{".txt", false},
		{".mp3", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			if got := isVideoExtension(tt.ext); got != tt.expect {
				t.Errorf("isVideoExtension(%q) = %v, want %v", tt.ext, got, tt.expect)
			}
		})
	}
}

type testImageExtractor struct {
	tm    time.Time
	found bool
	err   error
}

func (e *testImageExtractor) CreatedAt(path string, r io.Reader) (time.Time, bool, error) {
	return e.tm, e.found, e.err
}
