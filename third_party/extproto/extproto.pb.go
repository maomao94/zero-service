// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.29.3
// source: extproto.proto

package extproto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type CurrentUser struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserId        string                 `protobuf:"bytes,1,opt,name=userId,proto3" json:"userId,omitempty"`
	UserName      string                 `protobuf:"bytes,2,opt,name=userName,proto3" json:"userName,omitempty"`
	TenantId      string                 `protobuf:"bytes,3,opt,name=tenantId,proto3" json:"tenantId,omitempty"`
	Metadata      map[string]string      `protobuf:"bytes,100,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Dept          []*Dept                `protobuf:"bytes,101,rep,name=dept,proto3" json:"dept,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CurrentUser) Reset() {
	*x = CurrentUser{}
	mi := &file_extproto_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CurrentUser) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CurrentUser) ProtoMessage() {}

func (x *CurrentUser) ProtoReflect() protoreflect.Message {
	mi := &file_extproto_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CurrentUser.ProtoReflect.Descriptor instead.
func (*CurrentUser) Descriptor() ([]byte, []int) {
	return file_extproto_proto_rawDescGZIP(), []int{0}
}

func (x *CurrentUser) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *CurrentUser) GetUserName() string {
	if x != nil {
		return x.UserName
	}
	return ""
}

func (x *CurrentUser) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}

func (x *CurrentUser) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *CurrentUser) GetDept() []*Dept {
	if x != nil {
		return x.Dept
	}
	return nil
}

type Dept struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	OrgId         string                 `protobuf:"bytes,1,opt,name=orgId,proto3" json:"orgId,omitempty"`
	OrgCode       string                 `protobuf:"bytes,2,opt,name=orgCode,proto3" json:"orgCode,omitempty"`
	OrgName       string                 `protobuf:"bytes,3,opt,name=orgName,proto3" json:"orgName,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Dept) Reset() {
	*x = Dept{}
	mi := &file_extproto_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Dept) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Dept) ProtoMessage() {}

func (x *Dept) ProtoReflect() protoreflect.Message {
	mi := &file_extproto_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Dept.ProtoReflect.Descriptor instead.
func (*Dept) Descriptor() ([]byte, []int) {
	return file_extproto_proto_rawDescGZIP(), []int{1}
}

func (x *Dept) GetOrgId() string {
	if x != nil {
		return x.OrgId
	}
	return ""
}

func (x *Dept) GetOrgCode() string {
	if x != nil {
		return x.OrgCode
	}
	return ""
}

func (x *Dept) GetOrgName() string {
	if x != nil {
		return x.OrgName
	}
	return ""
}

var File_extproto_proto protoreflect.FileDescriptor

const file_extproto_proto_rawDesc = "" +
	"\n" +
	"\x0eextproto.proto\x12\bextproto\"\xff\x01\n" +
	"\vCurrentUser\x12\x16\n" +
	"\x06userId\x18\x01 \x01(\tR\x06userId\x12\x1a\n" +
	"\buserName\x18\x02 \x01(\tR\buserName\x12\x1a\n" +
	"\btenantId\x18\x03 \x01(\tR\btenantId\x12?\n" +
	"\bmetadata\x18d \x03(\v2#.extproto.CurrentUser.MetadataEntryR\bmetadata\x12\"\n" +
	"\x04dept\x18e \x03(\v2\x0e.extproto.DeptR\x04dept\x1a;\n" +
	"\rMetadataEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"P\n" +
	"\x04Dept\x12\x14\n" +
	"\x05orgId\x18\x01 \x01(\tR\x05orgId\x12\x18\n" +
	"\aorgCode\x18\x02 \x01(\tR\aorgCode\x12\x18\n" +
	"\aorgName\x18\x03 \x01(\tR\aorgNameBM\n" +
	"\x13com.github.extprotoB\bExtProtoP\x01Z*zero-service/third_party/extproto;extprotob\x06proto3"

var (
	file_extproto_proto_rawDescOnce sync.Once
	file_extproto_proto_rawDescData []byte
)

func file_extproto_proto_rawDescGZIP() []byte {
	file_extproto_proto_rawDescOnce.Do(func() {
		file_extproto_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_extproto_proto_rawDesc), len(file_extproto_proto_rawDesc)))
	})
	return file_extproto_proto_rawDescData
}

var file_extproto_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_extproto_proto_goTypes = []any{
	(*CurrentUser)(nil), // 0: extproto.CurrentUser
	(*Dept)(nil),        // 1: extproto.Dept
	nil,                 // 2: extproto.CurrentUser.MetadataEntry
}
var file_extproto_proto_depIdxs = []int32{
	2, // 0: extproto.CurrentUser.metadata:type_name -> extproto.CurrentUser.MetadataEntry
	1, // 1: extproto.CurrentUser.dept:type_name -> extproto.Dept
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_extproto_proto_init() }
func file_extproto_proto_init() {
	if File_extproto_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_extproto_proto_rawDesc), len(file_extproto_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_extproto_proto_goTypes,
		DependencyIndexes: file_extproto_proto_depIdxs,
		MessageInfos:      file_extproto_proto_msgTypes,
	}.Build()
	File_extproto_proto = out.File
	file_extproto_proto_goTypes = nil
	file_extproto_proto_depIdxs = nil
}
