// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.2
// source: protobuf/dapplink-wallet.proto

package da_wallet_go

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
	BusinessMiddleWireService_BusinessRegister_FullMethodName            = "/syncs.BusinessMiddleWireService/businessRegister"
	BusinessMiddleWireService_ExportAddressesByPublicKeys_FullMethodName = "/syncs.BusinessMiddleWireService/exportAddressesByPublicKeys"
	BusinessMiddleWireService_CreateUnSignTransaction_FullMethodName     = "/syncs.BusinessMiddleWireService/createUnSignTransaction"
	BusinessMiddleWireService_BuildSignedTransaction_FullMethodName      = "/syncs.BusinessMiddleWireService/buildSignedTransaction"
	BusinessMiddleWireService_SetTokenAddress_FullMethodName             = "/syncs.BusinessMiddleWireService/setTokenAddress"
)

// BusinessMiddleWireServiceClient is the client API for BusinessMiddleWireService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BusinessMiddleWireServiceClient interface {
	BusinessRegister(ctx context.Context, in *BusinessRegisterRequest, opts ...grpc.CallOption) (*BusinessRegisterResponse, error)
	ExportAddressesByPublicKeys(ctx context.Context, in *ExportAddressesRequest, opts ...grpc.CallOption) (*ExportAddressesResponse, error)
	CreateUnSignTransaction(ctx context.Context, in *UnSignTransactionRequest, opts ...grpc.CallOption) (*UnSignTransactionResponse, error)
	BuildSignedTransaction(ctx context.Context, in *SignTransactionRequest, opts ...grpc.CallOption) (*SignTransactionResponse, error)
	SetTokenAddress(ctx context.Context, in *SetTokenAddressRequest, opts ...grpc.CallOption) (*SetTokenAddressResponse, error)
}

type businessMiddleWireServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBusinessMiddleWireServiceClient(cc grpc.ClientConnInterface) BusinessMiddleWireServiceClient {
	return &businessMiddleWireServiceClient{cc}
}

