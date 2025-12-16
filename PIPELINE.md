# Media Organizer Pipeline Concept

This document captures the intended high-level architecture for the media organizer as a **multi-stage pipeline**, similar in spirit to dbt/ELT workflows.

The goal is to make each stage small, testable, and composable, with a strict separation between **planning** (pure, safe, easy to test) and **execution** (filesystem writes).

## Big Picture

**Extract** → **Transform** → **Load (materialize)**

- **Extract**: discover media files and (optionally) read metadata.
- **Transform**: derive a canonical `created_at` timestamp and compute destination paths.
- **Load**: copy files into a partitioned library layout.

## Core Workflow

Pipeline intent:

1. Find media files (photos/videos) under a root directory.
2. Determine a best-effort creation date for each file using:
   1. embedded metadata (EXIF / container metadata)
   2. filename parsing
   3. filesystem timestamps (mtime) as a fallback
3. Copy files into:

```
<dest>/YYYY/MM/DD/<original_filename>
```

4. If the destination filename already exists, append a suffix:

- `file.jpg`
- `file_1.jpg`
- `file_2.jpg`

## Why a Pipeline (dbt/ELT mindset)

- Each stage produces a well-defined dataset (records) for the next stage.
- Stages are independently unit-testable.
- Deterministic outputs support idempotency and incremental runs.
- A “manifest”/report can document what happened and why.

## Stages

### Stage 1: Discover (Inventory)

**Input**
- `root` directory
- options like `maxDepth`, extension allowlists

**Output**
- stable, sorted list of **records** (root-relative paths + file stats)

Record fields (current implementation)
- `path` (root-relative, forward-slash)
- `file_size_bytes`
- `mod_time` (mtime)

Notes
- Extension matching is case-insensitive.
- Default output contains **only media files**.

### Stage 2: Attribute Timestamp (CreatedAt)

**Input**
- discovered records (paths + file stats)

**Output**
- enriched records with:
  - `created_at` candidates (dictionary-like):
    - `metadata` (EXIF/container metadata)
    - `filename` (parsed from filename)
    - `filestat` (mtime fallback)
  - `best_created_at` (chosen using priority `metadata -> filename -> filestat`)

Notes
- Keep all candidates for explainability/debugging.
- Decide timezone policy early (how to interpret timestamps without offsets).
- On Linux, the file-stat fallback is mtime (creation time is generally not reliably available).

### Stage 3: Plan Destination (Partitioning)

**Input**
- `best_created_at` (or unknown)
- original filename/path
- destination root

**Output**
- planned operations: `src -> proposedDst`

Rules
- If `best_created_at` is known:
  - `proposedDst = <dest>/YYYY/MM/DD/<original_filename>`
- If `best_created_at` is unknown:
  - `proposedDst = <dest>/unknown/<original_filename>`

### Stage 4: Resolve Collisions (Deterministic)

**Input**
- proposed destination paths

**Output**
- finalized operations with unique destination paths (within the run)

Rules
- Suffix is inserted before the extension: `name_1.jpg`.
- Collision handling is deterministic and stable.

### Stage 4b: Deduplicate Sources (Exact Content)

**Input**
- discovered source files in the current run

**Output**
- keep exactly one file per exact-duplicate group; skip the rest

Rules
- Duplicate definition: exact duplicate content (byte-for-byte identical).
- Canonical choice: keep the oldest `best_created_at` (unknown timestamps do not win; ties break deterministically).
- Uses a tiered approach: size grouping -> header bytes (64KiB) -> full byte comparison.

### Stage 4c: Reconcile Against Destination (Read-only)

**Input**
- planned operations for kept sources

**Output**
- final per-source decision:
  - `copy` / `copy_renamed`
  - `skipped_identical`
  - `skipped_duplicate_source`

Rules
- If a destination candidate exists and is identical, skip.
- If it exists and differs, choose next suffix path.

### Stage 5: Materialize (Copy)

**Input**
- finalized operations/decisions from planning + reconcile stages

**Output**
- copy results + summary report

Notes
- Keep all filesystem mutation here.
- Never overwrite existing files.
- In execute mode, only perform `copy` / `copy_renamed` actions.
- In dry-run mode, print the planned decisions and destinations.

## Planning vs Execution

A recommended split:

- **Plan**: compute the list of operations (safe; unit test heavily).
- **Execute**: perform the copy operations (small surface area; integration tests).

This enables:
- `--dry-run` (plan only)
- later additions like a journal/undo, incremental behavior, or verification.

## Suggested Outputs

- Default human-friendly mode:
  - prints planned/copied/skipped paths, one per line
- Optional machine-friendly mode:
  - `--json` for piping and reproducible automation

Current CLI behavior
- `scan --json`: emits inventory + `created_at` candidates (no destination, no dedupe).
- `organize --json`: emits inventory + `created_at` candidates + destination/decision fields.

## Testing Strategy

- Unit tests for stage logic (especially discovery, timestamp attribution rules, path planning, and collision resolution).
- Minimal integration tests at the CLI boundary to confirm flag parsing and wiring.
