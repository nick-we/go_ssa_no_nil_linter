# gRPC Nil-Value Linter Implementation Plan

## Project Structure

```
go_ssa_no_nil_linter/
├── cmd/
│   └── grpc-nil-linter/
│       └── main.go                    # CLI entry point
├── pkg/
│   ├── analyzer/
│   │   ├── analyzer.go               # Main analyzer implementation
│   │   ├── grpc_detector.go          # gRPC handler detection
│   │   ├── proto_field.go            # Proto field analysis
│   │   ├── nil_flow.go               # SSA-based nil flow analysis
│   │   └── callgraph.go              # Call graph traversal
│   └── report/
│       ├── reporter.go               # Issue reporting
│       └── format.go                 # Output formatting
├── testdata/
│   ├── fixtures/
│   │   ├── proto/
│   │   │   └── user.proto            # Sample proto definitions
│   │   └── service/
│   │       └── user_service.go       # Sample gRPC service
│   └── expected/
│       └── issues.json               # Expected linter findings
├── go.mod
├── go.sum
├── README.md
├── ARCHITECTURE.md
└── IMPLEMENTATION_PLAN.md
```

## Component Breakdown

### 1. Proto Field Analyzer

Analyzes proto-generated Go structs to identify risky fields that could be nil.

**Key Responsibilities**:
- Parse struct field tags looking for `oneof` markers
- Identify pointer fields that represent sub-messages
- Distinguish safe types (scalars, maps, slices) from risky ones (message pointers)
- Build and cache a registry of proto message types

**Core Logic**:
```go
type ProtoFieldAnalyzer struct {
    typeCache map[*types.Named]*ProtoMessageInfo
}

// Returns list of fields that must not be nil
func (a *ProtoFieldAnalyzer) GetRiskyFields(typ types.Type) []FieldInfo
```

### 2. gRPC Handler Detector

Identifies methods that implement gRPC service handlers.

**Detection Criteria**:
- Signature: `(ctx context.Context, req *XxxRequest) (*XxxResponse, error)`
- Receiver implements gRPC service interface
- Return types are proto-generated messages

**Core Logic**:
```go
type GRPCDetector struct {
    program *ssa.Program
}

// Returns all gRPC handlers found in package
func (d *GRPCDetector) DetectHandlers(pkg *ssa.Package) []HandlerInfo
```

### 3. Nil Flow Analyzer

Uses SSA to trace data flow and detect possible nil values.

**Analysis Strategy**:
- Find all Store instructions to response message fields
- Trace values backward through SSA instructions
- Handle different instruction types (Alloc, Call, Phi, etc.)
- Build human-readable trace chains

**Core Logic**:
```go
type NilFlowAnalyzer struct {
    program   *ssa.Program
    callGraph *callgraph.Graph
    visited   map[ssa.Value]NilStatus
}

// Returns nil status for a value
func (a *NilFlowAnalyzer) TraceValueNilStatus(val ssa.Value) NilStatus
```

**SSA Instructions to Handle**:
- `*ssa.Const`: Check if nil constant
- `*ssa.Alloc`: Always not-nil (new allocation)
- `*ssa.Call`: Recursively analyze callee function
- `*ssa.Phi`: Check all incoming control flow edges
- `*ssa.FieldAddr`, `*ssa.Field`: Trace base object
- `*ssa.Store`: Track what gets stored where

### 4. Call Graph Analyzer

Enables interprocedural analysis across function boundaries.

**Strategy**:
- Build call graph using RTA (Rapid Type Analysis)
- Cache function analysis results
- Handle recursive calls with depth limits
- Propagate nil status through function calls

**Core Logic**:
```go
type CallGraphAnalyzer struct {
    graph    *callgraph.Graph
    maxDepth int
    cache    map[*ssa.Function]*FunctionSummary
}

// Analyzes function and returns summary
func (a *CallGraphAnalyzer) AnalyzeFunction(fn *ssa.Function) *FunctionSummary
```

### 5. Main Analyzer

Orchestrates all components and integrates with the analysis framework.

**Workflow**:
1. Load packages and build SSA
2. Construct call graph
3. Detect gRPC handlers
4. For each handler:
   - Get response type and risky fields
   - Trace nil flow for each field
   - Build issue reports
5. Report findings

**Core Logic**:
```go
type Analyzer struct {
    protoAnalyzer  *ProtoFieldAnalyzer
    grpcDetector   *GRPCDetector
    nilAnalyzer    *NilFlowAnalyzer
    callGraph      *CallGraphAnalyzer
}

// Main entry point for analysis
func (a *Analyzer) Run(pass *analysis.Pass) (interface{}, error)
```

## Implementation Phases

### Phase 1: Foundation
- Set up Go module with dependencies
- Create basic project structure  
- Implement proto field analyzer
- Write unit tests

### Phase 2: Handler Detection
- Implement gRPC handler detector
- Create test fixtures
- Test with various gRPC patterns

### Phase 3: SSA Analysis
- Implement basic nil tracing
- Handle common SSA instructions
- Add control flow support
- Comprehensive testing

### Phase 4: Interprocedural Analysis
- Integrate call graph
- Implement function caching
- Handle recursive calls
- Performance optimization

### Phase 5: Integration
- Connect all components
- Implement reporting
- Create CLI tool
- Add output formatting

### Phase 6: Testing & Documentation
- Integration tests
- Real-world testing
- Documentation
- Examples

## Key Algorithms

### Field Classification Algorithm

```
For each struct field:
  1. If not a pointer type -> SAFE (scalars don't cause nil panics)
  2. If has "oneof" tag -> SAFE (explicitly optional)
  3. If slice or map type -> SAFE (nil is valid)
  4. If proto message type -> RISKY (must not be nil)
  5. Otherwise -> SAFE
```

### Nil Tracing Algorithm

```
Function TraceNil(value):
  If value in visited cache:
    Return cached result
  
  Switch on value type:
    Case Const:
      Return NIL if nil constant, else NOT_NIL
    
    Case Alloc:
      Return NOT_NIL (new allocation)
    
    Case Call:
      Analyze callee function recursively
      Return callee's return nil status
    
    Case Phi:
      For each incoming edge:
        If edge could be NIL:
          Return MAYBE_NIL
      Return NOT_NIL
    
    Case FieldAddr/Field:
      Trace base object
    
    Default:
      Return UNKNOWN
```

## CLI Usage

```bash
# Analyze current package
grpc-nil-linter ./...

# JSON output
grpc-nil-linter -format json ./...

# Verbose output
grpc-nil-linter -v ./...
```

## Testing Strategy

**Unit Tests**:
- Field classification edge cases
- Handler detection patterns
- SSA instruction handling
- Nil status merging

**Integration Tests**:
- Real gRPC services
- Various nil flow patterns
- Complex call graphs
- Edge cases

**Test Fixtures**:
- Simple direct assignments
- Helper function calls
- Conditional logic
- Loops and control flow
- Nested function calls

## Success Criteria

- Detect 90%+ of real nil issues
- False positive rate < 15%
- Analyze 1000 handlers in < 10 seconds
- Clear, actionable error messages
- Easy integration with existing tools