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

	// For each instruction, look for stores to response fields or slice elements.
	for _, b := range h.Function.Blocks {
		for _, instr := range b.Instrs {
			store, ok := instr.(*ssa.Store)
			if !ok {
				continue
			}

			switch addr := store.Addr.(type) {
			case *ssa.FieldAddr:
				// Direct struct field assignment, e.g. resp.Profile = v.
				if !isResponsePointer(addr.X.Type(), respNamed) {
					continue
				}

				// Map field index to FieldInfo.
				fieldInfo, ok := msgInfo.FieldByID[addr.Field]
				if !ok {
					continue
				}
				// Only scalar message-pointer fields are treated as direct-field risks.
				if fieldInfo.Risk != FieldRiskMessagePointer {
					continue
				}

				// Check the value being stored for potential nil.
				nilAnalyzer.Reset()
				if !nilAnalyzer.IsMaybeNil(store.Val) {
					continue
				}

				// Report diagnostic for direct field.
				pass.Reportf(
					store.Pos(),
					"potential nil field in gRPC response %s.%s (handler %s.%s)",
					respNamed.Obj().Name(),
					fieldInfo.Name,
					h.ServiceName,
					h.MethodName,
				)

			case *ssa.IndexAddr:
				// Slice/array element assignment, e.g. resp.Users[i] = v.
				// We conservatively match based on the element container type:
				// if the slice type matches a repeated message field on the response,
				// we treat this as a potential nil element assignment.
				fieldInfo, ok := matchRepeatedSliceField(addr.X.Type(), msgInfo)
				if !ok || fieldInfo.Risk != FieldRiskRepeatedMessagePointer {
					continue
				}

				// Check the value being stored for potential nil.
				nilAnalyzer.Reset()
				if !nilAnalyzer.IsMaybeNil(store.Val) {
					continue
				}

				// Report diagnostic for slice element.
				pass.Reportf(
					store.Pos(),
					"potential nil element in gRPC response slice %s (handler %s.%s)",
					fieldInfo.Name,
					h.ServiceName,
					h.MethodName,
				)
			}
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

// matchRepeatedSliceField tries to find a repeated-message field on the
// response whose Go type matches the provided slice/array type.
func matchRepeatedSliceField(t types.Type, msgInfo *ProtoMessageInfo) (FieldInfo, bool) {
	if msgInfo == nil || t == nil {
		return FieldInfo{}, false
	}
	for _, fi := range msgInfo.Fields {
		if fi.Risk != FieldRiskRepeatedMessagePointer {
			continue
		}
		if types.Identical(fi.Type, t) {
			return fi, true
		}
	}
	return FieldInfo{}, false
}
