// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v5.29.2
// source: trigger.proto

package trigger

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
		mi := &file_trigger_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Req) ProtoMessage() {}

func (x *Req) ProtoReflect() protoreflect.Message {
	mi := &file_trigger_proto_msgTypes[0]
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
	return file_trigger_proto_rawDescGZIP(), []int{0}
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
		mi := &file_trigger_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Res) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Res) ProtoMessage() {}

func (x *Res) ProtoReflect() protoreflect.Message {
	mi := &file_trigger_proto_msgTypes[1]
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
	return file_trigger_proto_rawDescGZIP(), []int{1}
}

func (x *Res) GetPong() string {
	if x != nil {
		return x.Pong
	}
	return ""
}

type SendTriggerReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	MsgId       string `protobuf:"bytes,1,opt,name=msgId,proto3" json:"msgId,omitempty"`             // 唯一消息 id
	Body        string `protobuf:"bytes,2,opt,name=body,proto3" json:"body,omitempty"`               // 触发内容，可为空
	ProcessIn   int64  `protobuf:"varint,3,opt,name=processIn,proto3" json:"processIn,omitempty"`    // 秒
	TriggerTime string `protobuf:"bytes,4,opt,name=triggerTime,proto3" json:"triggerTime,omitempty"` // 触发时间 2019-01-01 00:00:00 二选一
	Url         string `protobuf:"bytes,5,opt,name=url,proto3" json:"url,omitempty"`                 // POST json提交
	MaxRetry    int64  `protobuf:"varint,6,opt,name=maxRetry,proto3" json:"maxRetry,omitempty"`      // 重试次数 默认 25
}

func (x *SendTriggerReq) Reset() {
	*x = SendTriggerReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_trigger_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SendTriggerReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SendTriggerReq) ProtoMessage() {}

