package analyzer

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

// NilFlowAnalyzer performs a lightweight, conservative nil-flow analysis over SSA values.
type NilFlowAnalyzer struct {
	visited map[ssa.Value]NilStatus
}

// NewNilFlowAnalyzer constructs a new NilFlowAnalyzer.
func NewNilFlowAnalyzer() *NilFlowAnalyzer {
	return &NilFlowAnalyzer{
		visited: make(map[ssa.Value]NilStatus),
	}
}

// Reset clears internal caches between analyses.
func (a *NilFlowAnalyzer) Reset() {
	for k := range a.visited {
		delete(a.visited, k)
	}
}

// ValueNilStatus computes a conservative nil-status for v.
func (a *NilFlowAnalyzer) ValueNilStatus(v ssa.Value) NilStatus {
	if v == nil {
		return NilStatusUnknown
	}
	if s, ok := a.visited[v]; ok {
		return s
	}
	// Mark as unknown to break cycles.
	a.visited[v] = NilStatusUnknown

	switch val := v.(type) {
	case *ssa.Const:
		if val.IsNil() {
			a.visited[v] = NilStatusDefinitelyNil
		} else {
			a.visited[v] = NilStatusNotNil
		}
	case *ssa.Alloc:
		// New allocations are never nil.
		a.visited[v] = NilStatusNotNil
	case *ssa.MakeInterface:
		a.visited[v] = a.ValueNilStatus(val.X)
	case *ssa.ChangeInterface:
		a.visited[v] = a.ValueNilStatus(val.X)
	case *ssa.Phi:
		status := NilStatusNotNil
		for _, edge := range val.Edges {
			edgeStatus := a.ValueNilStatus(edge)
			status = joinNilStatus(status, edgeStatus)
			if status == NilStatusMaybeNil || status == NilStatusDefinitelyNil {
				break
			}
		}
		a.visited[v] = status
	case *ssa.UnOp:
		// For *ptr, propagate ptr's nil status.
		if val.Op == token.MUL {
			a.visited[v] = a.ValueNilStatus(val.X)
		}
	default:
		// Unknown instruction kinds are treated as unknown.
		a.visited[v] = NilStatusUnknown
	}

	return a.visited[v]
}

// joinNilStatus merges two NilStatus values conservatively.
func joinNilStatus(a, b NilStatus) NilStatus {
	switch {
	case a == NilStatusUnknown || b == NilStatusUnknown:
		if a == NilStatusDefinitelyNil || b == NilStatusDefinitelyNil {
			return NilStatusMaybeNil
		}
		return NilStatusUnknown
	case a == NilStatusDefinitelyNil && b == NilStatusDefinitelyNil:
		return NilStatusDefinitelyNil
	case a == NilStatusNotNil && b == NilStatusNotNil:
		return NilStatusNotNil
	default:
		return NilStatusMaybeNil
	}
}

// IsMaybeNil reports whether v could be nil (including unknown cases).
func (a *NilFlowAnalyzer) IsMaybeNil(v ssa.Value) bool {
	s := a.ValueNilStatus(v)
	return s == NilStatusMaybeNil || s == NilStatusDefinitelyNil || s == NilStatusUnknown
}
