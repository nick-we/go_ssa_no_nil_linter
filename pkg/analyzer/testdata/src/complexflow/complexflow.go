package complexflow

import "context"

// Request / Response / Sub-message types

type GetUserRequest struct{}

func (*GetUserRequest) ProtoMessage() {}

type UserProfile struct{}

func (*UserProfile) ProtoMessage() {}

type GetUserResponse struct {
	Profile *UserProfile `protobuf:"bytes,1,opt,name=profile,proto3"`
}

func (*GetUserResponse) ProtoMessage() {}

// Helpers

func cond() bool { return true }

// buildProfileNonNil always returns a non-nil profile.
func buildProfileNonNil() *UserProfile {
	return &UserProfile{}
}

// buildProfileMaybeNil may return nil based on a condition.
func buildProfileMaybeNil() *UserProfile {
	if cond() {
		return &UserProfile{}
	}
	return nil
}

// Service is a minimal gRPC-like service implementation.
type Service struct{}

// 1) If/else flows

// GetUserIfElseSafe assigns a non-nil profile in both branches, so the
// Profile field is never nil at the return site and should not be flagged.
func (s *Service) GetUserIfElseSafe(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	resp := &GetUserResponse{}
	var profile *UserProfile
	if cond() {
		profile = &UserProfile{}
	} else {
		profile = &UserProfile{}
	}
	resp.Profile = profile
	return resp, nil
}

// GetUserIfElseMaybe assigns nil in one branch and a non-nil profile in the
// other; the resulting Profile field may be nil and should be flagged.
func (s *Service) GetUserIfElseMaybe(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	resp := &GetUserResponse{}
	var profile *UserProfile
	if cond() {
		profile = &UserProfile{}
	} else {
		profile = nil
	}
	resp.Profile = profile // want "potential nil field in gRPC response GetUserResponse.Profile"
	return resp, nil
}

// 2) Switch flows

// GetUserSwitchSafe assigns non-nil profiles in all cases; Profile should not
// be considered nil at the return site.
func (s *Service) GetUserSwitchSafe(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	resp := &GetUserResponse{}
	var profile *UserProfile
	switch {
	case cond():
		profile = &UserProfile{}
	default:
		profile = &UserProfile{}
	}
	resp.Profile = profile
	return resp, nil
}

// GetUserSwitchMaybe has one switch arm that assigns nil; Profile may be nil
// and should be flagged.
func (s *Service) GetUserSwitchMaybe(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	resp := &GetUserResponse{}
	var profile *UserProfile
	switch {
	case cond():
		profile = &UserProfile{}
	default:
		profile = nil
	}
	resp.Profile = profile // want "potential nil field in gRPC response GetUserResponse.Profile"
	return resp, nil
}

// 3) Sub-function calls

// GetUserSubcallSafe calls a helper that always returns a non-nil profile;
// Profile should not be considered nil at the return site.
func (s *Service) GetUserSubcallSafe(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	resp := &GetUserResponse{}
	resp.Profile = buildProfileNonNil()
	return resp, nil
}

// GetUserSubcallMaybe calls a helper that may return nil; Profile may be nil
// and should be flagged.
func (s *Service) GetUserSubcallMaybe(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	resp := &GetUserResponse{}
	resp.Profile = buildProfileMaybeNil() // want "potential nil field in gRPC response GetUserResponse.Profile"
	return resp, nil
}
