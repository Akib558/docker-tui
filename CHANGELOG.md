# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Dashboard screenshot in the README.
- Expanded README with clearer installation, usage, configuration, and development docs.
- `CODE_OF_CONDUCT.md` for contributor and community behavior standards.

### Changed

- Normalized module and import path to `github.com/akib558/docker-tui`.
- Updated docs and security links to the current GitHub repository.
- Updated and unignored `.goreleaser.yaml` for release pipeline configuration.
- Updated `.golangci.yml` to a compatible, passing baseline configuration.
- Pinned CI linting to golangci-lint v2.12.2 through `golangci/golangci-lint-action` for Go 1.25 compatibility.
