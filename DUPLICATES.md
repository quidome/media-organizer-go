# Duplicate Detection Concept

This document captures the current conclusions about **duplicate detection** for `media-organizer`.

## Scope

- Duplicate detection is treated as a **separate workflow/command** from the main organize pipeline.
- The organize pipeline can stay “copy-first” and deterministic, while duplicate detection is used to measure and report duplicates across one or more roots (including multiple disks).

## Definition of “Duplicate”

- Primary definition: **exact duplicate content** (byte-for-byte identical).
- We may optionally report “candidates” (high confidence) vs “confirmed duplicates” (guaranteed) depending on which verification step is enabled.

## Strategy (Tiered / ELT-like)

### 1) Candidate grouping by file size
- Files with different sizes cannot be identical.
- Therefore, only files in the same **exact size group** require further comparison.

### 2) Cheap filter: header-bytes comparison
- Within a same-size group, read the first `N` bytes (e.g. 64 KiB) of each file.
  - If the file is smaller than `N`, read the entire file.
- Compare (or hash) those header bytes to split groups.

This step is favored because it is **meaningful** and **low I/O cost**, and it avoids hashing every file.

### 3) Optional confirmation step
Header-bytes matching is a strong filter but is not a formal proof of equality (files could differ after the header).

If the command should output **confirmed duplicates**, add one of:
- Full-content hash (e.g. SHA-256) computed only within remaining candidate groups
- Streaming byte-for-byte comparison between files in the remaining groups

## Outputs

Suggested output shapes:
- Human-friendly: groups of duplicate paths per candidate/confirmed key
- Machine-friendly: `--json` or `--ndjson` (one group per line)

## Notes / Non-goals (for now)

- A persistent database (e.g. SQLite) is optional and can be added later to cache fingerprints/hashes and speed up repeated runs.
- “Near duplicate” detection (perceptual hashing for resized/edited images) is out of scope for the initial implementation.
