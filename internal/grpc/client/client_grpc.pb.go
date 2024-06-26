// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.19.3
// source: grpc/client.proto

package client

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// FileServerClient is the client API for FileServer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type FileServerClient interface {
	Upload(ctx context.Context, in *UploadRequest, opts ...grpc.CallOption) (*UploadResponse, error)
	Download(ctx context.Context, in *DownloadRequest, opts ...grpc.CallOption) (*DownloadResponse, error)
	List(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ListResponse, error)
	CheckIfExists(ctx context.Context, in *DownloadRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type fileServerClient struct {
	cc grpc.ClientConnInterface
}

func NewFileServerClient(cc grpc.ClientConnInterface) FileServerClient {
	return &fileServerClient{cc}
}

func (c *fileServerClient) Upload(ctx context.Context, in *UploadRequest, opts ...grpc.CallOption) (*UploadResponse, error) {
	out := new(UploadResponse)
	err := c.cc.Invoke(ctx, "/client.FileServer/Upload", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fileServerClient) Download(ctx context.Context, in *DownloadRequest, opts ...grpc.CallOption) (*DownloadResponse, error) {
	out := new(DownloadResponse)
	err := c.cc.Invoke(ctx, "/client.FileServer/Download", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fileServerClient) List(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ListResponse, error) {
	out := new(ListResponse)
	err := c.cc.Invoke(ctx, "/client.FileServer/List", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fileServerClient) CheckIfExists(ctx context.Context, in *DownloadRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/client.FileServer/CheckIfExists", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FileServerServer is the server API for FileServer service.
// All implementations must embed UnimplementedFileServerServer
// for forward compatibility
type FileServerServer interface {
	Upload(context.Context, *UploadRequest) (*UploadResponse, error)
	Download(context.Context, *DownloadRequest) (*DownloadResponse, error)
	List(context.Context, *emptypb.Empty) (*ListResponse, error)
	CheckIfExists(context.Context, *DownloadRequest) (*emptypb.Empty, error)
	mustEmbedUnimplementedFileServerServer()
}

// UnimplementedFileServerServer must be embedded to have forward compatible implementations.
type UnimplementedFileServerServer struct {
}

func (UnimplementedFileServerServer) Upload(context.Context, *UploadRequest) (*UploadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Upload not implemented")
}
func (UnimplementedFileServerServer) Download(context.Context, *DownloadRequest) (*DownloadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Download not implemented")
}
func (UnimplementedFileServerServer) List(context.Context, *emptypb.Empty) (*ListResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method List not implemented")
}
func (UnimplementedFileServerServer) CheckIfExists(context.Context, *DownloadRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CheckIfExists not implemented")
}
func (UnimplementedFileServerServer) mustEmbedUnimplementedFileServerServer() {}

// UnsafeFileServerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to FileServerServer will
// result in compilation errors.
type UnsafeFileServerServer interface {
	mustEmbedUnimplementedFileServerServer()
}

func RegisterFileServerServer(s grpc.ServiceRegistrar, srv FileServerServer) {
	s.RegisterService(&FileServer_ServiceDesc, srv)
}

func _FileServer_Upload_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UploadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileServerServer).Upload(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/client.FileServer/Upload",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileServerServer).Upload(ctx, req.(*UploadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FileServer_Download_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DownloadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileServerServer).Download(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/client.FileServer/Download",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileServerServer).Download(ctx, req.(*DownloadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FileServer_List_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileServerServer).List(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/client.FileServer/List",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileServerServer).List(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _FileServer_CheckIfExists_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DownloadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FileServerServer).CheckIfExists(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/client.FileServer/CheckIfExists",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FileServerServer).CheckIfExists(ctx, req.(*DownloadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// FileServer_ServiceDesc is the grpc.ServiceDesc for FileServer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var FileServer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "client.FileServer",
	HandlerType: (*FileServerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Upload",
			Handler:    _FileServer_Upload_Handler,
		},
		{
			MethodName: "Download",
			Handler:    _FileServer_Download_Handler,
		},
		{
			MethodName: "List",
			Handler:    _FileServer_List_Handler,
		},
		{
			MethodName: "CheckIfExists",
			Handler:    _FileServer_CheckIfExists_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "grpc/client.proto",
}
