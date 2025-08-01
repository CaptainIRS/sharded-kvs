// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v5.29.3
// source: protos/shard.proto

package internal

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

// ShardRPCClient is the client API for ShardRPC service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ShardRPCClient interface {
	Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error)
	Put(ctx context.Context, in *PutRequest, opts ...grpc.CallOption) (*PutResponse, error)
	Delete(ctx context.Context, in *DeleteRequest, opts ...grpc.CallOption) (*DeleteResponse, error)
	PauseWrites(ctx context.Context, in *PauseWritesRequest, opts ...grpc.CallOption) (*PauseWritesResponse, error)
	SendKeys(ctx context.Context, in *SendKeysRequest, opts ...grpc.CallOption) (*SendKeysResponse, error)
	PurgeKeys(ctx context.Context, in *PurgeKeysRequest, opts ...grpc.CallOption) (*PurgeKeysResponse, error)
	ResumeWrites(ctx context.Context, in *ResumeWritesRequest, opts ...grpc.CallOption) (*ResumeWritesResponse, error)
}

type shardRPCClient struct {
	cc grpc.ClientConnInterface
}

func NewShardRPCClient(cc grpc.ClientConnInterface) ShardRPCClient {
	return &shardRPCClient{cc}
}

func (c *shardRPCClient) Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error) {
	out := new(GetResponse)
	err := c.cc.Invoke(ctx, "/protos.ShardRPC/Get", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *shardRPCClient) Put(ctx context.Context, in *PutRequest, opts ...grpc.CallOption) (*PutResponse, error) {
	out := new(PutResponse)
	err := c.cc.Invoke(ctx, "/protos.ShardRPC/Put", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *shardRPCClient) Delete(ctx context.Context, in *DeleteRequest, opts ...grpc.CallOption) (*DeleteResponse, error) {
	out := new(DeleteResponse)
	err := c.cc.Invoke(ctx, "/protos.ShardRPC/Delete", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *shardRPCClient) PauseWrites(ctx context.Context, in *PauseWritesRequest, opts ...grpc.CallOption) (*PauseWritesResponse, error) {
	out := new(PauseWritesResponse)
	err := c.cc.Invoke(ctx, "/protos.ShardRPC/PauseWrites", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *shardRPCClient) SendKeys(ctx context.Context, in *SendKeysRequest, opts ...grpc.CallOption) (*SendKeysResponse, error) {
	out := new(SendKeysResponse)
	err := c.cc.Invoke(ctx, "/protos.ShardRPC/SendKeys", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *shardRPCClient) PurgeKeys(ctx context.Context, in *PurgeKeysRequest, opts ...grpc.CallOption) (*PurgeKeysResponse, error) {
	out := new(PurgeKeysResponse)
	err := c.cc.Invoke(ctx, "/protos.ShardRPC/PurgeKeys", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *shardRPCClient) ResumeWrites(ctx context.Context, in *ResumeWritesRequest, opts ...grpc.CallOption) (*ResumeWritesResponse, error) {
	out := new(ResumeWritesResponse)
	err := c.cc.Invoke(ctx, "/protos.ShardRPC/ResumeWrites", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ShardRPCServer is the server API for ShardRPC service.
// All implementations must embed UnimplementedShardRPCServer
// for forward compatibility
type ShardRPCServer interface {
	Get(context.Context, *GetRequest) (*GetResponse, error)
	Put(context.Context, *PutRequest) (*PutResponse, error)
	Delete(context.Context, *DeleteRequest) (*DeleteResponse, error)
	PauseWrites(context.Context, *PauseWritesRequest) (*PauseWritesResponse, error)
	SendKeys(context.Context, *SendKeysRequest) (*SendKeysResponse, error)
	PurgeKeys(context.Context, *PurgeKeysRequest) (*PurgeKeysResponse, error)
	ResumeWrites(context.Context, *ResumeWritesRequest) (*ResumeWritesResponse, error)
	mustEmbedUnimplementedShardRPCServer()
}

// UnimplementedShardRPCServer must be embedded to have forward compatible implementations.
type UnimplementedShardRPCServer struct {
}

func (UnimplementedShardRPCServer) Get(context.Context, *GetRequest) (*GetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Get not implemented")
}
func (UnimplementedShardRPCServer) Put(context.Context, *PutRequest) (*PutResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Put not implemented")
}
func (UnimplementedShardRPCServer) Delete(context.Context, *DeleteRequest) (*DeleteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedShardRPCServer) PauseWrites(context.Context, *PauseWritesRequest) (*PauseWritesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PauseWrites not implemented")
}
func (UnimplementedShardRPCServer) SendKeys(context.Context, *SendKeysRequest) (*SendKeysResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SendKeys not implemented")
}
func (UnimplementedShardRPCServer) PurgeKeys(context.Context, *PurgeKeysRequest) (*PurgeKeysResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PurgeKeys not implemented")
}
func (UnimplementedShardRPCServer) ResumeWrites(context.Context, *ResumeWritesRequest) (*ResumeWritesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ResumeWrites not implemented")
}
func (UnimplementedShardRPCServer) mustEmbedUnimplementedShardRPCServer() {}

// UnsafeShardRPCServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ShardRPCServer will
// result in compilation errors.
type UnsafeShardRPCServer interface {
	mustEmbedUnimplementedShardRPCServer()
}

func RegisterShardRPCServer(s grpc.ServiceRegistrar, srv ShardRPCServer) {
	s.RegisterService(&ShardRPC_ServiceDesc, srv)
}

func _ShardRPC_Get_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ShardRPCServer).Get(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.ShardRPC/Get",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ShardRPCServer).Get(ctx, req.(*GetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ShardRPC_Put_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PutRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ShardRPCServer).Put(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.ShardRPC/Put",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ShardRPCServer).Put(ctx, req.(*PutRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ShardRPC_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ShardRPCServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.ShardRPC/Delete",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ShardRPCServer).Delete(ctx, req.(*DeleteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ShardRPC_PauseWrites_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PauseWritesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ShardRPCServer).PauseWrites(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.ShardRPC/PauseWrites",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ShardRPCServer).PauseWrites(ctx, req.(*PauseWritesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ShardRPC_SendKeys_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SendKeysRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ShardRPCServer).SendKeys(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.ShardRPC/SendKeys",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ShardRPCServer).SendKeys(ctx, req.(*SendKeysRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ShardRPC_PurgeKeys_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PurgeKeysRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ShardRPCServer).PurgeKeys(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.ShardRPC/PurgeKeys",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ShardRPCServer).PurgeKeys(ctx, req.(*PurgeKeysRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ShardRPC_ResumeWrites_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ResumeWritesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ShardRPCServer).ResumeWrites(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.ShardRPC/ResumeWrites",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ShardRPCServer).ResumeWrites(ctx, req.(*ResumeWritesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ShardRPC_ServiceDesc is the grpc.ServiceDesc for ShardRPC service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ShardRPC_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "protos.ShardRPC",
	HandlerType: (*ShardRPCServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Get",
			Handler:    _ShardRPC_Get_Handler,
		},
		{
			MethodName: "Put",
			Handler:    _ShardRPC_Put_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _ShardRPC_Delete_Handler,
		},
		{
			MethodName: "PauseWrites",
			Handler:    _ShardRPC_PauseWrites_Handler,
		},
		{
			MethodName: "SendKeys",
			Handler:    _ShardRPC_SendKeys_Handler,
		},
		{
			MethodName: "PurgeKeys",
			Handler:    _ShardRPC_PurgeKeys_Handler,
		},
		{
			MethodName: "ResumeWrites",
			Handler:    _ShardRPC_ResumeWrites_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "protos/shard.proto",
}
