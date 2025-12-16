# Duplicate Detection Concept

This document captures the current conclusions about **duplicate detection** for `media-organizer`.

## Scope

- Duplicate detection is treated as a **separate workflow/command** from the main organize pipeline for reporting/measurement across one or more roots (including multiple disks).
- The organize pipeline stays deterministic and copy-oriented, but it also performs **pre-copy exact-dedupe safeguards**:
  - do not copy exact duplicates within a single run
  - do not duplicate bytes when the destination already contains identical content

## Definition of “Duplicate”

- Primary definition: **exact duplicate content** (byte-for-byte identical).
- We may optionally report “candidates” (high confidence) vs “confirmed duplicates” (guaranteed) depending on which verification step is enabled.

## Strategy (Tiered / ELT-like)

The tiered strategy is used both by a future duplicate-reporting command and by the organize pipeline’s pre-copy dedupe stage.

### 1) Candidate grouping by file size
- Files with different sizes cannot be identical.
- Only files in the same **exact size group** require further comparison.

### 2) Cheap filter: header-bytes comparison
- Within a same-size group, read/hash the first `N` bytes (currently `64 KiB`).
  - If the file is smaller than `N`, read the entire file.
- Compare (or hash) those header bytes to split groups.

This step is favored because it is meaningful, low I/O cost, and avoids hashing every file.

### 3) Confirmation step
- For remaining candidates, perform a full byte-for-byte comparison (exact duplicates).

## Organize Pipeline Behavior

Within a single `organize` run:
- Exact duplicate sources are detected and **skipped** (never copied).
- If multiple sources are identical, the canonical file is the **oldest** by best-effort `created_at` (priority `metadata -> filename -> filestat`).
  - Unknown timestamps do not win; ties break deterministically.
- When a destination candidate already exists:
  - if identical, skip
  - if different, select the next suffix path (`_1`, `_2`, …)

## Outputs

Suggested output shapes:
- Human-friendly: groups of duplicate paths per candidate/confirmed key
- Machine-friendly: `--json` or `--ndjson` (one group per line)

## Notes / Non-goals (for now)

- A persistent database (e.g. SQLite) is optional and can be added later to cache fingerprints/hashes and speed up repeated runs.
- “Near duplicate” detection (perceptual hashing for resized/edited images) is out of scope for the initial implementation.
