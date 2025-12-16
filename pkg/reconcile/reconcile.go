package reconcile

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/quidome/media-organizer-go/pkg/createdat"
	"github.com/quidome/media-organizer-go/pkg/plan"
)

const headerBytes = 64 * 1024

// Action describes what should happen for a source.
type Action string

const (
	ActionCopy                Action = "copy"
	ActionCopyRenamed         Action = "copy_renamed"
	ActionCopied              Action = "copied"
	ActionCopiedRenamed       Action = "copied_renamed"
	ActionSkippedIdentical    Action = "skipped_identical"
	ActionSkippedDuplicateSrc Action = "skipped_duplicate_source"
	ActionFailed              Action = "failed"
)

// Decision describes what should happen for a given source file.
type Decision struct {
	SourcePath      string
	DestinationPath string // planned destination

	FinalDestinationPath string
	Action               Action

	DuplicateOf string
	Error       error
}

// DedupeSources groups source files by exact content and chooses a single canonical file
// per duplicate group.
//
// If multiple sources are identical, it keeps the oldest (earliest) Best.CreatedAt timestamp.
// When timestamps tie (or are zero), it uses lexicographic SourcePath ordering.
func DedupeSources(sources []string, details map[string]createdat.DetailedResult, sizes map[string]int64) (kept []string, decisions []Decision, err error) {
	bySize := make(map[int64][]string)
	for _, p := range sources {
		size, ok := sizes[p]
		if !ok {
			return nil, nil, fmt.Errorf("missing size for %s", p)
		}
		bySize[size] = append(bySize[size], p)
	}

	keptSet := make(map[string]bool)
	skipSet := make(map[string]bool)
	duplicateOf := make(map[string]string)

	for size, paths := range bySize {
		if len(paths) == 1 {
			keptSet[paths[0]] = true
			continue
		}

		// Group by header hash.
		headerGroups := make(map[[32]byte][]string)
		for _, p := range paths {
			h, hashErr := headerHash(p, size)
			if hashErr != nil {
				return nil, nil, hashErr
			}
			headerGroups[h] = append(headerGroups[h], p)
		}

		for _, candidates := range headerGroups {
			if len(candidates) == 1 {
				keptSet[candidates[0]] = true
				continue
			}

			// Partition into exact-equality clusters.
			reps := make([]string, 0)
			clusters := make(map[string][]string) // rep -> members
			for _, p := range candidates {
				assigned := false
				for _, rep := range reps {
					identical, cmpErr := filesAreIdentical(p, rep)
					if cmpErr != nil {
						return nil, nil, cmpErr
					}
					if identical {
						clusters[rep] = append(clusters[rep], p)
						assigned = true
						break
					}
				}
				if !assigned {
					reps = append(reps, p)
					clusters[p] = []string{p}
				}
			}

			// For each cluster, choose the canonical one.
			for _, rep := range reps {
				members := clusters[rep]
				canon := pickOldest(members, details)
				keptSet[canon] = true
				for _, m := range members {
					if m == canon {
						continue
					}
					skipSet[m] = true
					duplicateOf[m] = canon
				}
			}
		}
	}

	decisions = make([]Decision, 0, len(sources))
	kept = make([]string, 0, len(sources))
	for _, p := range sources {
		if skipSet[p] {
			decisions = append(decisions, Decision{SourcePath: p, Action: ActionSkippedDuplicateSrc, DuplicateOf: duplicateOf[p]})
			continue
		}
		if keptSet[p] {
			kept = append(kept, p)
			decisions = append(decisions, Decision{SourcePath: p, Action: ActionCopy})
			continue
		}

		// Should not happen, but fail safe.
		decisions = append(decisions, Decision{SourcePath: p, Action: ActionFailed})
	}

	return kept, decisions, nil
}

// PlanDestinations plans deterministic destination paths for the kept sources.
//
// If a file has no known created_at, it is placed under:
//
//	<destRoot>/unknown/<filename>
func PlanDestinations(destRoot string, sources []string, bestCreatedAt map[string]time.Time) ([]plan.Operation, error) {
	existing := make(map[string]bool)
	ops := make([]plan.Operation, 0, len(sources))
	for _, src := range sources {
		filename := filepath.Base(src)

		createdAt, ok := bestCreatedAt[src]
		var dst string
		if ok && !createdAt.IsZero() {
			dst = plan.Destination(destRoot, filename, createdAt, existing)
		} else {
			dst = unknownDestination(destRoot, filename, existing)
		}

		existing[dst] = true
		ops = append(ops, plan.Operation{SourcePath: src, DestinationPath: dst})
	}
	return ops, nil
}

func unknownDestination(destRoot, filename string, existing map[string]bool) string {
	dir := filepath.Join(destRoot, "unknown")

	basePath := filepath.Join(dir, filename)
	if !existing[basePath] {
		existing[basePath] = true
		return basePath
	}

	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)
	for i := 1; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s_%d%s", nameWithoutExt, i, ext))
		if !existing[candidate] {
			existing[candidate] = true
			return candidate
		}
	}
}

