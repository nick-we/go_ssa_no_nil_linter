# gRPC Nil-Value Linter

A static analysis tool for Go that detects potential nil pointer dereferences in gRPC endpoint response messages using SSA (Static Single Assignment) analysis.

## Problem Statement

In gRPC services, returning response messages with nil values in non-optional fields can cause runtime panics for clients. While scalar types, maps, and repeated fields safely default to zero values when nil, **pointer fields representing sub-messages or Well-Known Types (like `google.protobuf.Timestamp`) will cause nil pointer dereferences** if accessed by clients expecting valid values.

This linter uses deep data flow analysis to detect such issues before they reach production.

## Features

- **Deep SSA Analysis**: Traces data flow through your entire codebase using Go's SSA representation
- **Interprocedural Analysis**: Follows function calls across the entire call graph to detect nil propagation
- **Proto-Aware**: Understands protobuf-generated types and distinguishes between optional and required fields
- **Low False Positives**: Conservative analysis with confidence scoring
- **Clear Reports**: Actionable error messages with full trace chains showing how nil values propagate

## What It Detects

### ✅ Safe (No Warning)
```go
// Scalar fields - safe when nil (become zero values)
response.Count = someNilableInt

// Maps and slices - safe when nil (become empty)
response.Tags = someNilableSlice
response.Metadata = someNilableMap

// Optional fields (marked with oneof)
response.OptionalField = someNilableValue
```

### ⚠️ Risky (Will Warn)
```go
// Sub-message pointers - will cause nil panic
response.Profile = getUserProfile() // if this returns nil
response.AllProfiles[0] = getUserProfile() // if this returns nil, or if the returned user profile has one of it's non-optional fields set to nil (implicitely or explicitly)

// Well-Known Types - will cause nil panic
response.CreatedAt = getTimestamp() // if this returns nil
response.UpdatedAt = &timestamppb.Timestamp{} // accessing nil fields inside
```

## Installation

```bash
go install github.com/nick-we/go_ssa_no_nil_linter/cmd/grpc-nil-linter@latest
```

## Usage

### Analyze your code

```bash
# Analyze current package
grpc-nil-linter ./...

# Analyze specific package
grpc-nil-linter ./internal/service

# Output as JSON
grpc-nil-linter -format json ./...

# Verbose mode with full traces
grpc-nil-linter -v ./...
```

### Example Output

```
internal/service/user_service.go:45:2: potential nil field in gRPC response
  Handler:    UserService.GetUser
  Response:   *pb.GetUserResponse
  Field:      Profile (*pb.UserProfile)
  Status:     MaybeNil
  Confidence: 85%
  
  Trace:
    1. Profile assigned at user_service.go:45
    2. Value from getUserProfile() at user_service.go:50
    3. getUserProfile may return nil at helpers.go:23
  
  Suggestion: Check if Profile is nil before assignment
              or ensure getUserProfile never returns nil
```

## How It Works

The linter performs sophisticated static analysis using four main components:

1. **Proto Field Analyzer**: Identifies which fields in proto-generated structs are risky (pointer types representing sub-messages)

2. **gRPC Handler Detector**: Finds methods that implement gRPC service handlers by analyzing method signatures and receiver types

3. **SSA-Based Nil Flow Analyzer**: Uses Go's SSA (Static Single Assignment) representation to trace data flow and identify potential nil values

4. **Call Graph Traversal**: Performs interprocedural analysis by following function calls across the entire codebase

### Analysis Flow

```
Source Code → SSA → Call Graph → Handler Detection → Field Analysis → Nil Tracing → Report
```

## Architecture

See [`ARCHITECTURE.md`](ARCHITECTURE.md) for detailed system design and algorithms.

See [`IMPLEMENTATION_PLAN.md`](IMPLEMENTATION_PLAN.md) for component breakdown and implementation roadmap.

## Example: What Gets Caught

