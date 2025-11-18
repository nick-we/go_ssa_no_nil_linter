package analyzer

import (
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ssa"
)

// analyzeHandler performs direct-field SSA analysis for a single gRPC handler.
// It looks for assignments to risky response fields and reports if the assigned
// value may be nil according to NilFlowAnalyzer.
func analyzeHandler(pass *analysis.Pass, protoAnalyzer *ProtoFieldAnalyzer, nilAnalyzer *NilFlowAnalyzer, h HandlerInfo) {
	if h.Function == nil {
		return
	}

	// Determine the concrete response message type (strip pointer if needed).
	respType := h.ResponseType
	if ptr, ok := respType.(*types.Pointer); ok {
		respType = ptr.Elem()
	}
	respNamed, ok := respType.(*types.Named)
	if !ok {
		return
	}

	msgInfo := protoAnalyzer.AnalyzeMessage(respNamed)
	if msgInfo == nil || len(msgInfo.Risky) == 0 {
		// No risky fields => nothing to check.
		return
	}

	// NOTE: For now we conservatively treat any store whose base address has
	// type *Resp as a response field assignment, without restricting to
	// specific alloc sites. This is sufficient for unit tests and keeps the
	// analysis simple; it can be refined later to track specific response
	// instances.

	// For each instruction, look for stores to response fields.
	for _, b := range h.Function.Blocks {
		for _, instr := range b.Instrs {
			store, ok := instr.(*ssa.Store)
			if !ok {
				continue
			}
			fieldAddr, ok := store.Addr.(*ssa.FieldAddr)
			if !ok {
				continue
			}

			// Ensure the base address is a pointer to the response type.
			if !isResponsePointer(fieldAddr.X.Type(), respNamed) {
				continue
			}

			// Map field index to FieldInfo.
			fieldInfo, ok := msgInfo.FieldByID[fieldAddr.Field]
			if !ok {
				continue
			}
			if fieldInfo.Risk == FieldRiskSafe {
				continue
			}

			// Check the value being stored for potential nil.
			nilAnalyzer.Reset()
			if !nilAnalyzer.IsMaybeNil(store.Val) {
				continue
			}

			// Report diagnostic.
			pass.Reportf(
				store.Pos(),
				"potential nil field in gRPC response %s.%s (handler %s.%s)",
				respNamed.Obj().Name(),
				fieldInfo.Name,
				h.ServiceName,
				h.MethodName,
			)
		}
	}
}

// isResponsePointer reports whether t is *respNamed.
func isResponsePointer(t types.Type, respNamed *types.Named) bool {
	if respNamed == nil || t == nil {
		return false
	}
	ptrType, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	elemNamed, ok := ptrType.Elem().(*types.Named)
	if !ok {
		return false
	}
	return types.Identical(elemNamed, respNamed)
}

// resolveAlloc attempts to recover the underlying *ssa.Alloc for v,
// following simple phi/unop edges. It is intentionally conservative.
func resolveAlloc(v ssa.Value, seen map[ssa.Value]bool) *ssa.Alloc {
	if v == nil {
		return nil
	}
	if seen == nil {
		seen = make(map[ssa.Value]bool)
	}
	if seen[v] {
		return nil
	}
	seen[v] = true

	switch val := v.(type) {
	case *ssa.Alloc:
		return val
	case *ssa.Phi:
		for _, e := range val.Edges {
			if a := resolveAlloc(e, seen); a != nil {
				return a
			}
		}
	case *ssa.UnOp:
		// &x or *x style operations; follow the operand.
		return resolveAlloc(val.X, seen)
	case *ssa.ChangeInterface:
		return resolveAlloc(val.X, seen)
	case *ssa.MakeInterface:
		return resolveAlloc(val.X, seen)
	}

	return nil
}
