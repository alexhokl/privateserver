# privateserver CRUSH Profile

This profile guides agentic coding assistants in the `privateserver` repository.

## Commands

- **Build:** `go build -o /dev/null`
- **Test:** `go test ./...`
- **Lint:** `golangci-lint run`
- **Run a single test:** `go test -run ^TestMyTest$`
- **Security check:** `gosec ./...`

## Code Style

### Imports

- Standard library imports are grouped together.
- Third-party imports are grouped together.
- Blank line between import groups.

### Formatting

- Use `gofmt` for all Go code.
- Keep lines under 120 characters.

### Types

- Use structs for configuration objects.
- Use pointers to structs when modifying them.

### Naming Conventions

- Use camelCase for variables and function names.
- Use PascalCase for structs and interfaces.
- Test functions are named `TestXxx`.

### Error Handling

- Use `fmt.Errorf` to wrap errors with context.
- Use `%w` to wrap the original error.
- Handle errors explicitly; do not discard them.

### Testing

- Use table-driven tests for multiple scenarios.
- Use the `testing` package for tests.
