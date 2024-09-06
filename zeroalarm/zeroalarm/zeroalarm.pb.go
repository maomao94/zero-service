// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v5.27.0
// source: zeroalarm.proto

package zeroalarm

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
		mi := &file_zeroalarm_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Req) ProtoMessage() {}

func (x *Req) ProtoReflect() protoreflect.Message {
	mi := &file_zeroalarm_proto_msgTypes[0]
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
	return file_zeroalarm_proto_rawDescGZIP(), []int{0}
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
		mi := &file_zeroalarm_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Res) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Res) ProtoMessage() {}

func (x *Res) ProtoReflect() protoreflect.Message {
	mi := &file_zeroalarm_proto_msgTypes[1]
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
	return file_zeroalarm_proto_rawDescGZIP(), []int{1}
}

func (x *Res) GetPong() string {
	if x != nil {
		return x.Pong
	}
	return ""
}

type AlarmReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ChatName    string   `protobuf:"bytes,1,opt,name=chatName,proto3" json:"chatName,omitempty"`       // 服务告警 P0:线上事故处理 P1:线上事故处理 P2:线上事故处理 P3:线上事故处理 其他会新建群
	Description string   `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"` // 报警描述
	Title       string   `protobuf:"bytes,3,opt,name=title,proto3" json:"title,omitempty"`             // 报警标题
	Project     string   `protobuf:"bytes,4,opt,name=project,proto3" json:"project,omitempty"`         // 项目名称
	DateTime    string   `protobuf:"bytes,5,opt,name=dateTime,proto3" json:"dateTime,omitempty"`       // 2019-01-01 00:00:00
	AlarmId     string   `protobuf:"bytes,6,opt,name=alarmId,proto3" json:"alarmId,omitempty"`         // 唯一报警 id
	Content     string   `protobuf:"bytes,7,opt,name=content,proto3" json:"content,omitempty"`         // 报警内容
	Error       string   `protobuf:"bytes,8,opt,name=error,proto3" json:"error,omitempty"`             // 错误信息
	UserId      []string `protobuf:"bytes,9,rep,name=userId,proto3" json:"userId,omitempty"`           // 报警人 userId
	Ip          string   `protobuf:"bytes,10,opt,name=ip,proto3" json:"ip,omitempty"`                  // 报警 ip
}

func (x *AlarmReq) Reset() {
	*x = AlarmReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_zeroalarm_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AlarmReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AlarmReq) ProtoMessage() {}

func (x *AlarmReq) ProtoReflect() protoreflect.Message {
	mi := &file_zeroalarm_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AlarmReq.ProtoReflect.Descriptor instead.
func (*AlarmReq) Descriptor() ([]byte, []int) {
	return file_zeroalarm_proto_rawDescGZIP(), []int{2}
}

func (x *AlarmReq) GetChatName() string {
	if x != nil {
		return x.ChatName
	}
	return ""
}

func (x *AlarmReq) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *AlarmReq) GetTitle() string {
	if x != nil {
		return x.Title
	}
	return ""
}

func (x *AlarmReq) GetProject() string {
	if x != nil {
		return x.Project
	}
	return ""
}

func (x *AlarmReq) GetDateTime() string {
	if x != nil {
		return x.DateTime
	}
	return ""
}

func (x *AlarmReq) GetAlarmId() string {
	if x != nil {
		return x.AlarmId
	}
	return ""
}

func (x *AlarmReq) GetContent() string {
	if x != nil {
		return x.Content
	}
	return ""
}

func (x *AlarmReq) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

func (x *AlarmReq) GetUserId() []string {
	if x != nil {
		return x.UserId
	}
	return nil
}

func (x *AlarmReq) GetIp() string {
	if x != nil {
		return x.Ip
	}
	return ""
}

type AlarmRes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *AlarmRes) Reset() {
	*x = AlarmRes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_zeroalarm_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AlarmRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AlarmRes) ProtoMessage() {}