func (x *SendTriggerReq) ProtoReflect() protoreflect.Message {
	mi := &file_trigger_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SendTriggerReq.ProtoReflect.Descriptor instead.
func (*SendTriggerReq) Descriptor() ([]byte, []int) {
	return file_trigger_proto_rawDescGZIP(), []int{2}
}

func (x *SendTriggerReq) GetMsgId() string {
	if x != nil {
		return x.MsgId
	}
	return ""
}

func (x *SendTriggerReq) GetBody() string {
	if x != nil {
		return x.Body
	}
	return ""
}

func (x *SendTriggerReq) GetProcessIn() int64 {
	if x != nil {
		return x.ProcessIn
	}
	return 0
}

func (x *SendTriggerReq) GetTriggerTime() string {
	if x != nil {
		return x.TriggerTime
	}
	return ""
}

func (x *SendTriggerReq) GetUrl() string {
	if x != nil {
		return x.Url
	}
	return ""
}

func (x *SendTriggerReq) GetMaxRetry() int64 {
	if x != nil {
		return x.MaxRetry
	}
	return 0
}

type SendTriggerRes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TraceId string `protobuf:"bytes,1,opt,name=traceId,proto3" json:"traceId,omitempty"` // 唯一追踪 id
	// ID is the identifier of the task.
	Id string `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	// Queue is the name of the queue in which the task belongs.
	Queue string `protobuf:"bytes,3,opt,name=queue,proto3" json:"queue,omitempty"`
}

func (x *SendTriggerRes) Reset() {
	*x = SendTriggerRes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_trigger_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SendTriggerRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SendTriggerRes) ProtoMessage() {}

func (x *SendTriggerRes) ProtoReflect() protoreflect.Message {
	mi := &file_trigger_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SendTriggerRes.ProtoReflect.Descriptor instead.
func (*SendTriggerRes) Descriptor() ([]byte, []int) {
	return file_trigger_proto_rawDescGZIP(), []int{3}
}

func (x *SendTriggerRes) GetTraceId() string {
	if x != nil {
		return x.TraceId
	}
	return ""
}

func (x *SendTriggerRes) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *SendTriggerRes) GetQueue() string {
	if x != nil {
		return x.Queue
	}
	return ""
}

var File_trigger_proto protoreflect.FileDescriptor

var file_trigger_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x74, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x07, 0x74, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x22, 0x19, 0x0a, 0x03, 0x52, 0x65, 0x71, 0x12,
	0x12, 0x0a, 0x04, 0x70, 0x69, 0x6e, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70,
	0x69, 0x6e, 0x67, 0x22, 0x19, 0x0a, 0x03, 0x52, 0x65, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x6f,
	0x6e, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x6f, 0x6e, 0x67, 0x22, 0xa8,
	0x01, 0x0a, 0x0e, 0x53, 0x65, 0x6e, 0x64, 0x54, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x52, 0x65,
	0x71, 0x12, 0x14, 0x0a, 0x05, 0x6d, 0x73, 0x67, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x6d, 0x73, 0x67, 0x49, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x12, 0x1c, 0x0a, 0x09, 0x70,
	0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x49, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09,
	0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x49, 0x6e, 0x12, 0x20, 0x0a, 0x0b, 0x74, 0x72, 0x69,
	0x67, 0x67, 0x65, 0x72, 0x54, 0x69, 0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b,
	0x74, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x75,
	0x72, 0x6c, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x75, 0x72, 0x6c, 0x12, 0x1a, 0x0a,
	0x08, 0x6d, 0x61, 0x78, 0x52, 0x65, 0x74, 0x72, 0x79, 0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x08, 0x6d, 0x61, 0x78, 0x52, 0x65, 0x74, 0x72, 0x79, 0x22, 0x50, 0x0a, 0x0e, 0x53, 0x65, 0x6e,
	0x64, 0x54, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x52, 0x65, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x74,
	0x72, 0x61, 0x63, 0x65, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x74, 0x72,
	0x61, 0x63, 0x65, 0x49, 0x64, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x71, 0x75, 0x65, 0x75, 0x65, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x71, 0x75, 0x65, 0x75, 0x65, 0x32, 0x71, 0x0a, 0x0a, 0x54,
	0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x52, 0x70, 0x63, 0x12, 0x22, 0x0a, 0x04, 0x50, 0x69, 0x6e,
	0x67, 0x12, 0x0c, 0x2e, 0x74, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x2e, 0x52, 0x65, 0x71, 0x1a,
	0x0c, 0x2e, 0x74, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x2e, 0x52, 0x65, 0x73, 0x12, 0x3f, 0x0a,
	0x0b, 0x53, 0x65, 0x6e, 0x64, 0x54, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x12, 0x17, 0x2e, 0x74,
	0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x2e, 0x53, 0x65, 0x6e, 0x64, 0x54, 0x72, 0x69, 0x67, 0x67,
	0x65, 0x72, 0x52, 0x65, 0x71, 0x1a, 0x17, 0x2e, 0x74, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x2e,
	0x53, 0x65, 0x6e, 0x64, 0x54, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x52, 0x65, 0x73, 0x42, 0x0b,
	0x5a, 0x09, 0x2e, 0x2f, 0x74, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_trigger_proto_rawDescOnce sync.Once
	file_trigger_proto_rawDescData = file_trigger_proto_rawDesc
)

func file_trigger_proto_rawDescGZIP() []byte {
	file_trigger_proto_rawDescOnce.Do(func() {
		file_trigger_proto_rawDescData = protoimpl.X.CompressGZIP(file_trigger_proto_rawDescData)
	})
	return file_trigger_proto_rawDescData
}

var file_trigger_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_trigger_proto_goTypes = []interface{}{
	(*Req)(nil),            // 0: trigger.Req
	(*Res)(nil),            // 1: trigger.Res
	(*SendTriggerReq)(nil), // 2: trigger.SendTriggerReq
	(*SendTriggerRes)(nil), // 3: trigger.SendTriggerRes
}
var file_trigger_proto_depIdxs = []int32{
	0, // 0: trigger.TriggerRpc.Ping:input_type -> trigger.Req
	2, // 1: trigger.TriggerRpc.SendTrigger:input_type -> trigger.SendTriggerReq
	1, // 2: trigger.TriggerRpc.Ping:output_type -> trigger.Res
	3, // 3: trigger.TriggerRpc.SendTrigger:output_type -> trigger.SendTriggerRes
	2, // [2:4] is the sub-list for method output_type
	0, // [0:2] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_trigger_proto_init() }
func file_trigger_proto_init() {
	if File_trigger_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_trigger_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
		file_trigger_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
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
		file_trigger_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SendTriggerReq); i {
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
		file_trigger_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SendTriggerRes); i {
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
			RawDescriptor: file_trigger_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_trigger_proto_goTypes,
		DependencyIndexes: file_trigger_proto_depIdxs,
		MessageInfos:      file_trigger_proto_msgTypes,
	}.Build()
	File_trigger_proto = out.File
	file_trigger_proto_rawDesc = nil
	file_trigger_proto_goTypes = nil
	file_trigger_proto_depIdxs = nil
}