func (c *businessMiddleWireServiceClient) BusinessRegister(ctx context.Context, in *BusinessRegisterRequest, opts ...grpc.CallOption) (*BusinessRegisterResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(BusinessRegisterResponse)
	err := c.cc.Invoke(ctx, BusinessMiddleWireService_BusinessRegister_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *businessMiddleWireServiceClient) ExportAddressesByPublicKeys(ctx context.Context, in *ExportAddressesRequest, opts ...grpc.CallOption) (*ExportAddressesResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ExportAddressesResponse)
	err := c.cc.Invoke(ctx, BusinessMiddleWireService_ExportAddressesByPublicKeys_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *businessMiddleWireServiceClient) CreateUnSignTransaction(ctx context.Context, in *UnSignTransactionRequest, opts ...grpc.CallOption) (*UnSignTransactionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UnSignTransactionResponse)
	err := c.cc.Invoke(ctx, BusinessMiddleWireService_CreateUnSignTransaction_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *businessMiddleWireServiceClient) BuildSignedTransaction(ctx context.Context, in *SignTransactionRequest, opts ...grpc.CallOption) (*SignTransactionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SignTransactionResponse)
	err := c.cc.Invoke(ctx, BusinessMiddleWireService_BuildSignedTransaction_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *businessMiddleWireServiceClient) SetTokenAddress(ctx context.Context, in *SetTokenAddressRequest, opts ...grpc.CallOption) (*SetTokenAddressResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SetTokenAddressResponse)
	err := c.cc.Invoke(ctx, BusinessMiddleWireService_SetTokenAddress_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BusinessMiddleWireServiceServer is the server API for BusinessMiddleWireService service.
// All implementations should embed UnimplementedBusinessMiddleWireServiceServer
// for forward compatibility.
type BusinessMiddleWireServiceServer interface {
	BusinessRegister(context.Context, *BusinessRegisterRequest) (*BusinessRegisterResponse, error)
	ExportAddressesByPublicKeys(context.Context, *ExportAddressesRequest) (*ExportAddressesResponse, error)
	CreateUnSignTransaction(context.Context, *UnSignTransactionRequest) (*UnSignTransactionResponse, error)
	BuildSignedTransaction(context.Context, *SignTransactionRequest) (*SignTransactionResponse, error)
	SetTokenAddress(context.Context, *SetTokenAddressRequest) (*SetTokenAddressResponse, error)
}

// UnimplementedBusinessMiddleWireServiceServer should be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedBusinessMiddleWireServiceServer struct{}

func (UnimplementedBusinessMiddleWireServiceServer) BusinessRegister(context.Context, *BusinessRegisterRequest) (*BusinessRegisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BusinessRegister not implemented")
}
func (UnimplementedBusinessMiddleWireServiceServer) ExportAddressesByPublicKeys(context.Context, *ExportAddressesRequest) (*ExportAddressesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ExportAddressesByPublicKeys not implemented")
}
func (UnimplementedBusinessMiddleWireServiceServer) CreateUnSignTransaction(context.Context, *UnSignTransactionRequest) (*UnSignTransactionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateUnSignTransaction not implemented")
}
func (UnimplementedBusinessMiddleWireServiceServer) BuildSignedTransaction(context.Context, *SignTransactionRequest) (*SignTransactionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BuildSignedTransaction not implemented")
}
func (UnimplementedBusinessMiddleWireServiceServer) SetTokenAddress(context.Context, *SetTokenAddressRequest) (*SetTokenAddressResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetTokenAddress not implemented")
}
func (UnimplementedBusinessMiddleWireServiceServer) testEmbeddedByValue() {}

// UnsafeBusinessMiddleWireServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BusinessMiddleWireServiceServer will
// result in compilation errors.
type UnsafeBusinessMiddleWireServiceServer interface {
	mustEmbedUnimplementedBusinessMiddleWireServiceServer()
}

func RegisterBusinessMiddleWireServiceServer(s grpc.ServiceRegistrar, srv BusinessMiddleWireServiceServer) {
	// If the following call pancis, it indicates UnimplementedBusinessMiddleWireServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&BusinessMiddleWireService_ServiceDesc, srv)
}

func _BusinessMiddleWireService_BusinessRegister_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BusinessRegisterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BusinessMiddleWireServiceServer).BusinessRegister(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BusinessMiddleWireService_BusinessRegister_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BusinessMiddleWireServiceServer).BusinessRegister(ctx, req.(*BusinessRegisterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BusinessMiddleWireService_ExportAddressesByPublicKeys_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ExportAddressesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BusinessMiddleWireServiceServer).ExportAddressesByPublicKeys(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BusinessMiddleWireService_ExportAddressesByPublicKeys_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BusinessMiddleWireServiceServer).ExportAddressesByPublicKeys(ctx, req.(*ExportAddressesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BusinessMiddleWireService_CreateUnSignTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UnSignTransactionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BusinessMiddleWireServiceServer).CreateUnSignTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BusinessMiddleWireService_CreateUnSignTransaction_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BusinessMiddleWireServiceServer).CreateUnSignTransaction(ctx, req.(*UnSignTransactionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BusinessMiddleWireService_BuildSignedTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignTransactionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BusinessMiddleWireServiceServer).BuildSignedTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BusinessMiddleWireService_BuildSignedTransaction_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BusinessMiddleWireServiceServer).BuildSignedTransaction(ctx, req.(*SignTransactionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BusinessMiddleWireService_SetTokenAddress_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetTokenAddressRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BusinessMiddleWireServiceServer).SetTokenAddress(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BusinessMiddleWireService_SetTokenAddress_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BusinessMiddleWireServiceServer).SetTokenAddress(ctx, req.(*SetTokenAddressRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// BusinessMiddleWireService_ServiceDesc is the grpc.ServiceDesc for BusinessMiddleWireService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BusinessMiddleWireService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "syncs.BusinessMiddleWireService",
	HandlerType: (*BusinessMiddleWireServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "businessRegister",
			Handler:    _BusinessMiddleWireService_BusinessRegister_Handler,
		},
		{
			MethodName: "exportAddressesByPublicKeys",
			Handler:    _BusinessMiddleWireService_ExportAddressesByPublicKeys_Handler,
		},
		{
			MethodName: "createUnSignTransaction",
			Handler:    _BusinessMiddleWireService_CreateUnSignTransaction_Handler,
		},
		{
			MethodName: "buildSignedTransaction",
			Handler:    _BusinessMiddleWireService_BuildSignedTransaction_Handler,
		},
		{
			MethodName: "setTokenAddress",
			Handler:    _BusinessMiddleWireService_SetTokenAddress_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "protobuf/dapplink-wallet.proto",
}