func (x *AlarmRes) ProtoReflect() protoreflect.Message {
	mi := &file_zeroalarm_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AlarmRes.ProtoReflect.Descriptor instead.
func (*AlarmRes) Descriptor() ([]byte, []int) {
	return file_zeroalarm_proto_rawDescGZIP(), []int{3}
}

var File_zeroalarm_proto protoreflect.FileDescriptor

var file_zeroalarm_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x7a, 0x65, 0x72, 0x6f, 0x61, 0x6c, 0x61, 0x72, 0x6d, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x09, 0x7a, 0x65, 0x72, 0x6f, 0x61, 0x6c, 0x61, 0x72, 0x6d, 0x22, 0x19, 0x0a, 0x03,
	0x52, 0x65, 0x71, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x69, 0x6e, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x70, 0x69, 0x6e, 0x67, 0x22, 0x19, 0x0a, 0x03, 0x52, 0x65, 0x73, 0x12, 0x12,
	0x0a, 0x04, 0x70, 0x6f, 0x6e, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x6f,
	0x6e, 0x67, 0x22, 0x86, 0x02, 0x0a, 0x08, 0x41, 0x6c, 0x61, 0x72, 0x6d, 0x52, 0x65, 0x71, 0x12,
	0x1a, 0x0a, 0x08, 0x63, 0x68, 0x61, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x63, 0x68, 0x61, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x20, 0x0a, 0x0b, 0x64,
	0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x14, 0x0a,
	0x05, 0x74, 0x69, 0x74, 0x6c, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x69,
	0x74, 0x6c, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x12, 0x1a, 0x0a,
	0x08, 0x64, 0x61, 0x74, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x08, 0x64, 0x61, 0x74, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x6c, 0x61,
	0x72, 0x6d, 0x49, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x61, 0x6c, 0x61, 0x72,
	0x6d, 0x49, 0x64, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x18, 0x07,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x12, 0x14, 0x0a,
	0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x08, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x72,
	0x72, 0x6f, 0x72, 0x12, 0x16, 0x0a, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x18, 0x09, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x12, 0x0e, 0x0a, 0x02, 0x69,
	0x70, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x70, 0x22, 0x0a, 0x0a, 0x08, 0x41,
	0x6c, 0x61, 0x72, 0x6d, 0x52, 0x65, 0x73, 0x32, 0x66, 0x0a, 0x09, 0x5a, 0x65, 0x72, 0x6f, 0x61,
	0x6c, 0x61, 0x72, 0x6d, 0x12, 0x26, 0x0a, 0x04, 0x50, 0x69, 0x6e, 0x67, 0x12, 0x0e, 0x2e, 0x7a,
	0x65, 0x72, 0x6f, 0x61, 0x6c, 0x61, 0x72, 0x6d, 0x2e, 0x52, 0x65, 0x71, 0x1a, 0x0e, 0x2e, 0x7a,
	0x65, 0x72, 0x6f, 0x61, 0x6c, 0x61, 0x72, 0x6d, 0x2e, 0x52, 0x65, 0x73, 0x12, 0x31, 0x0a, 0x05,
	0x41, 0x6c, 0x61, 0x72, 0x6d, 0x12, 0x13, 0x2e, 0x7a, 0x65, 0x72, 0x6f, 0x61, 0x6c, 0x61, 0x72,
	0x6d, 0x2e, 0x41, 0x6c, 0x61, 0x72, 0x6d, 0x52, 0x65, 0x71, 0x1a, 0x13, 0x2e, 0x7a, 0x65, 0x72,
	0x6f, 0x61, 0x6c, 0x61, 0x72, 0x6d, 0x2e, 0x41, 0x6c, 0x61, 0x72, 0x6d, 0x52, 0x65, 0x73, 0x42,
	0x0d, 0x5a, 0x0b, 0x2e, 0x2f, 0x7a, 0x65, 0x72, 0x6f, 0x61, 0x6c, 0x61, 0x72, 0x6d, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_zeroalarm_proto_rawDescOnce sync.Once
	file_zeroalarm_proto_rawDescData = file_zeroalarm_proto_rawDesc
)

func file_zeroalarm_proto_rawDescGZIP() []byte {
	file_zeroalarm_proto_rawDescOnce.Do(func() {
		file_zeroalarm_proto_rawDescData = protoimpl.X.CompressGZIP(file_zeroalarm_proto_rawDescData)
	})
	return file_zeroalarm_proto_rawDescData
}

var file_zeroalarm_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_zeroalarm_proto_goTypes = []interface{}{
	(*Req)(nil),      // 0: zeroalarm.Req
	(*Res)(nil),      // 1: zeroalarm.Res
	(*AlarmReq)(nil), // 2: zeroalarm.AlarmReq
	(*AlarmRes)(nil), // 3: zeroalarm.AlarmRes
}
var file_zeroalarm_proto_depIdxs = []int32{
	0, // 0: zeroalarm.Zeroalarm.Ping:input_type -> zeroalarm.Req
	2, // 1: zeroalarm.Zeroalarm.Alarm:input_type -> zeroalarm.AlarmReq
	1, // 2: zeroalarm.Zeroalarm.Ping:output_type -> zeroalarm.Res
	3, // 3: zeroalarm.Zeroalarm.Alarm:output_type -> zeroalarm.AlarmRes
	2, // [2:4] is the sub-list for method output_type
	0, // [0:2] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_zeroalarm_proto_init() }
func file_zeroalarm_proto_init() {
	if File_zeroalarm_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_zeroalarm_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
		file_zeroalarm_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
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
		file_zeroalarm_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AlarmReq); i {
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
		file_zeroalarm_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AlarmRes); i {
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
			RawDescriptor: file_zeroalarm_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_zeroalarm_proto_goTypes,
		DependencyIndexes: file_zeroalarm_proto_depIdxs,
		MessageInfos:      file_zeroalarm_proto_msgTypes,
	}.Build()
	File_zeroalarm_proto = out.File
	file_zeroalarm_proto_rawDesc = nil
	file_zeroalarm_proto_goTypes = nil
	file_zeroalarm_proto_depIdxs = nil
}
