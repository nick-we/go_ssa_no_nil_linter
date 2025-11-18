package analyzer

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

// NilFlowAnalyzer performs a lightweight, conservative nil-flow analysis over SSA values.
type NilFlowAnalyzer struct {
	visited     map[ssa.Value]NilStatus
	funcSummary map[*ssa.Function]NilStatus
}

// NewNilFlowAnalyzer constructs a new NilFlowAnalyzer.
func NewNilFlowAnalyzer() *NilFlowAnalyzer {
	return &NilFlowAnalyzer{
		visited:     make(map[ssa.Value]NilStatus),
		funcSummary: make(map[*ssa.Function]NilStatus),
	}
}

// Reset clears internal caches between analyses.
func (a *NilFlowAnalyzer) Reset() {
	for k := range a.visited {
		delete(a.visited, k)
	}
	// funcSummary is kept across Resets; summaries are per-function and
	// can be reused across handlers.
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
	case *ssa.Call:
		a.visited[v] = a.callNilStatus(val)
	default:
		// Unknown instruction kinds are treated as unknown.
		a.visited[v] = NilStatusUnknown
	}

	return a.visited[v]
}

// callNilStatus summarizes the nil behavior of a call expression.
// For now we use a simple, conservative summary: if all return sites in the
// callee return a fresh allocation or a non-nil constant, we treat the result
// as NotNil. If any return site is nil or unknown, we treat it as MaybeNil.
func (a *NilFlowAnalyzer) callNilStatus(call *ssa.Call) NilStatus {
	if call == nil {
		return NilStatusUnknown
	}
	common := call.Common()
	if common == nil {
		return NilStatusUnknown
	}
	fn := common.StaticCallee()
	if fn == nil {
		return NilStatusUnknown
	}

	// Reuse cached summary if available.
	if s, ok := a.funcSummary[fn]; ok {
		return s
	}

	status := NilStatusNotNil

	// Inspect all return sites of the callee.
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			ret, ok := instr.(*ssa.Return)
			if !ok || len(ret.Results) == 0 {
				continue
			}

			rv := ret.Results[0]
			switch rvt := rv.(type) {
			case *ssa.Alloc:
				// Fresh allocation is non-nil.
				status = joinNilStatus(status, NilStatusNotNil)
			case *ssa.Const:
				if rvt.IsNil() {
					status = joinNilStatus(status, NilStatusDefinitelyNil)
				} else {
					status = joinNilStatus(status, NilStatusNotNil)
				}
			default:
				// For complex expressions (Phi, nested calls, etc.) we are conservative.
				status = joinNilStatus(status, NilStatusMaybeNil)
			}

			if status == NilStatusMaybeNil || status == NilStatusDefinitelyNil {
				break
			}
		}
		if status == NilStatusMaybeNil || status == NilStatusDefinitelyNil {
			break
		}
	}

	a.funcSummary[fn] = status
	return status
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
