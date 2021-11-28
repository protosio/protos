// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package apic

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

// ProtosClientApiClient is the client API for ProtosClientApi service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ProtosClientApiClient interface {
	Init(ctx context.Context, in *InitRequest, opts ...grpc.CallOption) (*InitResponse, error)
	// App methods
	GetApps(ctx context.Context, in *GetAppsRequest, opts ...grpc.CallOption) (*GetAppsResponse, error)
	RunApp(ctx context.Context, in *RunAppRequest, opts ...grpc.CallOption) (*RunAppResponse, error)
	StartApp(ctx context.Context, in *StartAppRequest, opts ...grpc.CallOption) (*StartAppResponse, error)
	StopApp(ctx context.Context, in *StopAppRequest, opts ...grpc.CallOption) (*StopAppResponse, error)
	RemoveApp(ctx context.Context, in *RemoveAppRequest, opts ...grpc.CallOption) (*RemoveAppResponse, error)
	// App store methods
	GetInstallers(ctx context.Context, in *GetInstallersRequest, opts ...grpc.CallOption) (*GetInstallersResponse, error)
	GetInstaller(ctx context.Context, in *GetInstallerRequest, opts ...grpc.CallOption) (*GetInstallerResponse, error)
	// Cloud provider methods
	GetSupportedCloudProviders(ctx context.Context, in *GetSupportedCloudProvidersRequest, opts ...grpc.CallOption) (*GetSupportedCloudProvidersResponse, error)
	GetCloudProviders(ctx context.Context, in *GetCloudProvidersRequest, opts ...grpc.CallOption) (*GetCloudProvidersResponse, error)
	GetCloudProvider(ctx context.Context, in *GetCloudProviderRequest, opts ...grpc.CallOption) (*GetCloudProviderResponse, error)
	AddCloudProvider(ctx context.Context, in *AddCloudProviderRequest, opts ...grpc.CallOption) (*AddCloudProviderResponse, error)
	RemoveCloudProvider(ctx context.Context, in *RemoveCloudProviderRequest, opts ...grpc.CallOption) (*RemoveCloudProviderResponse, error)
	// Cloud instance methods
	GetInstances(ctx context.Context, in *GetInstancesRequest, opts ...grpc.CallOption) (*GetInstancesResponse, error)
	GetInstance(ctx context.Context, in *GetInstanceRequest, opts ...grpc.CallOption) (*GetInstanceResponse, error)
	DeployInstance(ctx context.Context, in *DeployInstanceRequest, opts ...grpc.CallOption) (*DeployInstanceResponse, error)
	RemoveInstance(ctx context.Context, in *RemoveInstanceRequest, opts ...grpc.CallOption) (*RemoveInstanceResponse, error)
	StartInstance(ctx context.Context, in *StartInstanceRequest, opts ...grpc.CallOption) (*StartInstanceResponse, error)
	StopInstance(ctx context.Context, in *StopInstanceRequest, opts ...grpc.CallOption) (*StopInstanceResponse, error)
	GetInstanceKey(ctx context.Context, in *GetInstanceKeyRequest, opts ...grpc.CallOption) (*GetInstanceKeyResponse, error)
	GetInstanceLogs(ctx context.Context, in *GetInstanceLogsRequest, opts ...grpc.CallOption) (*GetInstanceLogsResponse, error)
	InitDevInstance(ctx context.Context, in *InitDevInstanceRequest, opts ...grpc.CallOption) (*InitDevInstanceResponse, error)
}

type protosClientApiClient struct {
	cc grpc.ClientConnInterface
}

func NewProtosClientApiClient(cc grpc.ClientConnInterface) ProtosClientApiClient {
	return &protosClientApiClient{cc}
}

