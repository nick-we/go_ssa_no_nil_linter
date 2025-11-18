package subnil

import (
	"context"
)

// GetUserRequest is a minimal proto-like request message.
type GetUserRequest struct{}

// ProtoMessage marks GetUserRequest as a proto message.
func (*GetUserRequest) ProtoMessage() {}

// GetUserResponse is a proto-like response message with a risky sub-message field.
type GetUserResponse struct {
	Profile *UserProfile `protobuf:"bytes,1,opt,name=profile,proto3"`
}

// ProtoMessage marks GetUserResponse as a proto message.
func (*GetUserResponse) ProtoMessage() {}

// UserProfile is a nested sub-message type.
type UserProfile struct{}

// ProtoMessage marks UserProfile as a proto message.
func (*UserProfile) ProtoMessage() {}

// Service is a minimal gRPC-like service implementation.
type Service struct{}

// GetUserImplicit leaves the non-optional Profile field completely unset,
// which should be treated as an implicit nil assignment in the response.
func (s *Service) GetUserImplicit(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	resp := &GetUserResponse{}
	// Profile is never assigned anywhere in this handler.
	return resp, nil // want "implicit nil field in gRPC response GetUserResponse.Profile"
}

// GetUserExplicit assigns an explicit nil value to the non-optional Profile
// field, which should also be flagged by the analyzer.
func (s *Service) GetUserExplicit(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	resp := &GetUserResponse{}
	resp.Profile = nil // want "potential nil field in gRPC response GetUserResponse.Profile"
	return resp, nil
}
