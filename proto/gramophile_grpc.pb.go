// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.12.4
// source: gramophile.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// QueueServiceClient is the client API for QueueService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type QueueServiceClient interface {
	Execute(ctx context.Context, in *ExecuteRequest, opts ...grpc.CallOption) (*ExecuteResponse, error)
}

type queueServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewQueueServiceClient(cc grpc.ClientConnInterface) QueueServiceClient {
	return &queueServiceClient{cc}
}

func (c *queueServiceClient) Execute(ctx context.Context, in *ExecuteRequest, opts ...grpc.CallOption) (*ExecuteResponse, error) {
	out := new(ExecuteResponse)
	err := c.cc.Invoke(ctx, "/gramophile.QueueService/Execute", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// QueueServiceServer is the server API for QueueService service.
// All implementations should embed UnimplementedQueueServiceServer
// for forward compatibility
type QueueServiceServer interface {
	Execute(context.Context, *ExecuteRequest) (*ExecuteResponse, error)
}

// UnimplementedQueueServiceServer should be embedded to have forward compatible implementations.
type UnimplementedQueueServiceServer struct {
}

func (UnimplementedQueueServiceServer) Execute(context.Context, *ExecuteRequest) (*ExecuteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Execute not implemented")
}

// UnsafeQueueServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to QueueServiceServer will
// result in compilation errors.
type UnsafeQueueServiceServer interface {
	mustEmbedUnimplementedQueueServiceServer()
}

func RegisterQueueServiceServer(s grpc.ServiceRegistrar, srv QueueServiceServer) {
	s.RegisterService(&QueueService_ServiceDesc, srv)
}

func _QueueService_Execute_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ExecuteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueueServiceServer).Execute(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.QueueService/Execute",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueueServiceServer).Execute(ctx, req.(*ExecuteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// QueueService_ServiceDesc is the grpc.ServiceDesc for QueueService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var QueueService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "gramophile.QueueService",
	HandlerType: (*QueueServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Execute",
			Handler:    _QueueService_Execute_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "gramophile.proto",
}

// GramophileEServiceClient is the client API for GramophileEService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type GramophileEServiceClient interface {
	GetURL(ctx context.Context, in *GetURLRequest, opts ...grpc.CallOption) (*GetURLResponse, error)
	GetLogin(ctx context.Context, in *GetLoginRequest, opts ...grpc.CallOption) (*GetLoginResponse, error)
	GetUser(ctx context.Context, in *GetUserRequest, opts ...grpc.CallOption) (*GetUserResponse, error)
}

type gramophileEServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewGramophileEServiceClient(cc grpc.ClientConnInterface) GramophileEServiceClient {
	return &gramophileEServiceClient{cc}
}

func (c *gramophileEServiceClient) GetURL(ctx context.Context, in *GetURLRequest, opts ...grpc.CallOption) (*GetURLResponse, error) {
	out := new(GetURLResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileEService/GetURL", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gramophileEServiceClient) GetLogin(ctx context.Context, in *GetLoginRequest, opts ...grpc.CallOption) (*GetLoginResponse, error) {
	out := new(GetLoginResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileEService/GetLogin", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gramophileEServiceClient) GetUser(ctx context.Context, in *GetUserRequest, opts ...grpc.CallOption) (*GetUserResponse, error) {
	out := new(GetUserResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileEService/GetUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GramophileEServiceServer is the server API for GramophileEService service.
// All implementations should embed UnimplementedGramophileEServiceServer
// for forward compatibility
type GramophileEServiceServer interface {
	GetURL(context.Context, *GetURLRequest) (*GetURLResponse, error)
	GetLogin(context.Context, *GetLoginRequest) (*GetLoginResponse, error)
	GetUser(context.Context, *GetUserRequest) (*GetUserResponse, error)
}

// UnimplementedGramophileEServiceServer should be embedded to have forward compatible implementations.
type UnimplementedGramophileEServiceServer struct {
}

func (UnimplementedGramophileEServiceServer) GetURL(context.Context, *GetURLRequest) (*GetURLResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetURL not implemented")
}
func (UnimplementedGramophileEServiceServer) GetLogin(context.Context, *GetLoginRequest) (*GetLoginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLogin not implemented")
}
func (UnimplementedGramophileEServiceServer) GetUser(context.Context, *GetUserRequest) (*GetUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUser not implemented")
}

// UnsafeGramophileEServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to GramophileEServiceServer will
// result in compilation errors.
type UnsafeGramophileEServiceServer interface {
	mustEmbedUnimplementedGramophileEServiceServer()
}

func RegisterGramophileEServiceServer(s grpc.ServiceRegistrar, srv GramophileEServiceServer) {
	s.RegisterService(&GramophileEService_ServiceDesc, srv)
}

func _GramophileEService_GetURL_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetURLRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileEServiceServer).GetURL(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileEService/GetURL",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileEServiceServer).GetURL(ctx, req.(*GetURLRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GramophileEService_GetLogin_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetLoginRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileEServiceServer).GetLogin(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileEService/GetLogin",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileEServiceServer).GetLogin(ctx, req.(*GetLoginRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GramophileEService_GetUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileEServiceServer).GetUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileEService/GetUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileEServiceServer).GetUser(ctx, req.(*GetUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// GramophileEService_ServiceDesc is the grpc.ServiceDesc for GramophileEService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var GramophileEService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "gramophile.GramophileEService",
	HandlerType: (*GramophileEServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetURL",
			Handler:    _GramophileEService_GetURL_Handler,
		},
		{
			MethodName: "GetLogin",
			Handler:    _GramophileEService_GetLogin_Handler,
		},
		{
			MethodName: "GetUser",
			Handler:    _GramophileEService_GetUser_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "gramophile.proto",
}

// GramophileServiceClient is the client API for GramophileService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type GramophileServiceClient interface {
	GetUsers(ctx context.Context, in *GetUsersRequest, opts ...grpc.CallOption) (*GetUsersResponse, error)
	GetUser(ctx context.Context, in *GetUserRequest, opts ...grpc.CallOption) (*GetUserResponse, error)
}

type gramophileServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewGramophileServiceClient(cc grpc.ClientConnInterface) GramophileServiceClient {
	return &gramophileServiceClient{cc}
}

func (c *gramophileServiceClient) GetUsers(ctx context.Context, in *GetUsersRequest, opts ...grpc.CallOption) (*GetUsersResponse, error) {
	out := new(GetUsersResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileService/GetUsers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gramophileServiceClient) GetUser(ctx context.Context, in *GetUserRequest, opts ...grpc.CallOption) (*GetUserResponse, error) {
	out := new(GetUserResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileService/GetUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GramophileServiceServer is the server API for GramophileService service.
// All implementations should embed UnimplementedGramophileServiceServer
// for forward compatibility
type GramophileServiceServer interface {
	GetUsers(context.Context, *GetUsersRequest) (*GetUsersResponse, error)
	GetUser(context.Context, *GetUserRequest) (*GetUserResponse, error)
}

// UnimplementedGramophileServiceServer should be embedded to have forward compatible implementations.
type UnimplementedGramophileServiceServer struct {
}

func (UnimplementedGramophileServiceServer) GetUsers(context.Context, *GetUsersRequest) (*GetUsersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUsers not implemented")
}
func (UnimplementedGramophileServiceServer) GetUser(context.Context, *GetUserRequest) (*GetUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUser not implemented")
}

// UnsafeGramophileServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to GramophileServiceServer will
// result in compilation errors.
type UnsafeGramophileServiceServer interface {
	mustEmbedUnimplementedGramophileServiceServer()
}

func RegisterGramophileServiceServer(s grpc.ServiceRegistrar, srv GramophileServiceServer) {
	s.RegisterService(&GramophileService_ServiceDesc, srv)
}

func _GramophileService_GetUsers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetUsersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileServiceServer).GetUsers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileService/GetUsers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileServiceServer).GetUsers(ctx, req.(*GetUsersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GramophileService_GetUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileServiceServer).GetUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileService/GetUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileServiceServer).GetUser(ctx, req.(*GetUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// GramophileService_ServiceDesc is the grpc.ServiceDesc for GramophileService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var GramophileService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "gramophile.GramophileService",
	HandlerType: (*GramophileServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetUsers",
			Handler:    _GramophileService_GetUsers_Handler,
		},
		{
			MethodName: "GetUser",
			Handler:    _GramophileService_GetUser_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "gramophile.proto",
}
