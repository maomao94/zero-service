// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v5.28.3
// source: ieccaller.proto

package ieccaller

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

type Req struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ping string `protobuf:"bytes,1,opt,name=ping,proto3" json:"ping,omitempty"`
}

func (x *Req) Reset() {
	*x = Req{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ieccaller_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Req) ProtoMessage() {}

func (x *Req) ProtoReflect() protoreflect.Message {
	mi := &file_ieccaller_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Req.ProtoReflect.Descriptor instead.
func (*Req) Descriptor() ([]byte, []int) {
	return file_ieccaller_proto_rawDescGZIP(), []int{0}
}

func (x *Req) GetPing() string {
	if x != nil {
		return x.Ping
	}
	return ""
}

type Res struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Pong string `protobuf:"bytes,1,opt,name=pong,proto3" json:"pong,omitempty"`
}

func (x *Res) Reset() {
	*x = Res{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ieccaller_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Res) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Res) ProtoMessage() {}

func (x *Res) ProtoReflect() protoreflect.Message {
	mi := &file_ieccaller_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Res.ProtoReflect.Descriptor instead.
func (*Res) Descriptor() ([]byte, []int) {
	return file_ieccaller_proto_rawDescGZIP(), []int{1}
}

func (x *Res) GetPong() string {
	if x != nil {
		return x.Pong
	}
	return ""
}

type SendTestCmdReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *SendTestCmdReq) Reset() {
	*x = SendTestCmdReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ieccaller_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SendTestCmdReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SendTestCmdReq) ProtoMessage() {}

func (x *SendTestCmdReq) ProtoReflect() protoreflect.Message {
	mi := &file_ieccaller_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SendTestCmdReq.ProtoReflect.Descriptor instead.
func (*SendTestCmdReq) Descriptor() ([]byte, []int) {
	return file_ieccaller_proto_rawDescGZIP(), []int{2}
}

type SendTestCmdRes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *SendTestCmdRes) Reset() {
	*x = SendTestCmdRes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ieccaller_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SendTestCmdRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SendTestCmdRes) ProtoMessage() {}

func (x *SendTestCmdRes) ProtoReflect() protoreflect.Message {
	mi := &file_ieccaller_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SendTestCmdRes.ProtoReflect.Descriptor instead.
func (*SendTestCmdRes) Descriptor() ([]byte, []int) {
	return file_ieccaller_proto_rawDescGZIP(), []int{3}
}

var File_ieccaller_proto protoreflect.FileDescriptor

var file_ieccaller_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x69, 0x65, 0x63, 0x63, 0x61, 0x6c, 0x6c, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x04, 0x66, 0x69, 0x6c, 0x65, 0x22, 0x19, 0x0a, 0x03, 0x52, 0x65, 0x71, 0x12, 0x12,
	0x0a, 0x04, 0x70, 0x69, 0x6e, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x69,
	0x6e, 0x67, 0x22, 0x19, 0x0a, 0x03, 0x52, 0x65, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x6f, 0x6e,
	0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x6f, 0x6e, 0x67, 0x22, 0x10, 0x0a,
	0x0e, 0x53, 0x65, 0x6e, 0x64, 0x54, 0x65, 0x73, 0x74, 0x43, 0x6d, 0x64, 0x52, 0x65, 0x71, 0x22,
	0x10, 0x0a, 0x0e, 0x53, 0x65, 0x6e, 0x64, 0x54, 0x65, 0x73, 0x74, 0x43, 0x6d, 0x64, 0x52, 0x65,
	0x73, 0x32, 0x64, 0x0a, 0x09, 0x49, 0x65, 0x63, 0x43, 0x61, 0x6c, 0x6c, 0x65, 0x72, 0x12, 0x1c,
	0x0a, 0x04, 0x50, 0x69, 0x6e, 0x67, 0x12, 0x09, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x52, 0x65,
	0x71, 0x1a, 0x09, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x52, 0x65, 0x73, 0x12, 0x39, 0x0a, 0x0b,
	0x53, 0x65, 0x6e, 0x64, 0x54, 0x65, 0x73, 0x74, 0x43, 0x6d, 0x64, 0x12, 0x14, 0x2e, 0x66, 0x69,
	0x6c, 0x65, 0x2e, 0x53, 0x65, 0x6e, 0x64, 0x54, 0x65, 0x73, 0x74, 0x43, 0x6d, 0x64, 0x52, 0x65,
	0x71, 0x1a, 0x14, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x53, 0x65, 0x6e, 0x64, 0x54, 0x65, 0x73,
	0x74, 0x43, 0x6d, 0x64, 0x52, 0x65, 0x73, 0x42, 0x0d, 0x5a, 0x0b, 0x2e, 0x2f, 0x69, 0x65, 0x63,
	0x63, 0x61, 0x6c, 0x6c, 0x65, 0x72, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_ieccaller_proto_rawDescOnce sync.Once
	file_ieccaller_proto_rawDescData = file_ieccaller_proto_rawDesc
)

func file_ieccaller_proto_rawDescGZIP() []byte {
	file_ieccaller_proto_rawDescOnce.Do(func() {
		file_ieccaller_proto_rawDescData = protoimpl.X.CompressGZIP(file_ieccaller_proto_rawDescData)
	})
	return file_ieccaller_proto_rawDescData
}

var file_ieccaller_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_ieccaller_proto_goTypes = []interface{}{
	(*Req)(nil),            // 0: file.Req
	(*Res)(nil),            // 1: file.Res
	(*SendTestCmdReq)(nil), // 2: file.SendTestCmdReq
	(*SendTestCmdRes)(nil), // 3: file.SendTestCmdRes
}
var file_ieccaller_proto_depIdxs = []int32{
	0, // 0: file.IecCaller.Ping:input_type -> file.Req
	2, // 1: file.IecCaller.SendTestCmd:input_type -> file.SendTestCmdReq
	1, // 2: file.IecCaller.Ping:output_type -> file.Res
	3, // 3: file.IecCaller.SendTestCmd:output_type -> file.SendTestCmdRes
	2, // [2:4] is the sub-list for method output_type
	0, // [0:2] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_ieccaller_proto_init() }
func file_ieccaller_proto_init() {
	if File_ieccaller_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_ieccaller_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Req); i {
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
		file_ieccaller_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Res); i {
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
		file_ieccaller_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SendTestCmdReq); i {
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
		file_ieccaller_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SendTestCmdRes); i {
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
			RawDescriptor: file_ieccaller_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_ieccaller_proto_goTypes,
		DependencyIndexes: file_ieccaller_proto_depIdxs,
		MessageInfos:      file_ieccaller_proto_msgTypes,
	}.Build()
	File_ieccaller_proto = out.File
	file_ieccaller_proto_rawDesc = nil
	file_ieccaller_proto_goTypes = nil
	file_ieccaller_proto_depIdxs = nil
}
