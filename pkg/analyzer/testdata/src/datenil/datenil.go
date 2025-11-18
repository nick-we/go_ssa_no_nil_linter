package datenil

import "context"

// GetEventRequest is a minimal proto-like request message.
type GetEventRequest struct{}

// ProtoMessage marks GetEventRequest as a proto message.
func (*GetEventRequest) ProtoMessage() {}

// Timestamp is a custom proto-like "date/time" message, similar to google.protobuf.Timestamp.
type Timestamp struct {
	Seconds int64
	Nanos   int32
}

// ProtoMessage marks Timestamp as a proto message.
func (*Timestamp) ProtoMessage() {}

// GetEventResponse is a proto-like response with:
//   - EventDate: non-optional sub-message (must not be nil)
//   - OptionalDate: optional sub-message via oneof (nil is allowed and should be ignored)
type GetEventResponse struct {
	EventDate    *Timestamp `protobuf:"bytes,1,opt,name=event_date,json=eventDate,proto3"`
	OptionalDate *Timestamp `protobuf:"bytes,2,opt,name=optional_date,json=optionalDate,proto3,oneof"`
}

// ProtoMessage marks GetEventResponse as a proto message.
func (*GetEventResponse) ProtoMessage() {}

// Service is a minimal gRPC-like service implementation.
type Service struct{}

// GetEventImplicit leaves the non-optional EventDate field completely unset,
// which should be treated as an implicit nil assignment in the response.
// OptionalDate is also unset but should be ignored because it is optional/oneof.
func (s *Service) GetEventImplicit(ctx context.Context, req *GetEventRequest) (*GetEventResponse, error) {
	resp := &GetEventResponse{}
	return resp, nil // want "implicit nil field in gRPC response GetEventResponse.EventDate"
}

// GetEventExplicit assigns an explicit nil value to the non-optional EventDate
// field, which should be flagged by the analyzer. OptionalDate explicitly
// receives nil but should be ignored as it is optional.
func (s *Service) GetEventExplicit(ctx context.Context, req *GetEventRequest) (*GetEventResponse, error) {
	resp := &GetEventResponse{}
	resp.EventDate = nil    // want "potential nil field in gRPC response GetEventResponse.EventDate"
	resp.OptionalDate = nil // optional, must NOT produce any diagnostic
	return resp, nil
}
