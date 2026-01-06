# Development Rules

## Code Quality

### Error Handling
- Return meaningful errors with context: `fmt.Errorf("context: %w", err)`
- Create custom error types for domain-specific errors
- Use sentinel errors (e.g., `var ErrInvalidInput = errors.New("...")`)
- NEVER ignore errors with blank identifiers (`_`)

### Concurrency
- Use `golang.org/x/sync/errgroup` for managing goroutine groups
- Always use context for cancellation and timeouts
- Prevent goroutine leaks - ensure proper termination
- Prefer select statements for multi-channel operations

### Interfaces
- Define interfaces for dependencies to enable loose coupling
- Inject dependencies through constructors
- Keep interfaces focused following Interface Segregation Principle

## Project Structure

Follow standard Go layout:
```
cmd/          - Main applications
internal/     - Private application code
  domain/     - Domain packages (user/, auth/, etc.)
    *.go           - Models
    repository.go  - Data access interface
    service.go     - Business logic
    *_test.go      - Tests
pkg/          - Public library code
configs/      - Configuration files
```

## Testing

### Required Libraries
- `github.com/stretchr/testify/assert` - Assertions
- `github.com/testcontainers/testcontainers-go` - Integration tests

### Patterns
- Use table-driven tests with subtests
- Use fields like `expectErr func(t *testing.T, err error)` in testCases to assert errors (always non-nil and always called in tests)
- Use fields like `expect func(t *testing.T, got T)` in testCases to assert got values (always non-nil and always called in tests)
- Mock dependencies using generated mocks
- Use fields like `assertMocks func(t *testing.T, m1 MockT1, m2 MockT2, ...)` in testCases to assert the mocks were called (when mocks are used - always non-nil and always called in tests)
- Verify mock calls: `assert.Len(t, mockRepo.FindByIDCalls(), 2)`
- Write benchmarks for performance-critical code

