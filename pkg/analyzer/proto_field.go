package analyzer

import (
	"go/types"
	"reflect"
	"strings"
)

// ProtoFieldAnalyzer inspects proto-generated Go structs and identifies risky fields.
type ProtoFieldAnalyzer struct {
	cache map[*types.Named]*ProtoMessageInfo
}

func NewProtoFieldAnalyzer() *ProtoFieldAnalyzer {
	return &ProtoFieldAnalyzer{
		cache: make(map[*types.Named]*ProtoMessageInfo),
	}
}

// AnalyzeMessage extracts metadata for a proto-generated message type.
func (p *ProtoFieldAnalyzer) AnalyzeMessage(named *types.Named) *ProtoMessageInfo {
	if named == nil {
		return nil
	}
	if info, ok := p.cache[named]; ok {
		return info
	}

	info := &ProtoMessageInfo{
		Type:      named,
		FieldByID: make(map[int]FieldInfo),
	}
	p.cache[named] = info // prevent recursion loops

	structType, ok := named.Underlying().(*types.Struct)
	if !ok {
		return info
	}

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		if !field.Exported() {
			continue
		}

		tag := structType.Tag(i)
		meta := p.classifyField(named, field, tag)
		info.Fields = append(info.Fields, meta)
		info.FieldByID[i] = meta
		if meta.Risk != FieldRiskSafe {
			info.Risky = append(info.Risky, meta)
		}
	}

	return info
}

// GetRiskyFields returns the risky fields for the provided type if it is a proto message.
func (p *ProtoFieldAnalyzer) GetRiskyFields(typ types.Type) []FieldInfo {
	named, _ := typ.(*types.Named)
	if named == nil {
		return nil
	}
	info := p.AnalyzeMessage(named)
	if info == nil {
		return nil
	}
	return info.Risky
}

func (p *ProtoFieldAnalyzer) classifyField(parent *types.Named, field *types.Var, tag string) FieldInfo {
	fieldType := field.Type()
	isPointer := isPointer(fieldType)
	isRepeated := isSlice(fieldType)
	isMap := isMap(fieldType)
	isScalar := !isPointer && !isRepeated && !isMap
	isProtoMessage := isProtoMessage(fieldType)
	isOptional := hasOneOfTag(tag)

	risk := FieldRiskSafe
	switch {
	case isRepeated && elementIsProtoMessage(fieldType):
		risk = FieldRiskRepeatedMessagePointer
	case isPointer && isProtoMessage && !isOptional:
		risk = FieldRiskMessagePointer
	}

	return FieldInfo{
		Name:            field.Name(),
		Parent:          parent,
		Type:            fieldType,
		Tag:             tag,
		IsPointer:       isPointer,
		IsRepeated:      isRepeated,
		IsMap:           isMap,
		IsScalar:        isScalar,
		IsOptional:      isOptional,
		IsProtoMessage:  isProtoMessage,
		MessageTypeName: messageTypeName(fieldType),
		Risk:            risk,
	}
}

func isPointer(t types.Type) bool {
	_, ok := t.(*types.Pointer)
	return ok
}

func isSlice(t types.Type) bool {
	_, ok := t.(*types.Slice)
	return ok
}

func isMap(t types.Type) bool {
	_, ok := t.(*types.Map)
	return ok
}

func elementIsProtoMessage(t types.Type) bool {
	slice, ok := t.(*types.Slice)
	if !ok {
		return false
	}
	return isProtoMessage(slice.Elem())
}

func isProtoMessage(t types.Type) bool {
	ptr, ok := t.(*types.Pointer)
	if ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	return implementsProtoMessage(named)
}

func implementsProtoMessage(named *types.Named) bool {
	if named == nil {
		return false
	}

	// Check method set of *T so we also see pointer-receiver methods like:
	//   func (*T) ProtoMessage()
	ptr := types.NewPointer(named)
	ms := types.NewMethodSet(ptr)
	for i := 0; i < ms.Len(); i++ {
		if ms.At(i).Obj().Name() == "ProtoMessage" {
			return true
		}
	}
	return false
}

func messageTypeName(t types.Type) string {
	ptr, ok := t.(*types.Pointer)
	if ok {
		t = ptr.Elem()
	}
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Pkg().Path() + "." + named.Obj().Name()
	}
	return t.String()
}

func hasOneOfTag(tag string) bool {
	return tagHasFlag(tag, "protobuf", "oneof")
}

func tagHasFlag(tag, key, part string) bool {
	if tag == "" {
		return false
	}
	reflectTag := reflect.StructTag(tag)
	value := reflectTag.Get(key)
	if value == "" {
		return false
	}
	if part == "" {
		return true
	}
	for _, segment := range strings.Split(value, ",") {
		if segment == part {
			return true
		}
	}
	return false
}
