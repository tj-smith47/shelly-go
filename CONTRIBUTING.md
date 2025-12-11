# Contributing to shelly-go

First off, thank you for considering contributing to shelly-go! This library aims to provide comprehensive support for all Shelly device generations and we welcome contributions from the community.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
- [Development Setup](#development-setup)
- [Coding Guidelines](#coding-guidelines)
- [Testing Requirements](#testing-requirements)
- [Pull Request Process](#pull-request-process)
- [Documentation](#documentation)

## Code of Conduct

This project adheres to a code of professional and respectful conduct. By participating, you are expected to:

- Use welcoming and inclusive language
- Be respectful of differing viewpoints and experiences
- Gracefully accept constructive criticism
- Focus on what is best for the community
- Show empathy towards other community members

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When creating a bug report, include:

- **Clear descriptive title**
- **Exact steps to reproduce** the problem
- **Expected behavior** vs **actual behavior**
- **Code samples** demonstrating the issue
- **Device information** (generation, model, firmware version)
- **Go version** and OS details

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

- Use a clear and descriptive title
- Provide a detailed description of the proposed functionality
- Explain why this enhancement would be useful
- Include code examples if applicable

### Adding Device Support

To add support for a new Shelly device:

1. Create device profile in `profiles/gen{1,2,3,4}/`
2. Add component implementations if new components are needed
3. Add comprehensive tests with ≥90% coverage
4. Update `DEVICES.md` with device information
5. Add example usage in `examples/`

### Pull Requests

- Fill in the required template
- Follow the coding guidelines
- Include tests for all changes (≥90% coverage required)
- Update documentation for API changes
- Ensure all tests pass and linter checks succeed

## Development Setup

### Prerequisites

- **Go 1.25.5 or later** (required for latest features)
- **golangci-lint** for code quality checks
- **make** (optional, for convenience commands)

### Setup Steps

```bash
# Clone the repository
git clone https://github.com/tj-smith47/shelly-go.git
cd shelly-go

# Install dependencies
go mod download

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run tests
go test ./...

# Run linter
golangci-lint run
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run tests for specific package
go test ./gen2/components/...

# Run integration tests (requires real device or mocks)
go test -tags=integration ./...
```

## Coding Guidelines

### General Principles

1. **Simplicity over cleverness** - Write clear, maintainable code
2. **Interface-driven design** - Depend on interfaces, not implementations
3. **Context propagation** - All operations should accept `context.Context`
4. **Error handling** - Use error wrapping with `%w`, define sentinel errors
5. **Thread safety** - Document concurrency guarantees, protect shared state
6. **Future-proofing** - Use `RawFields map[string]json.RawMessage` for extensibility

### Code Style

- Follow standard Go formatting (`gofmt`, `goimports`)
- Use meaningful variable and function names
- Keep functions focused and small
- Add comments for exported types and functions (godoc format)
- Use the Options pattern for configuration

### Package Organization

```
- types/        # Core interfaces and types
- transport/    # Communication layer
- rpc/          # RPC framework
- gen1/         # Generation 1 device support
- gen2/         # Generation 2+ device support
- cloud/        # Cloud API integration
- discovery/    # Device discovery
- helpers/      # Convenience utilities
```

### Naming Conventions

- **Interfaces**: Describe capability (e.g., `Transport`, `Device`)
- **Implementations**: Concrete name (e.g., `HTTPTransport`, `Gen2Device`)
- **Methods**: Use verb-noun pattern (e.g., `GetConfig`, `SetStatus`)
- **Constants**: Use descriptive names (e.g., `DefaultTimeout`)

### Error Handling

```go
// Define sentinel errors
var (
    ErrNotFound    = errors.New("shelly: resource not found")
    ErrAuth        = errors.New("shelly: authentication failed")
    ErrTimeout     = errors.New("shelly: operation timed out")
)

// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to get device status: %w", err)
}
```

### Options Pattern

```go
type ClientOption func(*Client)

func WithTimeout(d time.Duration) ClientOption {
    return func(c *Client) {
        c.timeout = d
    }
}

// Usage
client := NewClient("http://192.168.1.100",
    WithTimeout(30*time.Second),
    WithAuth("user", "password"))
```

## Testing Requirements

### Coverage Target

- **Minimum 90% line coverage** for all packages
- **Minimum 85% branch coverage**
- All exported functions must have tests
- Document untestable code with `// Coverage: skip - reason`

### Test Structure

```go
func TestComponentName_MethodName(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {
            name:    "successful case",
            input:   InputType{...},
            want:    OutputType{...},
            wantErr: false,
        },
        {
            name:    "error case",
            input:   InputType{...},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MethodName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("MethodName() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("MethodName() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Mock Usage

Use the test utilities in `internal/testutil/`:

```go
import "github.com/tj-smith47/shelly-go/internal/testutil"

func TestWithMock(t *testing.T) {
    mock := testutil.NewMockTransport()
    mock.AddResponse("Shelly.GetDeviceInfo", testutil.LoadFixture("gen2/device_info.json"))

    client := NewClient(mock)
    // Test using mock...
}
```

### Integration Tests

Mark integration tests that require real hardware:

```go
//go:build integration

func TestRealDevice(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    // Test with real device...
}
```

## Pull Request Process

1. **Fork** the repository and create your branch from `main`
2. **Make your changes** following coding guidelines
3. **Add tests** to achieve ≥90% coverage
4. **Update documentation** (README, godoc, examples)
5. **Run tests and linter** locally
6. **Commit** with clear, descriptive messages
7. **Push** to your fork and submit a pull request
8. **Address review feedback** promptly

### Commit Messages

Follow conventional commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding/updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvement
- `chore`: Maintenance tasks

Examples:
```
feat(gen2): add Smoke component support

Implements the Smoke component with GetConfig, SetConfig, GetStatus, and Mute methods.
Includes comprehensive tests achieving 92% coverage.

Closes #123

fix(transport): handle connection timeout correctly

The HTTP transport was not respecting the context deadline.
This fix ensures timeouts are properly propagated.

Fixes #456
```

### PR Checklist

Before submitting, ensure:

- [ ] Code follows the style guidelines
- [ ] All tests pass (`go test ./...`)
- [ ] Linter passes (`golangci-lint run`)
- [ ] Test coverage ≥90% for new code
- [ ] Documentation is updated
- [ ] Examples are added for new features
- [ ] CHANGELOG.md is updated
- [ ] Commit messages follow conventional format

## Documentation

### Godoc Comments

All exported types, functions, and constants must have godoc comments:

```go
// Device represents a Shelly device and provides methods for
// device interaction across different generations (Gen1, Gen2, Gen3, Gen4).
//
// Implementations should be safe for concurrent use.
type Device interface {
    // GetDeviceInfo returns device identification and capabilities.
    GetDeviceInfo(ctx context.Context) (*DeviceInfo, error)

    // GetStatus returns the current status of all device components.
    GetStatus(ctx context.Context) (*Status, error)
}
```

### Examples

Add testable examples in `_test.go` files:

```go
func ExampleSwitch_Set() {
    client := NewClient("http://192.168.1.100")
    sw := gen2.NewSwitch(client, 0)

    err := sw.Set(context.Background(), true)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Switch turned on")
    // Output: Switch turned on
}
```

### Documentation Files

Update these files as appropriate:

- **README.md** - Overview, quick start, examples
- **DEVICES.md** - Supported device matrix
- **ARCHITECTURE.md** - Design decisions, patterns
- **MIGRATION.md** - Migration guide from other libraries
- **CHANGELOG.md** - All notable changes

## Questions?

Feel free to open an issue with the `question` label if you have any questions about contributing.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
