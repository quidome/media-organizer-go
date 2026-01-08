package createdat

import (
	"io"
	"time"

	"github.com/tajtiattila/metadata/mp4"
)

type videoExtractor struct{}

func (e videoExtractor) CreatedAt(path string, r io.Reader) (time.Time, bool, error) {
	file, err := mp4.Parse(r)
	if err != nil {
		return time.Time{}, false, nil
	}

	if file.Header != nil && !file.Header.DateCreated.IsZero() {
		return file.Header.DateCreated, true, nil
	}

	return time.Time{}, false, nil
}
