package createdat

import (
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

// Source describes where a CreatedAt timestamp was derived from.
//
// The priority order is:
//  1. metadata
//  2. filename
//  3. mtime
//  4. unknown
type Source string

const (
	SourceMetadata Source = "metadata"
	SourceFilename Source = "filename"
	SourceMtime    Source = "mtime"
	SourceUnknown  Source = "unknown"
)

// Result contains a best-effort creation timestamp and its source.
type Result struct {
	CreatedAt time.Time
	Source    Source
}

// DetailedResult contains all considered timestamps from different sources.
type DetailedResult struct {
	// Best is the chosen timestamp using priority: metadata > filename > mtime
	Best Result

	// Metadata is the timestamp extracted from embedded metadata (EXIF, etc.)
	Metadata time.Time

	// Filename is the timestamp parsed from the filename
	Filename time.Time

	// Filestat is the mtime from filesystem metadata
	Filestat time.Time
}

// MetadataExtractor extracts an embedded creation timestamp from a media stream.
//
// Implementations should return (t, true, nil) when a timestamp is found.
// If no timestamp exists, return (time.Time{}, false, nil).
// Errors are treated as best-effort failures by Determine.
type MetadataExtractor interface {
	CreatedAt(path string, r io.Reader) (time.Time, bool, error)
}

// Options configures Determine.
type Options struct {
	// Location is used for timestamps parsed from filenames that contain no timezone.
	// If nil, time.Local is used.
	Location *time.Location

	// Metadata optionally extracts embedded timestamps.
	//
	// If nil, a default EXIF-based extractor is used.
	Metadata MetadataExtractor
}

// Determine returns the best-effort created-at timestamp for a path.
func Determine(fsys fs.FS, path string, opts Options) (Result, error) {
	detailed, err := DetermineDetailed(fsys, path, opts)
	if err != nil {
		return Result{}, err
	}
	return detailed.Best, nil
}

// DetermineDetailed returns all considered timestamps for a path.
func DetermineDetailed(fsys fs.FS, path string, opts Options) (DetailedResult, error) {
	path = filepath.Clean(path)

	info, err := fs.Stat(fsys, path)
	if err != nil {
		return DetailedResult{}, err
	}
	if info.IsDir() {
		return DetailedResult{}, fs.ErrInvalid
	}

	var result DetailedResult

	// Try metadata
	metadata := opts.Metadata
	if metadata == nil {
		metadata = exifExtractor{}
	}

	if metadata != nil {
		f, openErr := fsys.Open(path)
		if openErr != nil {
			return DetailedResult{}, openErr
		}
		createdAt, ok, metaErr := metadata.CreatedAt(path, f)
		_ = f.Close()
		if metaErr == nil && ok {
			result.Metadata = createdAt
		}
	}

	// Try filename
	loc := opts.Location
	if loc == nil {
		loc = time.Local
	}
	if createdAt, ok := parseFromFilename(filepath.Base(path), loc); ok {
		result.Filename = createdAt
	}

	// Get mtime
	mtime := info.ModTime()
	if !mtime.IsZero() {
		result.Filestat = mtime
	}

	// Determine best according to priority
	if !result.Metadata.IsZero() {
		result.Best = Result{CreatedAt: result.Metadata, Source: SourceMetadata}
	} else if !result.Filename.IsZero() {
		result.Best = Result{CreatedAt: result.Filename, Source: SourceFilename}
	} else if !result.Filestat.IsZero() {
		result.Best = Result{CreatedAt: result.Filestat, Source: SourceMtime}
	} else {
		result.Best = Result{CreatedAt: time.Time{}, Source: SourceUnknown}
	}

	return result, nil
}

var (
	reImgVidDateTime = regexp.MustCompile(`(?i)^(?:IMG|VID)_(\d{8})_(\d{6})`)
	rePxlDateTimeMs  = regexp.MustCompile(`(?i)^PXL_(\d{8})_(\d{6})\d{3,}`)
	reDashDots       = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})[ _](\d{2})\.(\d{2})\.(\d{2})`)
	reImgWhatsApp    = regexp.MustCompile(`(?i)^IMG-(\d{8})-WA\d+`)
	reScreenshot     = regexp.MustCompile(`(?i)^Screenshot_(\d{4})-(\d{2})-(\d{2})-(\d{2})-(\d{2})-(\d{2})`)
)

func parseFromFilename(filename string, loc *time.Location) (time.Time, bool) {
	if m := reImgVidDateTime.FindStringSubmatch(filename); m != nil {
		return parseYYYYMMDD_HHMMSS(m[1], m[2], loc)
	}
	if m := rePxlDateTimeMs.FindStringSubmatch(filename); m != nil {
		return parseYYYYMMDD_HHMMSS(m[1], m[2], loc)
	}
	if m := reDashDots.FindStringSubmatch(filename); m != nil {
		y, ok := atoi(m[1])
		if !ok {
			return time.Time{}, false
		}
		mo, ok := atoi(m[2])
		if !ok {
			return time.Time{}, false
		}
		d, ok := atoi(m[3])
		if !ok {
			return time.Time{}, false
		}
		h, ok := atoi(m[4])
		if !ok {
			return time.Time{}, false
		}
		mi, ok := atoi(m[5])
		if !ok {
			return time.Time{}, false
		}
		s, ok := atoi(m[6])
		if !ok {
			return time.Time{}, false
		}
		return time.Date(y, time.Month(mo), d, h, mi, s, 0, loc), true
	}
	if m := reImgWhatsApp.FindStringSubmatch(filename); m != nil {
		yyyymmdd := m[1]
		y, mo, d, ok := parseYYYYMMDD(yyyymmdd)
		if !ok {
			return time.Time{}, false
		}
		return time.Date(y, time.Month(mo), d, 0, 0, 0, 0, loc), true
	}
	if m := reScreenshot.FindStringSubmatch(filename); m != nil {
		y, ok := atoi(m[1])
		if !ok {
			return time.Time{}, false
		}
		mo, ok := atoi(m[2])
		if !ok {
			return time.Time{}, false
		}
		d, ok := atoi(m[3])
		if !ok {
			return time.Time{}, false
		}
		h, ok := atoi(m[4])
		if !ok {
			return time.Time{}, false
		}
		mi, ok := atoi(m[5])
		if !ok {
			return time.Time{}, false
		}
		s, ok := atoi(m[6])
		if !ok {
			return time.Time{}, false
		}
		return time.Date(y, time.Month(mo), d, h, mi, s, 0, loc), true
	}

	return time.Time{}, false
}

func parseYYYYMMDD_HHMMSS(yyyymmdd, hhmmss string, loc *time.Location) (time.Time, bool) {
	y, mo, d, ok := parseYYYYMMDD(yyyymmdd)
	if !ok {
		return time.Time{}, false
	}
	if len(hhmmss) != 6 {
		return time.Time{}, false
	}
	h, ok := atoi(hhmmss[0:2])
	if !ok {
		return time.Time{}, false
	}
	mi, ok := atoi(hhmmss[2:4])
	if !ok {
		return time.Time{}, false
	}
	s, ok := atoi(hhmmss[4:6])
	if !ok {
		return time.Time{}, false
	}
	return time.Date(y, time.Month(mo), d, h, mi, s, 0, loc), true
}

func parseYYYYMMDD(yyyymmdd string) (year int, month int, day int, ok bool) {
	if len(yyyymmdd) != 8 {
		return 0, 0, 0, false
	}
	y, ok := atoi(yyyymmdd[0:4])
	if !ok {
		return 0, 0, 0, false
	}
	mo, ok := atoi(yyyymmdd[4:6])
	if !ok {
		return 0, 0, 0, false
	}
	d, ok := atoi(yyyymmdd[6:8])
	if !ok {
		return 0, 0, 0, false
	}
	return y, mo, d, true
}

func atoi(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}