```go
// user_service.go
func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    user := s.repo.FindUser(req.UserId)
    
    return &pb.GetUserResponse{
        User: &pb.User{
            Id:      user.ID,
            Name:    user.Name,
            Profile: s.buildProfile(user), // ⚠️ buildProfile might return nil!
            // CreatedAt: date.Now(), ⚠️ non-optional field CreatedAt is not set and implicitely nil!
        },
    }, nil
}

func (s *UserService) buildProfile(user *model.User) *pb.UserProfile {
    if user.ProfileID == 0 {
        return nil // ⚠️ This nil will be caught by the linter!
    }
    // ... build profile
}

func (s *UserService) ListUsers(ctx context.Context, req *pb.ListUserRequest) (*pb.ListUsersResponse, error) {
    users := s.repo.ListUsers()
    
    return &pb.ListUsersResponse{
        Users: []*pb.User{
            {
                Id:      user.ID,
                Name:    user.Name,
                Profile: s.buildProfile(user), // ⚠️ buildProfile might return nil!
                // CreatedAt: date.Now(), ⚠️ non-optional field CreatedAt is not set and implicitely nil!
            },
        },
    }, nil
}
```

The linter will report:
```
user_service.go:8:13: potential nil field in gRPC response
  Field: Profile (*pb.UserProfile)
  The value assigned to Profile comes from buildProfile() which may return nil
  at user_service.go:15

user_service.go:5:8: implicit nil assignment to non-optional field in gRPC response
  Field: CreatedAt (*pb.User)

user_service.go:25:15: potential nil field in gRPC response
  Users[0]: Field: Profile (*pb.UserProfile)
  The value assigned to Profile comes from buildProfile() which may return nil
  at user_service.go:15

user_service.go:25:15: implicit nil assignment to non-optional field in gRPC response
  Users[0]: Field: CreatedAt (*pb.User)
```

## Technical Details

### SSA Instructions Analyzed

The linter handles these SSA instruction types:
- `*ssa.Alloc`: New allocations (always non-nil)
- `*ssa.Const`: Nil constants (always nil)
- `*ssa.Call`: Function calls (analyzed recursively)
- `*ssa.Phi`: Control flow merges (pessimistic analysis)
- `*ssa.FieldAddr`, `*ssa.Field`: Field access (trace base object)
- `*ssa.Store`: Assignments (track what gets assigned where)

### Call Graph Construction

Uses Rapid Type Analysis (RTA) to build a static call graph, enabling interprocedural analysis with:
- Function result caching for performance
- Depth limits to prevent infinite recursion
- Conservative handling of recursive functions

## Limitations

- **Static Analysis**: Cannot detect all runtime conditions (e.g., values from external services)
- **Conservative**: May report false positives in complex control flow scenarios
- **Depth Limited**: Interprocedural analysis has depth limits for performance
- **Go Only**: Does not analyze .proto files directly, only generated Go code

## Contributing

Contributions are welcome! Please see the implementation plan for current status and upcoming features.

## Development

### Running Tests

```bash
# Unit tests
go test ./pkg/...

# Integration tests
go test ./testdata/...

# All tests with coverage
go test -cover ./...
```

### Building

```bash
# Build CLI tool
go build -o grpc-nil-linter ./cmd/grpc-nil-linter

# Install locally
go install ./cmd/grpc-nil-linter
```

## License

MIT License - see LICENSE file for details

## Related Tools

- [`go/analysis`](https://pkg.go.dev/golang.org/x/tools/go/analysis): The analysis framework this tool is built on
- [`go/ssa`](https://pkg.go.dev/golang.org/x/tools/go/ssa): SSA representation of Go programs
- [golangci-lint](https://golangci-lint.run/): Can integrate this linter as a plugin

## References

- [Go SSA Package Documentation](https://pkg.go.dev/golang.org/x/tools/go/ssa)
- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)
- [Protocol Buffers Guide](https://protobuf.dev/)