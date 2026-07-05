# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-07-05

### Added

- Pre-open-source audit implementation across correctness, UI unification, performance, and OSS polish.
- Shared UI components: glyph registry, `ListTable`, `renderViewTitle`, `renderScreen`, `StateBadge`.
- Command palette `Run(Model) (Model, tea.Cmd)` contract with tests.
- Docker events view (`D` key + palette entry).
- `--version` flag.
- Cross-platform clipboard (`pbcopy`, `clip`, OSC52 fallback).
- `docker.FakeClient` test stub and extended `ClientAPI` (volumes).
- Makefile `fmt` / `fmt-check` targets; CI gofmt gate.

### Changed

- Unified progress bars through `renderBarSegments` (list, detail, dashboard).
- Column headers use `colorMuted`; dividers use `colorBorder`; bar empty track uses `colorBarTrack`.
- Filtered container list memoization; dashboard string cache; ordered central-log insert.
- History persistence uses compact JSON and prunes removed containers.
- `truncate()` uses runewidth; all text input uses rune-based helpers.
- Volumes/networks filters use dedicated state (no longer corrupt container cursor).
- `Model.client` typed as `docker.ClientAPI`.

### Fixed

- Command palette actions no longer silently discard model mutations.
- Batch volume/network remove confirm dialogs execute correctly.
- Reconnect carries client through message instead of mutating in tick goroutine.
- `saveHistory` deep-copies maps before async write (race fix).
- `RenderRow` width overflow via improved `fitCells` and hard clamp.
- Detail identity line clamped to terminal width.

### Removed

- Dead layout shims (`renderFullWidthRow`, `renderZebraRow`, `renderListRow`, `ContentWidth`).
- Dead `miniBar`, `progressBar`, `cleanDockerLogs`, `countByState`.
- Unused `logRegex` and `centralLogClipboard` model fields.
- Internal `docs/suggestions.md` notes.

### Previously unreleased

- Dashboard screenshot in the README.
- Expanded README with clearer installation, usage, configuration, and development docs.
- `CODE_OF_CONDUCT.md` for contributor and community behavior standards.
- Normalized module and import path to `github.com/akib558/docker-tui`.
