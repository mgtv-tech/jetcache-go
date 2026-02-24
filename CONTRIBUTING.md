# Contributing to jetcache-go

Thank you for contributing.

## Development Setup

## 1. Fork and clone

```bash
git clone https://github.com/<your-account>/jetcache-go.git
cd jetcache-go
```

## 2. Create branch

Use a clear branch name:

- `feature/<short-description>`
- `fix/<short-description>`
- `docs/<short-description>`

## 3. Install dependencies

```bash
go mod tidy
```

## 4. Run checks

```bash
go test ./...
```

If your change affects performance-sensitive paths, also run:

```bash
go test -run '^$' -bench 'BenchmarkOnce|BenchmarkMGet' -benchmem ./...
```

## Pull Request Requirements

A good PR should include:

- clear problem statement,
- change summary,
- compatibility/risk notes,
- test results.

## Code Guidelines

- Keep public API backward compatible unless explicitly discussed.
- Prefer small, focused commits.
- Add tests for new behavior and bug fixes.
- Keep hot path allocations minimal when possible.

## Documentation Guidelines

- Update both English and Chinese docs for user-facing changes.
- Keep README links valid.
- Prefer Mermaid for new architecture/flow diagrams.

## Commit Message Suggestions

Use concise prefixes, for example:

- `feat: add xxx`
- `fix: resolve xxx`
- `docs: update xxx`
- `test: cover xxx`

## Reporting Issues

Please include:

- minimal reproducible example,
- expected vs actual behavior,
- Go version and dependency versions,
- logs and metrics snippets when relevant.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
