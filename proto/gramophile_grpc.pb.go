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
	Enqueue(ctx context.Context, in *EnqueueRequest, opts ...grpc.CallOption) (*EnqueueResponse, error)
	Execute(ctx context.Context, in *EnqueueRequest, opts ...grpc.CallOption) (*EnqueueResponse, error)
	List(ctx context.Context, in *ListRequest, opts ...grpc.CallOption) (*ListResponse, error)
}

type queueServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewQueueServiceClient(cc grpc.ClientConnInterface) QueueServiceClient {
	return &queueServiceClient{cc}
}

func (c *queueServiceClient) Enqueue(ctx context.Context, in *EnqueueRequest, opts ...grpc.CallOption) (*EnqueueResponse, error) {
	out := new(EnqueueResponse)
	err := c.cc.Invoke(ctx, "/gramophile.QueueService/Enqueue", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queueServiceClient) Execute(ctx context.Context, in *EnqueueRequest, opts ...grpc.CallOption) (*EnqueueResponse, error) {
	out := new(EnqueueResponse)
	err := c.cc.Invoke(ctx, "/gramophile.QueueService/Execute", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queueServiceClient) List(ctx context.Context, in *ListRequest, opts ...grpc.CallOption) (*ListResponse, error) {
	out := new(ListResponse)
	err := c.cc.Invoke(ctx, "/gramophile.QueueService/List", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// QueueServiceServer is the server API for QueueService service.
// All implementations should embed UnimplementedQueueServiceServer
// for forward compatibility
type QueueServiceServer interface {
	Enqueue(context.Context, *EnqueueRequest) (*EnqueueResponse, error)
	Execute(context.Context, *EnqueueRequest) (*EnqueueResponse, error)
	List(context.Context, *ListRequest) (*ListResponse, error)
}

// UnimplementedQueueServiceServer should be embedded to have forward compatible implementations.
type UnimplementedQueueServiceServer struct {
}

func (UnimplementedQueueServiceServer) Enqueue(context.Context, *EnqueueRequest) (*EnqueueResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Enqueue not implemented")
}
func (UnimplementedQueueServiceServer) Execute(context.Context, *EnqueueRequest) (*EnqueueResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Execute not implemented")
}
func (UnimplementedQueueServiceServer) List(context.Context, *ListRequest) (*ListResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method List not implemented")
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

func _QueueService_Enqueue_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EnqueueRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueueServiceServer).Enqueue(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.QueueService/Enqueue",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueueServiceServer).Enqueue(ctx, req.(*EnqueueRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _QueueService_Execute_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EnqueueRequest)
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
		return srv.(QueueServiceServer).Execute(ctx, req.(*EnqueueRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _QueueService_List_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueueServiceServer).List(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.QueueService/List",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueueServiceServer).List(ctx, req.(*ListRequest))
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
			MethodName: "Enqueue",
			Handler:    _QueueService_Enqueue_Handler,
		},
		{
			MethodName: "Execute",
			Handler:    _QueueService_Execute_Handler,
		},
		{
			MethodName: "List",
			Handler:    _QueueService_List_Handler,
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
	GetState(ctx context.Context, in *GetStateRequest, opts ...grpc.CallOption) (*GetStateResponse, error)
	SetConfig(ctx context.Context, in *SetConfigRequest, opts ...grpc.CallOption) (*SetConfigResponse, error)
	SetIntent(ctx context.Context, in *SetIntentRequest, opts ...grpc.CallOption) (*SetIntentResponse, error)
	GetRecord(ctx context.Context, in *GetRecordRequest, opts ...grpc.CallOption) (*GetRecordResponse, error)
	GetOrg(ctx context.Context, in *GetOrgRequest, opts ...grpc.CallOption) (*GetOrgResponse, error)
	SetOrgSnapshot(ctx context.Context, in *SetOrgSnapshotRequest, opts ...grpc.CallOption) (*SetOrgSnapshotResponse, error)
	AddWant(ctx context.Context, in *AddWantRequest, opts ...grpc.CallOption) (*AddWantResponse, error)
	GetWants(ctx context.Context, in *GetWantsRequest, opts ...grpc.CallOption) (*GetWantsResponse, error)
	AddWantlist(ctx context.Context, in *AddWantlistRequest, opts ...grpc.CallOption) (*AddWantlistResponse, error)
	GetWantlist(ctx context.Context, in *GetWantlistRequest, opts ...grpc.CallOption) (*GetWantlistResponse, error)
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

func (c *gramophileEServiceClient) GetState(ctx context.Context, in *GetStateRequest, opts ...grpc.CallOption) (*GetStateResponse, error) {
	out := new(GetStateResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileEService/GetState", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gramophileEServiceClient) SetConfig(ctx context.Context, in *SetConfigRequest, opts ...grpc.CallOption) (*SetConfigResponse, error) {
	out := new(SetConfigResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileEService/SetConfig", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gramophileEServiceClient) SetIntent(ctx context.Context, in *SetIntentRequest, opts ...grpc.CallOption) (*SetIntentResponse, error) {
	out := new(SetIntentResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileEService/SetIntent", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gramophileEServiceClient) GetRecord(ctx context.Context, in *GetRecordRequest, opts ...grpc.CallOption) (*GetRecordResponse, error) {
	out := new(GetRecordResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileEService/GetRecord", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gramophileEServiceClient) GetOrg(ctx context.Context, in *GetOrgRequest, opts ...grpc.CallOption) (*GetOrgResponse, error) {
	out := new(GetOrgResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileEService/GetOrg", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gramophileEServiceClient) SetOrgSnapshot(ctx context.Context, in *SetOrgSnapshotRequest, opts ...grpc.CallOption) (*SetOrgSnapshotResponse, error) {
	out := new(SetOrgSnapshotResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileEService/SetOrgSnapshot", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gramophileEServiceClient) AddWant(ctx context.Context, in *AddWantRequest, opts ...grpc.CallOption) (*AddWantResponse, error) {
	out := new(AddWantResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileEService/AddWant", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gramophileEServiceClient) GetWants(ctx context.Context, in *GetWantsRequest, opts ...grpc.CallOption) (*GetWantsResponse, error) {
	out := new(GetWantsResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileEService/GetWants", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gramophileEServiceClient) AddWantlist(ctx context.Context, in *AddWantlistRequest, opts ...grpc.CallOption) (*AddWantlistResponse, error) {
	out := new(AddWantlistResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileEService/AddWantlist", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gramophileEServiceClient) GetWantlist(ctx context.Context, in *GetWantlistRequest, opts ...grpc.CallOption) (*GetWantlistResponse, error) {
	out := new(GetWantlistResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileEService/GetWantlist", in, out, opts...)
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
	GetState(context.Context, *GetStateRequest) (*GetStateResponse, error)
	SetConfig(context.Context, *SetConfigRequest) (*SetConfigResponse, error)
	SetIntent(context.Context, *SetIntentRequest) (*SetIntentResponse, error)
	GetRecord(context.Context, *GetRecordRequest) (*GetRecordResponse, error)
	GetOrg(context.Context, *GetOrgRequest) (*GetOrgResponse, error)
	SetOrgSnapshot(context.Context, *SetOrgSnapshotRequest) (*SetOrgSnapshotResponse, error)
	AddWant(context.Context, *AddWantRequest) (*AddWantResponse, error)
	GetWants(context.Context, *GetWantsRequest) (*GetWantsResponse, error)
	AddWantlist(context.Context, *AddWantlistRequest) (*AddWantlistResponse, error)
	GetWantlist(context.Context, *GetWantlistRequest) (*GetWantlistResponse, error)
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
func (UnimplementedGramophileEServiceServer) GetState(context.Context, *GetStateRequest) (*GetStateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetState not implemented")
}
func (UnimplementedGramophileEServiceServer) SetConfig(context.Context, *SetConfigRequest) (*SetConfigResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetConfig not implemented")
}
func (UnimplementedGramophileEServiceServer) SetIntent(context.Context, *SetIntentRequest) (*SetIntentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetIntent not implemented")
}
func (UnimplementedGramophileEServiceServer) GetRecord(context.Context, *GetRecordRequest) (*GetRecordResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRecord not implemented")
}
func (UnimplementedGramophileEServiceServer) GetOrg(context.Context, *GetOrgRequest) (*GetOrgResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetOrg not implemented")
}
func (UnimplementedGramophileEServiceServer) SetOrgSnapshot(context.Context, *SetOrgSnapshotRequest) (*SetOrgSnapshotResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetOrgSnapshot not implemented")
}
func (UnimplementedGramophileEServiceServer) AddWant(context.Context, *AddWantRequest) (*AddWantResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddWant not implemented")
}
func (UnimplementedGramophileEServiceServer) GetWants(context.Context, *GetWantsRequest) (*GetWantsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetWants not implemented")
}
func (UnimplementedGramophileEServiceServer) AddWantlist(context.Context, *AddWantlistRequest) (*AddWantlistResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddWantlist not implemented")
}
func (UnimplementedGramophileEServiceServer) GetWantlist(context.Context, *GetWantlistRequest) (*GetWantlistResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetWantlist not implemented")
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

func _GramophileEService_GetState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileEServiceServer).GetState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileEService/GetState",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileEServiceServer).GetState(ctx, req.(*GetStateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GramophileEService_SetConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetConfigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileEServiceServer).SetConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileEService/SetConfig",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileEServiceServer).SetConfig(ctx, req.(*SetConfigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GramophileEService_SetIntent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetIntentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileEServiceServer).SetIntent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileEService/SetIntent",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileEServiceServer).SetIntent(ctx, req.(*SetIntentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GramophileEService_GetRecord_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRecordRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileEServiceServer).GetRecord(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileEService/GetRecord",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileEServiceServer).GetRecord(ctx, req.(*GetRecordRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GramophileEService_GetOrg_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetOrgRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileEServiceServer).GetOrg(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileEService/GetOrg",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileEServiceServer).GetOrg(ctx, req.(*GetOrgRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GramophileEService_SetOrgSnapshot_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetOrgSnapshotRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileEServiceServer).SetOrgSnapshot(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileEService/SetOrgSnapshot",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileEServiceServer).SetOrgSnapshot(ctx, req.(*SetOrgSnapshotRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GramophileEService_AddWant_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddWantRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileEServiceServer).AddWant(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileEService/AddWant",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileEServiceServer).AddWant(ctx, req.(*AddWantRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GramophileEService_GetWants_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetWantsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileEServiceServer).GetWants(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileEService/GetWants",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileEServiceServer).GetWants(ctx, req.(*GetWantsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GramophileEService_AddWantlist_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddWantlistRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileEServiceServer).AddWantlist(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileEService/AddWantlist",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileEServiceServer).AddWantlist(ctx, req.(*AddWantlistRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GramophileEService_GetWantlist_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetWantlistRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileEServiceServer).GetWantlist(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileEService/GetWantlist",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileEServiceServer).GetWantlist(ctx, req.(*GetWantlistRequest))
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
		{
			MethodName: "GetState",
			Handler:    _GramophileEService_GetState_Handler,
		},
		{
			MethodName: "SetConfig",
			Handler:    _GramophileEService_SetConfig_Handler,
		},
		{
			MethodName: "SetIntent",
			Handler:    _GramophileEService_SetIntent_Handler,
		},
		{
			MethodName: "GetRecord",
			Handler:    _GramophileEService_GetRecord_Handler,
		},
		{
			MethodName: "GetOrg",
			Handler:    _GramophileEService_GetOrg_Handler,
		},
		{
			MethodName: "SetOrgSnapshot",
			Handler:    _GramophileEService_SetOrgSnapshot_Handler,
		},
		{
			MethodName: "AddWant",
			Handler:    _GramophileEService_AddWant_Handler,
		},
		{
			MethodName: "GetWants",
			Handler:    _GramophileEService_GetWants_Handler,
		},
		{
			MethodName: "AddWantlist",
			Handler:    _GramophileEService_AddWantlist_Handler,
		},
		{
			MethodName: "GetWantlist",
			Handler:    _GramophileEService_GetWantlist_Handler,
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
	DeleteUser(ctx context.Context, in *DeleteUserRequest, opts ...grpc.CallOption) (*DeleteUserResponse, error)
	Clean(ctx context.Context, in *CleanRequest, opts ...grpc.CallOption) (*CleanResponse, error)
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

func (c *gramophileServiceClient) DeleteUser(ctx context.Context, in *DeleteUserRequest, opts ...grpc.CallOption) (*DeleteUserResponse, error) {
	out := new(DeleteUserResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileService/DeleteUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gramophileServiceClient) Clean(ctx context.Context, in *CleanRequest, opts ...grpc.CallOption) (*CleanResponse, error) {
	out := new(CleanResponse)
	err := c.cc.Invoke(ctx, "/gramophile.GramophileService/Clean", in, out, opts...)
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
	DeleteUser(context.Context, *DeleteUserRequest) (*DeleteUserResponse, error)
	Clean(context.Context, *CleanRequest) (*CleanResponse, error)
}

// UnimplementedGramophileServiceServer should be embedded to have forward compatible implementations.
type UnimplementedGramophileServiceServer struct {
}

func (UnimplementedGramophileServiceServer) GetUsers(context.Context, *GetUsersRequest) (*GetUsersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUsers not implemented")
}
func (UnimplementedGramophileServiceServer) DeleteUser(context.Context, *DeleteUserRequest) (*DeleteUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteUser not implemented")
}
func (UnimplementedGramophileServiceServer) Clean(context.Context, *CleanRequest) (*CleanResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Clean not implemented")
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

func _GramophileService_DeleteUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileServiceServer).DeleteUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileService/DeleteUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileServiceServer).DeleteUser(ctx, req.(*DeleteUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GramophileService_Clean_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CleanRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GramophileServiceServer).Clean(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gramophile.GramophileService/Clean",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GramophileServiceServer).Clean(ctx, req.(*CleanRequest))
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
			MethodName: "DeleteUser",
			Handler:    _GramophileService_DeleteUser_Handler,
		},
		{
			MethodName: "Clean",
			Handler:    _GramophileService_Clean_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "gramophile.proto",
}
