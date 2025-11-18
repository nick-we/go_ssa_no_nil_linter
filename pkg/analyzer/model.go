package analyzer

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// NilStatus describes whether a value is guaranteed non-nil, definitely nil, or unknown.
type NilStatus int

const (
	NilStatusUnknown NilStatus = iota
	NilStatusNotNil
	NilStatusMaybeNil
	NilStatusDefinitelyNil
)

// FieldRisk identifies how "risky" it is for a field to be nil in a response.
type FieldRisk int

const (
	FieldRiskSafe FieldRisk = iota
	FieldRiskMessagePointer
	FieldRiskRepeatedMessagePointer
	FieldRiskImplicitRequirement
)

// FieldInfo captures proto field metadata derived from generated Go structs.
type FieldInfo struct {
	Name            string
	Parent          *types.Named
	Type            types.Type
	Tag             string
	IsPointer       bool
	IsRepeated      bool
	IsMap           bool
	IsScalar        bool
	IsOptional      bool
	IsProtoMessage  bool
	MessageTypeName string
	Risk            FieldRisk
}

// ProtoMessageInfo represents analysis results for a proto-generated message type.
type ProtoMessageInfo struct {
	Type      *types.Named
	Fields    []FieldInfo
	Risky     []FieldInfo
	FieldByID map[int]FieldInfo
}

// HandlerInfo tracks gRPC handler metadata discovered in the SSA program.
type HandlerInfo struct {
	Function     *ssa.Function
	ReceiverType *types.Named
	RequestType  types.Type
	ResponseType types.Type
	ServiceName  string
	MethodName   string
}

// TraceStep represents one instruction/edge in a nil-flow trace used for diagnostics.
type TraceStep struct {
	Instruction ssa.Instruction
	Position    token.Position
	Message     string
}

// Issue is the primary diagnostic produced by the analyzer.
type Issue struct {
	Handler    HandlerInfo
	Field      FieldInfo
	Status     NilStatus
	Trace      []TraceStep
	Suggestion string
}
