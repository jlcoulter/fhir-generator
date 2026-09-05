# Contributing

Thanks for your interest in `fhir-generator`.

## Development

This project is developed with strict test-driven development (TDD): every
behavior change is preceded by a failing test that defines the expected
behavior.

The tests load the sample package `testdata/au-base.tgz` (copied from
`fhir-registry`). Run them with:

```
go test ./...
```

## Guidelines

- **Write the test first.** For a bug fix, add a test that reproduces the bug
  and confirm it fails before fixing it. For a feature, add a test that defines
  the new behavior and confirm it fails for the right reason before
  implementing it.
- **Keep behavior unchanged when refactoring.** Refactor only when the suite is
  green, and re-run the suite to confirm it stays green.
- **Document exported API.** Public types, functions, options, and sentinel
  errors must carry godoc comments following Go conventions, and usage changes
  should be reflected in the README and this project's examples.
- **Update the changelog.** Add entries under `[Unreleased]` in
  `CHANGELOG.md` for user-visible changes, following
  [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## Pull requests

- Keep changes focused and reviewable; prefer small, stacked PRs.
- Ensure `go test ./...` passes.
- Write clear commit messages that describe the behavior change.
