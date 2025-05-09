// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: iecmock.proto

package iecmock

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	IecMockRpc_Ping_FullMethodName = "/iecmock.IecMockRpc/Ping"
)

// IecMockRpcClient is the client API for IecMockRpc service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type IecMockRpcClient interface {
	Ping(ctx context.Context, in *Req, opts ...grpc.CallOption) (*Res, error)
}

type iecMockRpcClient struct {
	cc grpc.ClientConnInterface
}

func NewIecMockRpcClient(cc grpc.ClientConnInterface) IecMockRpcClient {
	return &iecMockRpcClient{cc}
}

func (c *iecMockRpcClient) Ping(ctx context.Context, in *Req, opts ...grpc.CallOption) (*Res, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Res)
	err := c.cc.Invoke(ctx, IecMockRpc_Ping_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// IecMockRpcServer is the server API for IecMockRpc service.
// All implementations must embed UnimplementedIecMockRpcServer
// for forward compatibility.
type IecMockRpcServer interface {
	Ping(context.Context, *Req) (*Res, error)
	mustEmbedUnimplementedIecMockRpcServer()
}

// UnimplementedIecMockRpcServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedIecMockRpcServer struct{}

func (UnimplementedIecMockRpcServer) Ping(context.Context, *Req) (*Res, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedIecMockRpcServer) mustEmbedUnimplementedIecMockRpcServer() {}
func (UnimplementedIecMockRpcServer) testEmbeddedByValue()                    {}

// UnsafeIecMockRpcServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to IecMockRpcServer will
// result in compilation errors.
type UnsafeIecMockRpcServer interface {
	mustEmbedUnimplementedIecMockRpcServer()
}

func RegisterIecMockRpcServer(s grpc.ServiceRegistrar, srv IecMockRpcServer) {
	// If the following call pancis, it indicates UnimplementedIecMockRpcServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&IecMockRpc_ServiceDesc, srv)
}

func _IecMockRpc_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IecMockRpcServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: IecMockRpc_Ping_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IecMockRpcServer).Ping(ctx, req.(*Req))
	}
	return interceptor(ctx, in, info, handler)
}

// IecMockRpc_ServiceDesc is the grpc.ServiceDesc for IecMockRpc service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var IecMockRpc_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "iecmock.IecMockRpc",
	HandlerType: (*IecMockRpcServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ping",
			Handler:    _IecMockRpc_Ping_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "iecmock.proto",
}
