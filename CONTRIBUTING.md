# Contributing to go-iotclient

## Development Environment

- Go `1.21+`
- A platform capable of running `go test -race`

## Workflow

1. Create a feature branch from `main`.
2. Make focused changes with tests.
3. Run:
   - `go test ./...`
   - `go test -race ./...`
   - `go vet ./...`
4. Update docs when behavior, API, or support matrix changes.
5. Open a PR with:
   - clear summary
   - risk/compatibility notes
   - test evidence

## Coding Guidelines

- Keep external APIs backward-compatible whenever possible.
- Return errors through `core.Result` / `core.ResultT[T]` instead of panics.
- Add tests for normal path and failure path.
- Prefer small composable helpers over large monolithic functions.

## Commit Message Convention

Use concise, intent-oriented messages, for example:

- `feat(modbus): add RTU over TCP crc validation`
- `fix(plc): handle malformed allen-bradley responses`
- `docs(parity): update method-level coverage matrix`
