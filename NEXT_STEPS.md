# Next Steps

## 1) Quality and test coverage
- Add end-to-end tests with temporary Git repositories.
- Add deterministic test scenarios for `publish` conflicts and recovery.
- Add parser compatibility tests for `.publicignore` patterns.

## 2) CLI hardening
- Add non-interactive flags to `init` (`--public-url`, `--private-url`).
- Add non-interactive flags to `publish` (`--commits`, `--yes`).
- Improve error messages with explicit remediation commands.

## 3) Security safeguards
- Add optional scan step before `dualgit push` (secrets patterns + deny list).
- Add audit log output for blocked operations.
- Add strict mode checks for branch drift between public/private.

## 4) Distribution and release
- Publish tagged releases with Linux/macOS binaries and checksums.
- Add package metadata for Homebrew tap.
- Add signed artifacts and provenance for releases.

## 5) Documentation
- Add architecture doc with branch model and invariants.
- Add operational playbook (incident: leak risk, conflict, rollback).
- Add migration guide from plain Git workflow.