func (c *protosClientApiClient) Init(ctx context.Context, in *InitRequest, opts ...grpc.CallOption) (*InitResponse, error) {
	out := new(InitResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/Init", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) GetApps(ctx context.Context, in *GetAppsRequest, opts ...grpc.CallOption) (*GetAppsResponse, error) {
	out := new(GetAppsResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/GetApps", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) RunApp(ctx context.Context, in *RunAppRequest, opts ...grpc.CallOption) (*RunAppResponse, error) {
	out := new(RunAppResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/RunApp", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) StartApp(ctx context.Context, in *StartAppRequest, opts ...grpc.CallOption) (*StartAppResponse, error) {
	out := new(StartAppResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/StartApp", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) StopApp(ctx context.Context, in *StopAppRequest, opts ...grpc.CallOption) (*StopAppResponse, error) {
	out := new(StopAppResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/StopApp", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) RemoveApp(ctx context.Context, in *RemoveAppRequest, opts ...grpc.CallOption) (*RemoveAppResponse, error) {
	out := new(RemoveAppResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/RemoveApp", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) GetInstallers(ctx context.Context, in *GetInstallersRequest, opts ...grpc.CallOption) (*GetInstallersResponse, error) {
	out := new(GetInstallersResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/GetInstallers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) GetInstaller(ctx context.Context, in *GetInstallerRequest, opts ...grpc.CallOption) (*GetInstallerResponse, error) {
	out := new(GetInstallerResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/GetInstaller", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) GetSupportedCloudProviders(ctx context.Context, in *GetSupportedCloudProvidersRequest, opts ...grpc.CallOption) (*GetSupportedCloudProvidersResponse, error) {
	out := new(GetSupportedCloudProvidersResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/GetSupportedCloudProviders", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) GetCloudProviders(ctx context.Context, in *GetCloudProvidersRequest, opts ...grpc.CallOption) (*GetCloudProvidersResponse, error) {
	out := new(GetCloudProvidersResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/GetCloudProviders", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) GetCloudProvider(ctx context.Context, in *GetCloudProviderRequest, opts ...grpc.CallOption) (*GetCloudProviderResponse, error) {
	out := new(GetCloudProviderResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/GetCloudProvider", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) AddCloudProvider(ctx context.Context, in *AddCloudProviderRequest, opts ...grpc.CallOption) (*AddCloudProviderResponse, error) {
	out := new(AddCloudProviderResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/AddCloudProvider", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) RemoveCloudProvider(ctx context.Context, in *RemoveCloudProviderRequest, opts ...grpc.CallOption) (*RemoveCloudProviderResponse, error) {
	out := new(RemoveCloudProviderResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/RemoveCloudProvider", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) GetInstances(ctx context.Context, in *GetInstancesRequest, opts ...grpc.CallOption) (*GetInstancesResponse, error) {
	out := new(GetInstancesResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/GetInstances", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) GetInstance(ctx context.Context, in *GetInstanceRequest, opts ...grpc.CallOption) (*GetInstanceResponse, error) {
	out := new(GetInstanceResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/GetInstance", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) DeployInstance(ctx context.Context, in *DeployInstanceRequest, opts ...grpc.CallOption) (*DeployInstanceResponse, error) {
	out := new(DeployInstanceResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/DeployInstance", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) RemoveInstance(ctx context.Context, in *RemoveInstanceRequest, opts ...grpc.CallOption) (*RemoveInstanceResponse, error) {
	out := new(RemoveInstanceResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/RemoveInstance", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) StartInstance(ctx context.Context, in *StartInstanceRequest, opts ...grpc.CallOption) (*StartInstanceResponse, error) {
	out := new(StartInstanceResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/StartInstance", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) StopInstance(ctx context.Context, in *StopInstanceRequest, opts ...grpc.CallOption) (*StopInstanceResponse, error) {
	out := new(StopInstanceResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/StopInstance", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) GetInstanceKey(ctx context.Context, in *GetInstanceKeyRequest, opts ...grpc.CallOption) (*GetInstanceKeyResponse, error) {
	out := new(GetInstanceKeyResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/GetInstanceKey", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) GetInstanceLogs(ctx context.Context, in *GetInstanceLogsRequest, opts ...grpc.CallOption) (*GetInstanceLogsResponse, error) {
	out := new(GetInstanceLogsResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/GetInstanceLogs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *protosClientApiClient) InitDevInstance(ctx context.Context, in *InitDevInstanceRequest, opts ...grpc.CallOption) (*InitDevInstanceResponse, error) {
	out := new(InitDevInstanceResponse)
	err := c.cc.Invoke(ctx, "/apic.ProtosClientApi/InitDevInstance", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ProtosClientApiServer is the server API for ProtosClientApi service.
// All implementations must embed UnimplementedProtosClientApiServer
// for forward compatibility
type ProtosClientApiServer interface {
	Init(context.Context, *InitRequest) (*InitResponse, error)
	// App methods
	GetApps(context.Context, *GetAppsRequest) (*GetAppsResponse, error)
	RunApp(context.Context, *RunAppRequest) (*RunAppResponse, error)
	StartApp(context.Context, *StartAppRequest) (*StartAppResponse, error)
	StopApp(context.Context, *StopAppRequest) (*StopAppResponse, error)
	RemoveApp(context.Context, *RemoveAppRequest) (*RemoveAppResponse, error)
	// App store methods
	GetInstallers(context.Context, *GetInstallersRequest) (*GetInstallersResponse, error)
	GetInstaller(context.Context, *GetInstallerRequest) (*GetInstallerResponse, error)
	// Cloud provider methods
	GetSupportedCloudProviders(context.Context, *GetSupportedCloudProvidersRequest) (*GetSupportedCloudProvidersResponse, error)
	GetCloudProviders(context.Context, *GetCloudProvidersRequest) (*GetCloudProvidersResponse, error)
	GetCloudProvider(context.Context, *GetCloudProviderRequest) (*GetCloudProviderResponse, error)
	AddCloudProvider(context.Context, *AddCloudProviderRequest) (*AddCloudProviderResponse, error)
	RemoveCloudProvider(context.Context, *RemoveCloudProviderRequest) (*RemoveCloudProviderResponse, error)
	// Cloud instance methods
	GetInstances(context.Context, *GetInstancesRequest) (*GetInstancesResponse, error)
	GetInstance(context.Context, *GetInstanceRequest) (*GetInstanceResponse, error)
	DeployInstance(context.Context, *DeployInstanceRequest) (*DeployInstanceResponse, error)
	RemoveInstance(context.Context, *RemoveInstanceRequest) (*RemoveInstanceResponse, error)
	StartInstance(context.Context, *StartInstanceRequest) (*StartInstanceResponse, error)
	StopInstance(context.Context, *StopInstanceRequest) (*StopInstanceResponse, error)
	GetInstanceKey(context.Context, *GetInstanceKeyRequest) (*GetInstanceKeyResponse, error)
	GetInstanceLogs(context.Context, *GetInstanceLogsRequest) (*GetInstanceLogsResponse, error)
	InitDevInstance(context.Context, *InitDevInstanceRequest) (*InitDevInstanceResponse, error)
	mustEmbedUnimplementedProtosClientApiServer()
}

// UnimplementedProtosClientApiServer must be embedded to have forward compatible implementations.
type UnimplementedProtosClientApiServer struct {
}

func (UnimplementedProtosClientApiServer) Init(context.Context, *InitRequest) (*InitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Init not implemented")
}
func (UnimplementedProtosClientApiServer) GetApps(context.Context, *GetAppsRequest) (*GetAppsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetApps not implemented")
}
func (UnimplementedProtosClientApiServer) RunApp(context.Context, *RunAppRequest) (*RunAppResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RunApp not implemented")
}
func (UnimplementedProtosClientApiServer) StartApp(context.Context, *StartAppRequest) (*StartAppResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartApp not implemented")
}
func (UnimplementedProtosClientApiServer) StopApp(context.Context, *StopAppRequest) (*StopAppResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StopApp not implemented")
}
func (UnimplementedProtosClientApiServer) RemoveApp(context.Context, *RemoveAppRequest) (*RemoveAppResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveApp not implemented")
}
func (UnimplementedProtosClientApiServer) GetInstallers(context.Context, *GetInstallersRequest) (*GetInstallersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetInstallers not implemented")
}
func (UnimplementedProtosClientApiServer) GetInstaller(context.Context, *GetInstallerRequest) (*GetInstallerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetInstaller not implemented")
}
func (UnimplementedProtosClientApiServer) GetSupportedCloudProviders(context.Context, *GetSupportedCloudProvidersRequest) (*GetSupportedCloudProvidersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetSupportedCloudProviders not implemented")
}
func (UnimplementedProtosClientApiServer) GetCloudProviders(context.Context, *GetCloudProvidersRequest) (*GetCloudProvidersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCloudProviders not implemented")
}
func (UnimplementedProtosClientApiServer) GetCloudProvider(context.Context, *GetCloudProviderRequest) (*GetCloudProviderResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCloudProvider not implemented")
}
func (UnimplementedProtosClientApiServer) AddCloudProvider(context.Context, *AddCloudProviderRequest) (*AddCloudProviderResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddCloudProvider not implemented")
}
func (UnimplementedProtosClientApiServer) RemoveCloudProvider(context.Context, *RemoveCloudProviderRequest) (*RemoveCloudProviderResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveCloudProvider not implemented")
}
func (UnimplementedProtosClientApiServer) GetInstances(context.Context, *GetInstancesRequest) (*GetInstancesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetInstances not implemented")
}
func (UnimplementedProtosClientApiServer) GetInstance(context.Context, *GetInstanceRequest) (*GetInstanceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetInstance not implemented")
}
func (UnimplementedProtosClientApiServer) DeployInstance(context.Context, *DeployInstanceRequest) (*DeployInstanceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeployInstance not implemented")
}
func (UnimplementedProtosClientApiServer) RemoveInstance(context.Context, *RemoveInstanceRequest) (*RemoveInstanceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveInstance not implemented")
}
func (UnimplementedProtosClientApiServer) StartInstance(context.Context, *StartInstanceRequest) (*StartInstanceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartInstance not implemented")
}
func (UnimplementedProtosClientApiServer) StopInstance(context.Context, *StopInstanceRequest) (*StopInstanceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StopInstance not implemented")
}
func (UnimplementedProtosClientApiServer) GetInstanceKey(context.Context, *GetInstanceKeyRequest) (*GetInstanceKeyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetInstanceKey not implemented")
}
func (UnimplementedProtosClientApiServer) GetInstanceLogs(context.Context, *GetInstanceLogsRequest) (*GetInstanceLogsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetInstanceLogs not implemented")
}
func (UnimplementedProtosClientApiServer) InitDevInstance(context.Context, *InitDevInstanceRequest) (*InitDevInstanceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InitDevInstance not implemented")
}
func (UnimplementedProtosClientApiServer) mustEmbedUnimplementedProtosClientApiServer() {}

// UnsafeProtosClientApiServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ProtosClientApiServer will
// result in compilation errors.
type UnsafeProtosClientApiServer interface {
	mustEmbedUnimplementedProtosClientApiServer()
}

func RegisterProtosClientApiServer(s grpc.ServiceRegistrar, srv ProtosClientApiServer) {
	s.RegisterService(&ProtosClientApi_ServiceDesc, srv)
}

func _ProtosClientApi_Init_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).Init(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/Init",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).Init(ctx, req.(*InitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_GetApps_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetAppsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).GetApps(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/GetApps",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).GetApps(ctx, req.(*GetAppsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_RunApp_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RunAppRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).RunApp(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/RunApp",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).RunApp(ctx, req.(*RunAppRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_StartApp_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartAppRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).StartApp(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/StartApp",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).StartApp(ctx, req.(*StartAppRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_StopApp_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StopAppRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).StopApp(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/StopApp",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).StopApp(ctx, req.(*StopAppRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_RemoveApp_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveAppRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).RemoveApp(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/RemoveApp",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).RemoveApp(ctx, req.(*RemoveAppRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_GetInstallers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetInstallersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).GetInstallers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/GetInstallers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).GetInstallers(ctx, req.(*GetInstallersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_GetInstaller_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetInstallerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).GetInstaller(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/GetInstaller",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).GetInstaller(ctx, req.(*GetInstallerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_GetSupportedCloudProviders_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetSupportedCloudProvidersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).GetSupportedCloudProviders(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/GetSupportedCloudProviders",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).GetSupportedCloudProviders(ctx, req.(*GetSupportedCloudProvidersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_GetCloudProviders_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetCloudProvidersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).GetCloudProviders(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/GetCloudProviders",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).GetCloudProviders(ctx, req.(*GetCloudProvidersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_GetCloudProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetCloudProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).GetCloudProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/GetCloudProvider",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).GetCloudProvider(ctx, req.(*GetCloudProviderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_AddCloudProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddCloudProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).AddCloudProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/AddCloudProvider",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).AddCloudProvider(ctx, req.(*AddCloudProviderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_RemoveCloudProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveCloudProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).RemoveCloudProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/RemoveCloudProvider",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).RemoveCloudProvider(ctx, req.(*RemoveCloudProviderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_GetInstances_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetInstancesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).GetInstances(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/GetInstances",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).GetInstances(ctx, req.(*GetInstancesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_GetInstance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetInstanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).GetInstance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/GetInstance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).GetInstance(ctx, req.(*GetInstanceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_DeployInstance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeployInstanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).DeployInstance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/DeployInstance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).DeployInstance(ctx, req.(*DeployInstanceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_RemoveInstance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveInstanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).RemoveInstance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/RemoveInstance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).RemoveInstance(ctx, req.(*RemoveInstanceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_StartInstance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartInstanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).StartInstance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/StartInstance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).StartInstance(ctx, req.(*StartInstanceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_StopInstance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StopInstanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).StopInstance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/StopInstance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).StopInstance(ctx, req.(*StopInstanceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_GetInstanceKey_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetInstanceKeyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).GetInstanceKey(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/GetInstanceKey",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).GetInstanceKey(ctx, req.(*GetInstanceKeyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_GetInstanceLogs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetInstanceLogsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).GetInstanceLogs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/GetInstanceLogs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).GetInstanceLogs(ctx, req.(*GetInstanceLogsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProtosClientApi_InitDevInstance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitDevInstanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProtosClientApiServer).InitDevInstance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apic.ProtosClientApi/InitDevInstance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProtosClientApiServer).InitDevInstance(ctx, req.(*InitDevInstanceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ProtosClientApi_ServiceDesc is the grpc.ServiceDesc for ProtosClientApi service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ProtosClientApi_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "apic.ProtosClientApi",
	HandlerType: (*ProtosClientApiServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Init",
			Handler:    _ProtosClientApi_Init_Handler,
		},
		{
			MethodName: "GetApps",
			Handler:    _ProtosClientApi_GetApps_Handler,
		},
		{
			MethodName: "RunApp",
			Handler:    _ProtosClientApi_RunApp_Handler,
		},
		{
			MethodName: "StartApp",
			Handler:    _ProtosClientApi_StartApp_Handler,
		},
		{
			MethodName: "StopApp",
			Handler:    _ProtosClientApi_StopApp_Handler,
		},
		{
			MethodName: "RemoveApp",
			Handler:    _ProtosClientApi_RemoveApp_Handler,
		},
		{
			MethodName: "GetInstallers",
			Handler:    _ProtosClientApi_GetInstallers_Handler,
		},
		{
			MethodName: "GetInstaller",
			Handler:    _ProtosClientApi_GetInstaller_Handler,
		},
		{
			MethodName: "GetSupportedCloudProviders",
			Handler:    _ProtosClientApi_GetSupportedCloudProviders_Handler,
		},
		{
			MethodName: "GetCloudProviders",
			Handler:    _ProtosClientApi_GetCloudProviders_Handler,
		},
		{
			MethodName: "GetCloudProvider",
			Handler:    _ProtosClientApi_GetCloudProvider_Handler,
		},
		{
			MethodName: "AddCloudProvider",
			Handler:    _ProtosClientApi_AddCloudProvider_Handler,
		},
		{
			MethodName: "RemoveCloudProvider",
			Handler:    _ProtosClientApi_RemoveCloudProvider_Handler,
		},
		{
			MethodName: "GetInstances",
			Handler:    _ProtosClientApi_GetInstances_Handler,
		},
		{
			MethodName: "GetInstance",
			Handler:    _ProtosClientApi_GetInstance_Handler,
		},
		{
			MethodName: "DeployInstance",
			Handler:    _ProtosClientApi_DeployInstance_Handler,
		},
		{
			MethodName: "RemoveInstance",
			Handler:    _ProtosClientApi_RemoveInstance_Handler,
		},
		{
			MethodName: "StartInstance",
			Handler:    _ProtosClientApi_StartInstance_Handler,
		},
		{
			MethodName: "StopInstance",
			Handler:    _ProtosClientApi_StopInstance_Handler,
		},
		{
			MethodName: "GetInstanceKey",
			Handler:    _ProtosClientApi_GetInstanceKey_Handler,
		},
		{
			MethodName: "GetInstanceLogs",
			Handler:    _ProtosClientApi_GetInstanceLogs_Handler,
		},
		{
			MethodName: "InitDevInstance",
			Handler:    _ProtosClientApi_InitDevInstance_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "apic.proto",
}
