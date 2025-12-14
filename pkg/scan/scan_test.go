package scan

import (
	"reflect"
	"testing"
	"testing/fstest"
)

func TestScan_MaxDepth(t *testing.T) {
	fsys := fstest.MapFS{
		"root/a.jpg":            &fstest.MapFile{Data: []byte("a")},
		"root/b.MP4":            &fstest.MapFile{Data: []byte("b")},
		"root/c.txt":            &fstest.MapFile{Data: []byte("c")},
		"root/sub/d.png":        &fstest.MapFile{Data: []byte("d")},
		"root/sub/nested/e.mov": &fstest.MapFile{Data: []byte("e")},
	}

	testCases := []struct {
		name     string
		maxDepth int
		want     []string
	}{
		{
			name:     "depth 0 includes only top-level",
			maxDepth: 0,
			want:     []string{"a.jpg", "b.MP4"},
		},
		{
			name:     "depth 1 includes one subdirectory",
			maxDepth: 1,
			want:     []string{"a.jpg", "b.MP4", "sub/d.png"},
		},
		{
			name:     "depth 2 includes nested subdirectories",
			maxDepth: 2,
			want:     []string{"a.jpg", "b.MP4", "sub/d.png", "sub/nested/e.mov"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := DefaultOptions()
			opts.MaxDepth = tc.maxDepth

			got, err := Scan(fsys, "root", opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestScan_IgnoresNonMedia(t *testing.T) {
	fsys := fstest.MapFS{
		"root/a.txt": &fstest.MapFile{Data: []byte("a")},
		"root/b.xmp": &fstest.MapFile{Data: []byte("b")},
	}

	opts := DefaultOptions()
	got, err := Scan(fsys, "root", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("expected no media files, got %#v", got)
	}
}

func TestScan_InvalidMaxDepth(t *testing.T) {
	fsys := fstest.MapFS{}

	opts := DefaultOptions()
	opts.MaxDepth = -2

	_, err := Scan(fsys, "root", opts)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
