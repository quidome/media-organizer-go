package createdat

import (
	"bytes"
	"testing"
	"testing/fstest"
	"time"
)

func TestDefaultExifExtractor_ExtractsDateTimeOriginal(t *testing.T) {
	b, err := testdataFS.ReadFile("testdata/f1-exif.jpg")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	fsys := fstest.MapFS{
		"a.jpg": &fstest.MapFile{Data: b, ModTime: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	res, err := Determine(fsys, "a.jpg", Options{Location: time.UTC})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != SourceMetadata {
		t.Fatalf("expected metadata source, got %q", res.Source)
	}

	// The fixture contains EXIF DateTimeOriginal = 2012-11-04 05:42:02 +0100.
	want := time.Date(2012, 11, 4, 5, 42, 2, 0, time.FixedZone("", 3600))
	if !res.CreatedAt.Equal(want) {
		t.Fatalf("unexpected CreatedAt\n got: %v\nwant: %v", res.CreatedAt, want)
	}
}

func TestExifExtractor_NonExifDataIsNotFound(t *testing.T) {
	tm, ok, err := (exifExtractor{}).CreatedAt("a.jpg", bytes.NewReader([]byte("not a jpeg")))
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
