package analyzer

import (
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// GRPCDetector scans SSA functions to find unary gRPC handlers.
type GRPCDetector struct {
	program *ssa.Program
}

// NewGRPCDetector builds a detector using the provided SSA program.
func NewGRPCDetector(program *ssa.Program) *GRPCDetector {
	return &GRPCDetector{program: program}
}

// DetectHandlers walks all functions in the SSA program and returns gRPC handlers.
func (d *GRPCDetector) DetectHandlers() []HandlerInfo {
	if d == nil || d.program == nil {
		return nil
	}

	var handlers []HandlerInfo
	for _, pkg := range d.program.AllPackages() {
		for _, member := range pkg.Members {
			fn, ok := member.(*ssa.Function)
			if !ok {
				continue
			}
			if handler := d.inspectFunction(fn); handler != nil {
				handlers = append(handlers, *handler)
			}
		}
	}

	return handlers
}

func (d *GRPCDetector) inspectFunction(fn *ssa.Function) *HandlerInfo {
	if fn == nil || fn.Signature == nil {
		return nil
	}

	sig := fn.Signature
	recv := sig.Recv()
	if recv == nil {
		return nil
	}

	if sig.Params().Len() < 2 || sig.Results().Len() != 2 {
		return nil
	}

	ctxParam := sig.Params().At(0)
	reqParam := sig.Params().At(1)

	if !isContextType(ctxParam.Type()) {
		return nil
	}

	respResult := sig.Results().At(0)
	errResult := sig.Results().At(1)

	if !isErrorType(errResult.Type()) {
		return nil
	}

	if !isProtoMessage(reqParam.Type()) || !isProtoMessage(respResult.Type()) {
		return nil
	}

	serviceType := receiverNamedType(recv.Type())
	if serviceType == nil {
		return nil
	}

	return &HandlerInfo{
		Function:     fn,
		ReceiverType: serviceType,
		RequestType:  reqParam.Type(),
		ResponseType: respResult.Type(),
		ServiceName:  serviceType.Obj().Name(),
		MethodName:   fn.Name(),
	}
}

// DetectHandlerFromFunc inspects a single SSA function and returns a HandlerInfo
// if it matches the unary gRPC handler shape:
//
//	func (s *Service) Method(ctx context.Context, req *Req) (*Resp, error)
//
// where Req and Resp are proto messages.
func DetectHandlerFromFunc(fn *ssa.Function) *HandlerInfo {
	if fn == nil || fn.Signature == nil {
		return nil
	}

	sig := fn.Signature
	recv := sig.Recv()
	if recv == nil {
		return nil
	}

	// Expect at least (ctx, req) parameters and exactly (resp, error) results.
	if sig.Params().Len() < 2 || sig.Results().Len() != 2 {
		return nil
	}

	ctxParam := sig.Params().At(0)
	reqParam := sig.Params().At(1)

	if !isContextType(ctxParam.Type()) {
		return nil
	}

	respResult := sig.Results().At(0)
	errResult := sig.Results().At(1)

	if !isErrorType(errResult.Type()) {
		return nil
	}

	// Both request and response must be proto messages.
	if !isProtoMessage(reqParam.Type()) || !isProtoMessage(respResult.Type()) {
		return nil
	}

	serviceType := receiverNamedType(recv.Type())
	if serviceType == nil {
		return nil
	}

	return &HandlerInfo{
		Function:     fn,
		ReceiverType: serviceType,
		RequestType:  reqParam.Type(),
		ResponseType: respResult.Type(),
		ServiceName:  serviceType.Obj().Name(),
		MethodName:   fn.Name(),
	}
}

func receiverNamedType(t types.Type) *types.Named {
	ptr, ok := t.(*types.Pointer)
	if ok {
		t = ptr.Elem()
	}
	named, _ := t.(*types.Named)
	return named
}

func isContextType(t types.Type) bool {
	named := receiverNamedType(t)
	if named == nil {
		return false
	}
	if named.Obj().Pkg() == nil {
		return false
	}
	return named.Obj().Pkg().Path() == "context" && named.Obj().Name() == "Context"
}

func isErrorType(t types.Type) bool {
	named := receiverNamedType(t)
	if named == nil {
		return false
	}
	return named.Obj().Pkg() == nil && named.Obj().Name() == "error"
}
