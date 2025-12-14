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
- stable, sorted list of **root-relative** media paths

Notes
- Extension matching should be case-insensitive.
- Default output should contain **only media files**.

### Stage 2: Attribute Timestamp (CreatedAt)

**Input**
- discovered file paths (and optionally stat/metadata)

**Output**
- enriched records with:
  - `created_at` (timestamp)
  - `created_at_source` (one of `metadata`, `filename`, `mtime`, `unknown`)

Notes
- Track `created_at_source` for explainability and future debugging.
- Decide timezone policy early (how to interpret timestamps without offsets).

### Stage 3: Plan Destination (Partitioning)

**Input**
- `created_at` + original filename/path + destination root

**Output**
- planned operations: `src -> proposedDst`

Rule
- `proposedDst = <dest>/YYYY/MM/DD/<original_filename>`

### Stage 4: Resolve Collisions (Deterministic)

**Input**
- proposed destination paths

**Output**
- finalized operations with unique destination paths

Rules
- Suffix is inserted before the extension: `name_1.jpg`.
- Collision handling should be deterministic and stable.

### Stage 5: Materialize (Copy)

**Input**
- finalized operations

**Output**
- copy results + summary report

Notes
- Keep all filesystem mutation here.
- This stage should support `--dry-run` by skipping writes and only printing the plan.

## Planning vs Execution

A recommended split:

- **Plan**: compute the list of operations (safe; unit test heavily).
- **Execute**: perform the copy operations (small surface area; integration tests).

This enables:
- `--dry-run` (plan only)
- later additions like a journal/undo, incremental behavior, or verification.

## Suggested Outputs

- Default human-friendly mode:
  - prints planned/copied paths, one per line
- Optional machine-friendly mode:
  - `--json` / `--ndjson` for piping and reproducible automation

## Testing Strategy

- Unit tests for stage logic (especially discovery, timestamp attribution rules, path planning, and collision resolution).
- Minimal integration tests at the CLI boundary to confirm flag parsing and wiring.
