package listnil

import (
	"context"
	"time"
)

// ListUsersRequest is a minimal proto-like request message.
type ListUsersRequest struct{}

// ProtoMessage marks ListUsersRequest as a proto message.
func (*ListUsersRequest) ProtoMessage() {}

// ListUsersResponse is a proto-like response with a repeated field of
// non-optional message pointers. Nil elements should be considered unsafe.
type ListUsersResponse struct {
	Users []*User `protobuf:"bytes,1,opt,name=users,proto3"`
}

// ProtoMessage marks ListUsersResponse as a proto message.
func (*ListUsersResponse) ProtoMessage() {}

// User is a nested sub-message type returned in the Users list.
type User struct{}

// ProtoMessage marks User as a proto message.
func (*User) ProtoMessage() {}

// maybeUser returns a value that may be nil, modeled via simple control flow.
func maybeUser() *User {
	if time.Now().Unix()%2 == 0 {
		return &User{}
	}
	return nil
}

// Service is a minimal gRPC-like service implementation.
type Service struct{}

// ListUsers is a unary gRPC-style "list" endpoint that returns a slice of
// users. It assigns a maybe-nil value into a non-optional slice element,
// which the analyzer should flag.
func (s *Service) ListUsers(ctx context.Context, req *ListUsersRequest) (*ListUsersResponse, error) {
	resp := &ListUsersResponse{
		Users: make([]*User, 1),
	}
	resp.Users[0] = maybeUser() // want "potential nil element in gRPC response slice Users"
	return resp, nil
}
