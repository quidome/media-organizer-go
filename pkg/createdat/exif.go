package createdat

import (
	"io"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

type exifExtractor struct{}

func (e exifExtractor) CreatedAt(path string, r io.Reader) (time.Time, bool, error) {
	x, err := exif.Decode(r)
	if err != nil {
		// Best-effort: if decode isn't critical, try to salvage timestamps.
		if !exif.IsCriticalError(err) {
			// exif.Decode returns a partially-populated *Exif in these cases.
			// Unfortunately the library doesn't expose it when returning error,
			// so treat it as "not found".
			return time.Time{}, false, nil
		}
		return time.Time{}, false, nil
	}

	// Prefer DateTimeOriginal, then DateTimeDigitized, then DateTime.
	if tm, ok, err := exifTimeFromTag(x, exif.DateTimeOriginal); err == nil && ok {
		return tm, true, nil
	}
	if tm, ok, err := exifTimeFromTag(x, exif.DateTimeDigitized); err == nil && ok {
		return tm, true, nil
	}
	if tm, ok, err := exifTimeFromTag(x, exif.DateTime); err == nil && ok {
		return tm, true, nil
	}
	if t, err := x.DateTime(); err == nil {
		return t, true, nil
	}

	return time.Time{}, false, nil
}

func exifTimeFromTag(x *exif.Exif, tag exif.FieldName) (time.Time, bool, error) {
	f, err := x.Get(tag)
	if err != nil {
		return time.Time{}, false, nil
	}

	s, err := f.StringVal()
	if err != nil {
		return time.Time{}, false, nil
	}

	// EXIF DateTime format: "2006:01:02 15:04:05".
	// It often has no timezone; interpret as Local.
	tm, err := time.ParseInLocation("2006:01:02 15:04:05", s, time.Local)
	if err != nil {
		return time.Time{}, false, nil
	}

	return tm, true, nil
}
