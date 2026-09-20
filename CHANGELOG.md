# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0](https://github.com/jlcoulter/fhir-generator/compare/v0.1.1...v0.2.0) (2026-09-20)


### Features

* ensure fhir valid records  ([#4](https://github.com/jlcoulter/fhir-generator/issues/4)) ([d88d138](https://github.com/jlcoulter/fhir-generator/commit/d88d1384f6767718b33ba837f69bc7617d07e36c))

## [Unreleased]

### Added

- Package-level documentation in `doc.go` describing the library's purpose and
  quick-start usage.
- Runnable `Example*` tests in `example_test.go` documenting `New`,
  `Generate`, `GenerateForURL`, and the `With*` options.
- `CHANGELOG.md` and `CONTRIBUTING.md`.

### Documentation

- Expanded godoc on the exported `Generator`, `New`, `Option`, `Generate`,
  `GenerateForURL`, `WithValues`, and sentinel error values.
