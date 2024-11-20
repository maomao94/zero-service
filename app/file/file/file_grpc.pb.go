// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v5.28.3
// source: file.proto

package file

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

// FileRpcClient is the client API for FileRpc service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type FileRpcClient interface {
	Ping(ctx context.Context, in *Req, opts ...grpc.CallOption) (*Res, error)
	OssDetail(ctx context.Context, in *OssDetailReq, opts ...grpc.CallOption) (*OssDetailRes, error)
	OssList(ctx context.Context, in *OssListReq, opts ...grpc.CallOption) (*OssListRes, error)
	CreateOss(ctx context.Context, in *CreateOssReq, opts ...grpc.CallOption) (*CreateOssRes, error)
	UpdateOss(ctx context.Context, in *UpdateOssReq, opts ...grpc.CallOption) (*UpdateOssRes, error)
	DeleteOss(ctx context.Context, in *DeleteOssReq, opts ...grpc.CallOption) (*DeleteOssRes, error)
	MakeBucket(ctx context.Context, in *MakeBucketReq, opts ...grpc.CallOption) (*MakeBucketRes, error)
	RemoveBucket(ctx context.Context, in *RemoveBucketReq, opts ...grpc.CallOption) (*RemoveBucketRes, error)
	StatFile(ctx context.Context, in *StatFileReq, opts ...grpc.CallOption) (*StatFileRes, error)
	PutFile(ctx context.Context, in *PutFileReq, opts ...grpc.CallOption) (*PutFileRes, error)
	PutFileByte(ctx context.Context, opts ...grpc.CallOption) (FileRpc_PutFileByteClient, error)
	RemoveFile(ctx context.Context, in *RemoveFileReq, opts ...grpc.CallOption) (*RemoveFileRes, error)
	RemoveFiles(ctx context.Context, in *RemoveFilesReq, opts ...grpc.CallOption) (*RemoveFileRes, error)
}

type fileRpcClient struct {
	cc grpc.ClientConnInterface
}

func NewFileRpcClient(cc grpc.ClientConnInterface) FileRpcClient {
	return &fileRpcClient{cc}
}

