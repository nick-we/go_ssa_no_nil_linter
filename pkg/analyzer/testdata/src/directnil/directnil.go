package directnil

import (
	"context"
	"time"
)

// GetUserRequest is a minimal proto-like request message.
type GetUserRequest struct{}

// ProtoMessage marks GetUserRequest as a proto message for the analyzer.
func (*GetUserRequest) ProtoMessage() {}

// GetUserResponse is a proto-like response message with a risky field:
// Profile is a non-optional pointer to a sub-message.
type GetUserResponse struct {
	Profile *UserProfile `protobuf:"bytes,1,opt,name=profile,proto3"`
}

// ProtoMessage marks GetUserResponse as a proto message for the analyzer.
func (*GetUserResponse) ProtoMessage() {}

// UserProfile is a nested sub-message type.
type UserProfile struct{}

// ProtoMessage marks UserProfile as a proto message for the analyzer.
func (*UserProfile) ProtoMessage() {}

// maybeProfile returns a value that may be nil, modeled via simple control flow.
func maybeProfile() *UserProfile {
	if time.Now().Unix()%2 == 0 {
		return &UserProfile{}
	}
	return nil
}

// Service is a minimal gRPC-like service implementation.
type Service struct{}

// GetUser is a unary gRPC-style handler the analyzer should detect.
// It assigns a maybe-nil value into a non-optional response field.
func (s *Service) GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	var profile *UserProfile
	if ctx != nil {
		profile = maybeProfile()
	}

	resp := &GetUserResponse{}
	resp.Profile = profile // want "potential nil field in gRPC response GetUserResponse.Profile"
	return resp, nil
}
