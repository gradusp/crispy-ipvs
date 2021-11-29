// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package ipvs

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

// IpvsAdminClient is the client API for IpvsAdmin service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type IpvsAdminClient interface {
	// Find IP-virtual server by its identity
	FindVirtualServer(ctx context.Context, in *FindVirtualServerRequest, opts ...grpc.CallOption) (*FindVirtualServerResponse, error)
	// List all IP-virtual servers with/without theirs real servers
	ListVirtualServers(ctx context.Context, in *ListVirtualServersRequest, opts ...grpc.CallOption) (*ListVirtualServersResponse, error)
	// Update IP-virtual servers
	UpdateVirtualServers(ctx context.Context, in *UpdateVirtualServersRequest, opts ...grpc.CallOption) (*UpdateVirtualServersResponse, error)
	// Update real servers for one IP-virtual server
	UpdateRealServers(ctx context.Context, in *UpdateRealServersRequest, opts ...grpc.CallOption) (*UpdateRealServersResponse, error)
}

type ipvsAdminClient struct {
	cc grpc.ClientConnInterface
}

func NewIpvsAdminClient(cc grpc.ClientConnInterface) IpvsAdminClient {
	return &ipvsAdminClient{cc}
}

func (c *ipvsAdminClient) FindVirtualServer(ctx context.Context, in *FindVirtualServerRequest, opts ...grpc.CallOption) (*FindVirtualServerResponse, error) {
	out := new(FindVirtualServerResponse)
	err := c.cc.Invoke(ctx, "/crispy.ipvs.IpvsAdmin/FindVirtualServer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *ipvsAdminClient) ListVirtualServers(ctx context.Context, in *ListVirtualServersRequest, opts ...grpc.CallOption) (*ListVirtualServersResponse, error) {
	out := new(ListVirtualServersResponse)
	err := c.cc.Invoke(ctx, "/crispy.ipvs.IpvsAdmin/ListVirtualServers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *ipvsAdminClient) UpdateVirtualServers(ctx context.Context, in *UpdateVirtualServersRequest, opts ...grpc.CallOption) (*UpdateVirtualServersResponse, error) {
	out := new(UpdateVirtualServersResponse)
	err := c.cc.Invoke(ctx, "/crispy.ipvs.IpvsAdmin/UpdateVirtualServers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *ipvsAdminClient) UpdateRealServers(ctx context.Context, in *UpdateRealServersRequest, opts ...grpc.CallOption) (*UpdateRealServersResponse, error) {
	out := new(UpdateRealServersResponse)
	err := c.cc.Invoke(ctx, "/crispy.ipvs.IpvsAdmin/UpdateRealServers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// IpvsAdminServer is the server API for IpvsAdmin service.
// All implementations must embed UnimplementedIpvsAdminServer
// for forward compatibility
type IpvsAdminServer interface {
	// Find IP-virtual server by its identity
	FindVirtualServer(context.Context, *FindVirtualServerRequest) (*FindVirtualServerResponse, error)
	// List all IP-virtual servers with/without theirs real servers
	ListVirtualServers(context.Context, *ListVirtualServersRequest) (*ListVirtualServersResponse, error)
	// Update IP-virtual servers
	UpdateVirtualServers(context.Context, *UpdateVirtualServersRequest) (*UpdateVirtualServersResponse, error)
	// Update real servers for one IP-virtual server
	UpdateRealServers(context.Context, *UpdateRealServersRequest) (*UpdateRealServersResponse, error)
	mustEmbedUnimplementedIpvsAdminServer()
}

// UnimplementedIpvsAdminServer must be embedded to have forward compatible implementations.
type UnimplementedIpvsAdminServer struct {
}

func (UnimplementedIpvsAdminServer) FindVirtualServer(context.Context, *FindVirtualServerRequest) (*FindVirtualServerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FindVirtualServer not implemented")
}
func (UnimplementedIpvsAdminServer) ListVirtualServers(context.Context, *ListVirtualServersRequest) (*ListVirtualServersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListVirtualServers not implemented")
}
func (UnimplementedIpvsAdminServer) UpdateVirtualServers(context.Context, *UpdateVirtualServersRequest) (*UpdateVirtualServersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateVirtualServers not implemented")
}
func (UnimplementedIpvsAdminServer) UpdateRealServers(context.Context, *UpdateRealServersRequest) (*UpdateRealServersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateRealServers not implemented")
}
func (UnimplementedIpvsAdminServer) mustEmbedUnimplementedIpvsAdminServer() {}

// UnsafeIpvsAdminServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to IpvsAdminServer will
// result in compilation errors.
type UnsafeIpvsAdminServer interface {
	mustEmbedUnimplementedIpvsAdminServer()
}

func RegisterIpvsAdminServer(s grpc.ServiceRegistrar, srv IpvsAdminServer) {
	s.RegisterService(&IpvsAdmin_ServiceDesc, srv)
}

func _IpvsAdmin_FindVirtualServer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FindVirtualServerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IpvsAdminServer).FindVirtualServer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/crispy.ipvs.IpvsAdmin/FindVirtualServer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IpvsAdminServer).FindVirtualServer(ctx, req.(*FindVirtualServerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IpvsAdmin_ListVirtualServers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListVirtualServersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IpvsAdminServer).ListVirtualServers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/crispy.ipvs.IpvsAdmin/ListVirtualServers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IpvsAdminServer).ListVirtualServers(ctx, req.(*ListVirtualServersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IpvsAdmin_UpdateVirtualServers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateVirtualServersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IpvsAdminServer).UpdateVirtualServers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/crispy.ipvs.IpvsAdmin/UpdateVirtualServers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IpvsAdminServer).UpdateVirtualServers(ctx, req.(*UpdateVirtualServersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IpvsAdmin_UpdateRealServers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateRealServersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IpvsAdminServer).UpdateRealServers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/crispy.ipvs.IpvsAdmin/UpdateRealServers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IpvsAdminServer).UpdateRealServers(ctx, req.(*UpdateRealServersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// IpvsAdmin_ServiceDesc is the grpc.ServiceDesc for IpvsAdmin service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var IpvsAdmin_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "crispy.ipvs.IpvsAdmin",
	HandlerType: (*IpvsAdminServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "FindVirtualServer",
			Handler:    _IpvsAdmin_FindVirtualServer_Handler,
		},
		{
			MethodName: "ListVirtualServers",
			Handler:    _IpvsAdmin_ListVirtualServers_Handler,
		},
		{
			MethodName: "UpdateVirtualServers",
			Handler:    _IpvsAdmin_UpdateVirtualServers_Handler,
		},
		{
			MethodName: "UpdateRealServers",
			Handler:    _IpvsAdmin_UpdateRealServers_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "ipvs/ipvs.proto",
}