func (c *fileRpcClient) Ping(ctx context.Context, in *Req, opts ...grpc.CallOption) (*Res, error) {
	out := new(Res)
	err := c.cc.Invoke(ctx, "/file.FileRpc/Ping", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fileRpcClient) OssDetail(ctx context.Context, in *OssDetailReq, opts ...grpc.CallOption) (*OssDetailRes, error) {
	out := new(OssDetailRes)
	err := c.cc.Invoke(ctx, "/file.FileRpc/OssDetail", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fileRpcClient) OssList(ctx context.Context, in *OssListReq, opts ...grpc.CallOption) (*OssListRes, error) {
	out := new(OssListRes)
	err := c.cc.Invoke(ctx, "/file.FileRpc/OssList", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fileRpcClient) CreateOss(ctx context.Context, in *CreateOssReq, opts ...grpc.CallOption) (*CreateOssRes, error) {
	out := new(CreateOssRes)
	err := c.cc.Invoke(ctx, "/file.FileRpc/CreateOss", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fileRpcClient) UpdateOss(ctx context.Context, in *UpdateOssReq, opts ...grpc.CallOption) (*UpdateOssRes, error) {
	out := new(UpdateOssRes)
	err := c.cc.Invoke(ctx, "/file.FileRpc/UpdateOss", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fileRpcClient) DeleteOss(ctx context.Context, in *DeleteOssReq, opts ...grpc.CallOption) (*DeleteOssRes, error) {
	out := new(DeleteOssRes)
	err := c.cc.Invoke(ctx, "/file.FileRpc/DeleteOss", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fileRpcClient) MakeBucket(ctx context.Context, in *MakeBucketReq, opts ...grpc.CallOption) (*MakeBucketRes, error) {
	out := new(MakeBucketRes)
	err := c.cc.Invoke(ctx, "/file.FileRpc/MakeBucket", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fileRpcClient) RemoveBucket(ctx context.Context, in *RemoveBucketReq, opts ...grpc.CallOption) (*RemoveBucketRes, error) {
	out := new(RemoveBucketRes)
	err := c.cc.Invoke(ctx, "/file.FileRpc/RemoveBucket", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fileRpcClient) StatFile(ctx context.Context, in *StatFileReq, opts ...grpc.CallOption) (*StatFileRes, error) {
	out := new(StatFileRes)
	err := c.cc.Invoke(ctx, "/file.FileRpc/StatFile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fileRpcClient) PutFile(ctx context.Context, in *PutFileReq, opts ...grpc.CallOption) (*PutFileRes, error) {
	out := new(PutFileRes)
	err := c.cc.Invoke(ctx, "/file.FileRpc/PutFile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fileRpcClient) PutFileByte(ctx context.Context, opts ...grpc.CallOption) (FileRpc_PutFileByteClient, error) {
	stream, err := c.cc.NewStream(ctx, &FileRpc_ServiceDesc.Streams[0], "/file.FileRpc/PutFileByte", opts...)
	if err != nil {
		return nil, err
	}
	x := &fileRpcPutFileByteClient{stream}
	return x, nil
}

type FileRpc_PutFileByteClient interface {
	Send(*PutFileByteReq) error
	CloseAndRecv() (*PutFileByteRes, error)
	grpc.ClientStream
}

type fileRpcPutFileByteClient struct {
	grpc.ClientStream
}

func (x *fileRpcPutFileByteClient) Send(m *PutFileByteReq) error {
	return x.ClientStream.SendMsg(m)
}

func (x *fileRpcPutFileByteClient) CloseAndRecv() (*PutFileByteRes, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(PutFileByteRes)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *fileRpcClient) RemoveFile(ctx context.Context, in *RemoveFileReq, opts ...grpc.CallOption) (*RemoveFileRes, error) {
	out := new(RemoveFileRes)
	err := c.cc.Invoke(ctx, "/file.FileRpc/RemoveFile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fileRpcClient) RemoveFiles(ctx context.Context, in *RemoveFilesReq, opts ...grpc.CallOption) (*RemoveFileRes, error) {
	out := new(RemoveFileRes)
	err := c.cc.Invoke(ctx, "/file.FileRpc/RemoveFiles", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FileRpcServer is the server API for FileRpc service.
// All implementations must embed UnimplementedFileRpcServer
// for forward compatibility
type FileRpcServer interface {
	Ping(context.Context, *Req) (*Res, error)
	OssDetail(context.Context, *OssDetailReq) (*OssDetailRes, error)
	OssList(context.Context, *OssListReq) (*OssListRes, error)
	CreateOss(context.Context, *CreateOssReq) (*CreateOssRes, error)
	UpdateOss(context.Context, *UpdateOssReq) (*UpdateOssRes, error)
	DeleteOss(context.Context, *DeleteOssReq) (*DeleteOssRes, error)
	MakeBucket(context.Context, *MakeBucketReq) (*MakeBucketRes, error)
	RemoveBucket(context.Context, *RemoveBucketReq) (*RemoveBucketRes, error)
	StatFile(context.Context, *StatFileReq) (*StatFileRes, error)
	PutFile(context.Context, *PutFileReq) (*PutFileRes, error)
	PutFileByte(FileRpc_PutFileByteServer) error
	RemoveFile(context.Context, *RemoveFileReq) (*RemoveFileRes, error)
	RemoveFiles(context.Context, *RemoveFilesReq) (*RemoveFileRes, error)
	mustEmbedUnimplementedFileRpcServer()
}

// UnimplementedFileRpcServer must be embedded to have forward compatible implementations.
type UnimplementedFileRpcServer struct {
}

func (UnimplementedFileRpcServer) Ping(context.Context, *Req) (*Res, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedFileRpcServer) OssDetail(context.Context, *OssDetailReq) (*OssDetailRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OssDetail not implemented")
}
func (UnimplementedFileRpcServer) OssList(context.Context, *OssListReq) (*OssListRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OssList not implemented")
}
func (UnimplementedFileRpcServer) CreateOss(context.Context, *CreateOssReq) (*CreateOssRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateOss not implemented")
}
func (UnimplementedFileRpcServer) UpdateOss(context.Context, *UpdateOssReq) (*UpdateOssRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateOss not implemented")
}
func (UnimplementedFileRpcServer) DeleteOss(context.Context, *DeleteOssReq) (*DeleteOssRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteOss not implemented")
}
func (UnimplementedFileRpcServer) MakeBucket(context.Context, *MakeBucketReq) (*MakeBucketRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MakeBucket not implemented")
}
func (UnimplementedFileRpcServer) RemoveBucket(context.Context, *RemoveBucketReq) (*RemoveBucketRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveBucket not implemented")
}
func (UnimplementedFileRpcServer) StatFile(context.Context, *StatFileReq) (*StatFileRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StatFile not implemented")
}
func (UnimplementedFileRpcServer) PutFile(context.Context, *PutFileReq) (*PutFileRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PutFile not implemented")
}
func (UnimplementedFileRpcServer) PutFileByte(FileRpc_PutFileByteServer) error {
	return status.Errorf(codes.Unimplemented, "method PutFileByte not implemented")
}
func (UnimplementedFileRpcServer) RemoveFile(context.Context, *RemoveFileReq) (*RemoveFileRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveFile not implemented")
}
func (UnimplementedFileRpcServer) RemoveFiles(context.Context, *RemoveFilesReq) (*RemoveFileRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveFiles not implemented")
}
func (UnimplementedFileRpcServer) mustEmbedUnimplementedFileRpcServer() {}

// UnsafeFileRpcServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to FileRpcServer will
// result in compilation errors.
type UnsafeFileRpcServer interface {
	mustEmbedUnimplementedFileRpcServer()
}

func RegisterFileRpcServer(s grpc.ServiceRegistrar, srv FileRpcServer) {
	s.RegisterService(&FileRpc_ServiceDesc, srv)
}

func _FileRpc_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileRpcServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/file.FileRpc/Ping",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileRpcServer).Ping(ctx, req.(*Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _FileRpc_OssDetail_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OssDetailReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileRpcServer).OssDetail(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/file.FileRpc/OssDetail",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileRpcServer).OssDetail(ctx, req.(*OssDetailReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _FileRpc_OssList_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OssListReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileRpcServer).OssList(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/file.FileRpc/OssList",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileRpcServer).OssList(ctx, req.(*OssListReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _FileRpc_CreateOss_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateOssReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileRpcServer).CreateOss(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/file.FileRpc/CreateOss",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileRpcServer).CreateOss(ctx, req.(*CreateOssReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _FileRpc_UpdateOss_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateOssReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileRpcServer).UpdateOss(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/file.FileRpc/UpdateOss",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileRpcServer).UpdateOss(ctx, req.(*UpdateOssReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _FileRpc_DeleteOss_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteOssReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileRpcServer).DeleteOss(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/file.FileRpc/DeleteOss",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileRpcServer).DeleteOss(ctx, req.(*DeleteOssReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _FileRpc_MakeBucket_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MakeBucketReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileRpcServer).MakeBucket(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/file.FileRpc/MakeBucket",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileRpcServer).MakeBucket(ctx, req.(*MakeBucketReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _FileRpc_RemoveBucket_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveBucketReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileRpcServer).RemoveBucket(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/file.FileRpc/RemoveBucket",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileRpcServer).RemoveBucket(ctx, req.(*RemoveBucketReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _FileRpc_StatFile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StatFileReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileRpcServer).StatFile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/file.FileRpc/StatFile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileRpcServer).StatFile(ctx, req.(*StatFileReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _FileRpc_PutFile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PutFileReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileRpcServer).PutFile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/file.FileRpc/PutFile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileRpcServer).PutFile(ctx, req.(*PutFileReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _FileRpc_PutFileByte_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(FileRpcServer).PutFileByte(&fileRpcPutFileByteServer{stream})
}

type FileRpc_PutFileByteServer interface {
	SendAndClose(*PutFileByteRes) error
	Recv() (*PutFileByteReq, error)
	grpc.ServerStream
}

type fileRpcPutFileByteServer struct {
	grpc.ServerStream
}

func (x *fileRpcPutFileByteServer) SendAndClose(m *PutFileByteRes) error {
	return x.ServerStream.SendMsg(m)
}

func (x *fileRpcPutFileByteServer) Recv() (*PutFileByteReq, error) {
	m := new(PutFileByteReq)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _FileRpc_RemoveFile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveFileReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileRpcServer).RemoveFile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/file.FileRpc/RemoveFile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileRpcServer).RemoveFile(ctx, req.(*RemoveFileReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _FileRpc_RemoveFiles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveFilesReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileRpcServer).RemoveFiles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/file.FileRpc/RemoveFiles",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileRpcServer).RemoveFiles(ctx, req.(*RemoveFilesReq))
	}
	return interceptor(ctx, in, info, handler)
}

// FileRpc_ServiceDesc is the grpc.ServiceDesc for FileRpc service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var FileRpc_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "file.FileRpc",
	HandlerType: (*FileRpcServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ping",
			Handler:    _FileRpc_Ping_Handler,
		},
		{
			MethodName: "OssDetail",
			Handler:    _FileRpc_OssDetail_Handler,
		},
		{
			MethodName: "OssList",
			Handler:    _FileRpc_OssList_Handler,
		},
		{
			MethodName: "CreateOss",
			Handler:    _FileRpc_CreateOss_Handler,
		},
		{
			MethodName: "UpdateOss",
			Handler:    _FileRpc_UpdateOss_Handler,
		},
		{
			MethodName: "DeleteOss",
			Handler:    _FileRpc_DeleteOss_Handler,
		},
		{
			MethodName: "MakeBucket",
			Handler:    _FileRpc_MakeBucket_Handler,
		},
		{
			MethodName: "RemoveBucket",
			Handler:    _FileRpc_RemoveBucket_Handler,
		},
		{
			MethodName: "StatFile",
			Handler:    _FileRpc_StatFile_Handler,
		},
		{
			MethodName: "PutFile",
			Handler:    _FileRpc_PutFile_Handler,
		},
		{
			MethodName: "RemoveFile",
			Handler:    _FileRpc_RemoveFile_Handler,
		},
		{
			MethodName: "RemoveFiles",
			Handler:    _FileRpc_RemoveFiles_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "PutFileByte",
			Handler:       _FileRpc_PutFileByte_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "file.proto",
}