// ResolveAgainstDestination checks for existing destination files.
// - If identical content exists at the planned destination, it marks skipped.
// - If different content exists, it searches for the next suffix path.
func ResolveAgainstDestination(ops []plan.Operation) ([]Decision, error) {
	decisions := make([]Decision, 0, len(ops))
	reserved := make(map[string]bool)

	for _, op := range ops {
		planned := op.DestinationPath
		destDir := filepath.Dir(planned)

		filename := filepath.Base(op.SourcePath)
		ext := filepath.Ext(filename)
		base := strings.TrimSuffix(filename, ext)

		var final string
		var action Action

		for n := 0; ; n++ {
			var candidate string
			if n == 0 {
				candidate = filepath.Join(destDir, filename)
			} else {
				candidate = filepath.Join(destDir, fmt.Sprintf("%s_%d%s", base, n, ext))
			}

			if reserved[candidate] {
				continue
			}

			st, err := os.Stat(candidate)
			if err != nil {
				if os.IsNotExist(err) {
					final = candidate
					if n == 0 {
						action = ActionCopy
					} else {
						action = ActionCopyRenamed
					}
					reserved[candidate] = true
					break
				}
				return nil, fmt.Errorf("stat %s: %w", candidate, err)
			}

			_ = st
			identical, cmpErr := filesAreIdentical(op.SourcePath, candidate)
			if cmpErr != nil {
				return nil, cmpErr
			}
			if identical {
				final = candidate
				action = ActionSkippedIdentical
				break
			}
		}

		decisions = append(decisions, Decision{
			SourcePath:           op.SourcePath,
			DestinationPath:      planned,
			FinalDestinationPath: final,
			Action:               action,
		})
	}

	return decisions, nil
}

func pickOldest(paths []string, details map[string]createdat.DetailedResult) string {
	best := ""
	bestTime := time.Time{}
	for _, p := range paths {
		t := details[p].Best.CreatedAt
		if t.IsZero() {
			// Treat unknown as newest.
			continue
		}
		if best == "" || bestTime.IsZero() || t.Before(bestTime) || (t.Equal(bestTime) && p < best) {
			best = p
			bestTime = t
		}
	}
	if best != "" {
		return best
	}

	// If everything is unknown, choose lexicographically smallest for stability.
	best = paths[0]
	for _, p := range paths[1:] {
		if p < best {
			best = p
		}
	}
	return best
}

func headerHash(path string, size int64) ([32]byte, error) {
	limit := headerBytes
	if size < int64(headerBytes) {
		limit = int(size)
	}

	f, err := os.Open(path)
	if err != nil {
		return [32]byte{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.CopyN(h, f, int64(limit)); err != nil && err != io.EOF {
		return [32]byte{}, fmt.Errorf("read header %s: %w", path, err)
	}

	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out, nil
}

func filesAreIdentical(path1, path2 string) (bool, error) {
	info1, err := os.Stat(path1)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", path1, err)
	}
	info2, err := os.Stat(path2)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", path2, err)
	}
	if info1.Size() != info2.Size() {
		return false, nil
	}

	// Header compare.
	size := info1.Size()
	limit := headerBytes
	if size < int64(headerBytes) {
		limit = int(size)
	}
	buf1 := make([]byte, limit)
	buf2 := make([]byte, limit)
	f1, err := os.Open(path1)
	if err != nil {
		return false, fmt.Errorf("open %s: %w", path1, err)
	}
	defer f1.Close()
	f2, err := os.Open(path2)
	if err != nil {
		return false, fmt.Errorf("open %s: %w", path2, err)
	}
	defer f2.Close()

	n1, err1 := io.ReadFull(f1, buf1)
	n2, err2 := io.ReadFull(f2, buf2)
	if err1 != nil && err1 != io.EOF && err1 != io.ErrUnexpectedEOF {
		return false, fmt.Errorf("read %s: %w", path1, err1)
	}
	if err2 != nil && err2 != io.EOF && err2 != io.ErrUnexpectedEOF {
		return false, fmt.Errorf("read %s: %w", path2, err2)
	}
	if n1 != n2 {
		return false, nil
	}
	for i := 0; i < n1; i++ {
		if buf1[i] != buf2[i] {
			return false, nil
		}
	}
	if int64(limit) >= size {
		return true, nil
	}

	// Full compare remainder.
	buf1 = make([]byte, 32*1024)
	buf2 = make([]byte, 32*1024)
	for {
		n1, err1 := f1.Read(buf1)
		n2, err2 := f2.Read(buf2)
		if n1 != n2 {
			return false, nil
		}
		for i := 0; i < n1; i++ {
			if buf1[i] != buf2[i] {
				return false, nil
			}
		}
		if err1 == io.EOF && err2 == io.EOF {
			return true, nil
		}
		if err1 != nil {
			return false, fmt.Errorf("read %s: %w", path1, err1)
		}
		if err2 != nil {
			return false, fmt.Errorf("read %s: %w", path2, err2)
		}
	}
}

var reSuffix = regexp.MustCompile(`^(.*)_(\d+)$`)

func nextSuffix(path string) string {
	dir := filepath.Dir(path)
	file := filepath.Base(path)
	ext := filepath.Ext(file)
	name := file[:len(file)-len(ext)]

	if m := reSuffix.FindStringSubmatch(name); m != nil {
		n, err := strconv.Atoi(m[2])
		if err == nil {
			return filepath.Join(dir, fmt.Sprintf("%s_%d%s", m[1], n+1, ext))
		}
	}

	return filepath.Join(dir, fmt.Sprintf("%s_1%s", name, ext))
}
