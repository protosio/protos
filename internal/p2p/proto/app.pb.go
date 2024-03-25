// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        (unknown)
// source: internal/p2p/proto/app.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type GetAppLogsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AppName string `protobuf:"bytes,1,opt,name=app_name,json=appName,proto3" json:"app_name,omitempty"`
}

func (x *GetAppLogsRequest) Reset() {
	*x = GetAppLogsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_p2p_proto_app_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetAppLogsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetAppLogsRequest) ProtoMessage() {}

func (x *GetAppLogsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_p2p_proto_app_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetAppLogsRequest.ProtoReflect.Descriptor instead.
func (*GetAppLogsRequest) Descriptor() ([]byte, []int) {
	return file_internal_p2p_proto_app_proto_rawDescGZIP(), []int{0}
}

func (x *GetAppLogsRequest) GetAppName() string {
	if x != nil {
		return x.AppName
	}
	return ""
}

type GetAppLogsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Logs string `protobuf:"bytes,1,opt,name=logs,proto3" json:"logs,omitempty"`
}

func (x *GetAppLogsResponse) Reset() {
	*x = GetAppLogsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_p2p_proto_app_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetAppLogsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetAppLogsResponse) ProtoMessage() {}

func (x *GetAppLogsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_p2p_proto_app_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetAppLogsResponse.ProtoReflect.Descriptor instead.
func (*GetAppLogsResponse) Descriptor() ([]byte, []int) {
	return file_internal_p2p_proto_app_proto_rawDescGZIP(), []int{1}
}

func (x *GetAppLogsResponse) GetLogs() string {
	if x != nil {
		return x.Logs
	}
	return ""
}

type GetAppStatusRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AppName string `protobuf:"bytes,1,opt,name=app_name,json=appName,proto3" json:"app_name,omitempty"`
}

func (x *GetAppStatusRequest) Reset() {
	*x = GetAppStatusRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_p2p_proto_app_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetAppStatusRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetAppStatusRequest) ProtoMessage() {}

func (x *GetAppStatusRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_p2p_proto_app_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetAppStatusRequest.ProtoReflect.Descriptor instead.
func (*GetAppStatusRequest) Descriptor() ([]byte, []int) {
	return file_internal_p2p_proto_app_proto_rawDescGZIP(), []int{2}
}

func (x *GetAppStatusRequest) GetAppName() string {
	if x != nil {
		return x.AppName
	}
	return ""
}

type GetAppStatusResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Status string `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
}

func (x *GetAppStatusResponse) Reset() {
	*x = GetAppStatusResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_p2p_proto_app_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetAppStatusResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetAppStatusResponse) ProtoMessage() {}

func (x *GetAppStatusResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_p2p_proto_app_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetAppStatusResponse.ProtoReflect.Descriptor instead.
func (*GetAppStatusResponse) Descriptor() ([]byte, []int) {
	return file_internal_p2p_proto_app_proto_rawDescGZIP(), []int{3}
}

func (x *GetAppStatusResponse) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

var File_internal_p2p_proto_app_proto protoreflect.FileDescriptor

var file_internal_p2p_proto_app_proto_rawDesc = []byte{
	0x0a, 0x1c, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x70, 0x32, 0x70, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x61, 0x70, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x2e, 0x0a, 0x11, 0x47, 0x65, 0x74, 0x41, 0x70, 0x70, 0x4c,
	0x6f, 0x67, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x19, 0x0a, 0x08, 0x61, 0x70,
	0x70, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x61, 0x70,
	0x70, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x28, 0x0a, 0x12, 0x47, 0x65, 0x74, 0x41, 0x70, 0x70, 0x4c,
	0x6f, 0x67, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6c,
	0x6f, 0x67, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6c, 0x6f, 0x67, 0x73, 0x22,
	0x30, 0x0a, 0x13, 0x47, 0x65, 0x74, 0x41, 0x70, 0x70, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x19, 0x0a, 0x08, 0x61, 0x70, 0x70, 0x5f, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x61, 0x70, 0x70, 0x4e, 0x61, 0x6d,
	0x65, 0x22, 0x2e, 0x0a, 0x14, 0x47, 0x65, 0x74, 0x41, 0x70, 0x70, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x32, 0x96, 0x01, 0x0a, 0x04, 0x41, 0x70, 0x70, 0x73, 0x12, 0x43, 0x0a, 0x0a, 0x47, 0x65,
	0x74, 0x41, 0x70, 0x70, 0x4c, 0x6f, 0x67, 0x73, 0x12, 0x18, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2e, 0x47, 0x65, 0x74, 0x41, 0x70, 0x70, 0x4c, 0x6f, 0x67, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x19, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x47, 0x65, 0x74, 0x41, 0x70,
	0x70, 0x4c, 0x6f, 0x67, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12,
	0x49, 0x0a, 0x0c, 0x47, 0x65, 0x74, 0x41, 0x70, 0x70, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12,
	0x1a, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x47, 0x65, 0x74, 0x41, 0x70, 0x70, 0x53, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1b, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x47, 0x65, 0x74, 0x41, 0x70, 0x70, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x09, 0x5a, 0x07, 0x2e, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_internal_p2p_proto_app_proto_rawDescOnce sync.Once
	file_internal_p2p_proto_app_proto_rawDescData = file_internal_p2p_proto_app_proto_rawDesc
)

func file_internal_p2p_proto_app_proto_rawDescGZIP() []byte {
	file_internal_p2p_proto_app_proto_rawDescOnce.Do(func() {
		file_internal_p2p_proto_app_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_p2p_proto_app_proto_rawDescData)
	})
	return file_internal_p2p_proto_app_proto_rawDescData
}

var file_internal_p2p_proto_app_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_internal_p2p_proto_app_proto_goTypes = []interface{}{
	(*GetAppLogsRequest)(nil),    // 0: proto.GetAppLogsRequest
	(*GetAppLogsResponse)(nil),   // 1: proto.GetAppLogsResponse
	(*GetAppStatusRequest)(nil),  // 2: proto.GetAppStatusRequest
	(*GetAppStatusResponse)(nil), // 3: proto.GetAppStatusResponse
}
var file_internal_p2p_proto_app_proto_depIdxs = []int32{
	0, // 0: proto.Apps.GetAppLogs:input_type -> proto.GetAppLogsRequest
	2, // 1: proto.Apps.GetAppStatus:input_type -> proto.GetAppStatusRequest
	1, // 2: proto.Apps.GetAppLogs:output_type -> proto.GetAppLogsResponse
	3, // 3: proto.Apps.GetAppStatus:output_type -> proto.GetAppStatusResponse
	2, // [2:4] is the sub-list for method output_type
	0, // [0:2] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_internal_p2p_proto_app_proto_init() }
func file_internal_p2p_proto_app_proto_init() {
	if File_internal_p2p_proto_app_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_internal_p2p_proto_app_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetAppLogsRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_p2p_proto_app_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetAppLogsResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_p2p_proto_app_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetAppStatusRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_p2p_proto_app_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetAppStatusResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_internal_p2p_proto_app_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_internal_p2p_proto_app_proto_goTypes,
		DependencyIndexes: file_internal_p2p_proto_app_proto_depIdxs,
		MessageInfos:      file_internal_p2p_proto_app_proto_msgTypes,
	}.Build()
	File_internal_p2p_proto_app_proto = out.File
	file_internal_p2p_proto_app_proto_rawDesc = nil
	file_internal_p2p_proto_app_proto_goTypes = nil
	file_internal_p2p_proto_app_proto_depIdxs = nil
}
