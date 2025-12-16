package scan

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Options struct {
	MaxDepth int

	PhotoExtensions []string
	VideoExtensions []string
}

func DefaultOptions() Options {
	return Options{
		MaxDepth: -1,
		PhotoExtensions: []string{
			".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".tif", ".tiff", ".bmp",
		},
		VideoExtensions: []string{
			".mp4", ".mov", ".m4v", ".mkv", ".avi", ".webm", ".mts", ".3gp",
		},
	}
}

type Record struct {
	Path          string    `json:"path"`
	FileSizeBytes int64     `json:"file_size_bytes"`
	ModTime       time.Time `json:"mod_time"`
}

func Scan(fsys fs.FS, root string, opts Options) ([]string, error) {
	records, err := ScanRecords(fsys, root, opts)
	if err != nil {
		return nil, err
	}

	matches := make([]string, 0, len(records))
	for _, r := range records {
		matches = append(matches, r.Path)
	}
	return matches, nil
}

func ScanRecords(fsys fs.FS, root string, opts Options) ([]Record, error) {
	if opts.MaxDepth < -1 {
		return nil, fs.ErrInvalid
	}

	photoExts := normalizeExts(opts.PhotoExtensions)
	videoExts := normalizeExts(opts.VideoExtensions)

	var matches []Record

	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if opts.MaxDepth >= 0 {
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					return relErr
				}
				if rel == "." {
					return nil
				}
				if depth(rel) > opts.MaxDepth {
					return fs.SkipDir
				}
			}
			return nil
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}

		if opts.MaxDepth >= 0 && depth(rel) > opts.MaxDepth {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(rel))
		if !(photoExts[ext] || videoExts[ext]) {
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
		}

		matches = append(matches, Record{
			Path:          filepath.ToSlash(rel),
			FileSizeBytes: info.Size(),
			ModTime:       info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Path < matches[j].Path
	})
	return matches, nil
}

func normalizeExts(exts []string) map[string]bool {
	m := make(map[string]bool, len(exts))
	for _, ext := range exts {
		e := strings.TrimSpace(strings.ToLower(ext))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		m[e] = true
	}
	return m
}

func depth(rel string) int {
	rel = filepath.Clean(rel)
	if rel == "." {
		return 0
	}
	return strings.Count(filepath.ToSlash(rel), "/")
}
