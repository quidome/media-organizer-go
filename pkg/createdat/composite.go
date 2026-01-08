package createdat

import (
	"io"
	"path/filepath"
	"strings"
	"time"
)

type compositeExtractor struct {
	imageExtractor MetadataExtractor
	videoExtractor MetadataExtractor
}

func (e *compositeExtractor) CreatedAt(path string, r io.Reader) (time.Time, bool, error) {
	ext := strings.ToLower(filepath.Ext(path))

	if isVideoExtension(ext) {
		return e.videoExtractor.CreatedAt(path, r)
	}

	return e.imageExtractor.CreatedAt(path, r)
}

func isVideoExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case ".mp4", ".mov", ".m4v", ".mkv", ".avi", ".webm", ".mts", ".3gp":
		return true
	default:
		return false
	}
}
