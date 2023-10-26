// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.12.4
// source: launchconfig.proto

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

type Enabled int32

const (
	Enabled_DISABLED Enabled = 0
	Enabled_ENABLED  Enabled = 1
)

// Enum value maps for Enabled.
var (
	Enabled_name = map[int32]string{
		0: "DISABLED",
		1: "ENABLED",
	}
	Enabled_value = map[string]int32{
		"DISABLED": 0,
		"ENABLED":  1,
	}
)

func (x Enabled) Enum() *Enabled {
	p := new(Enabled)
	*p = x
	return p
}

func (x Enabled) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Enabled) Descriptor() protoreflect.EnumDescriptor {
	return file_launchconfig_proto_enumTypes[0].Descriptor()
}

func (Enabled) Type() protoreflect.EnumType {
	return &file_launchconfig_proto_enumTypes[0]
}

func (x Enabled) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Enabled.Descriptor instead.
func (Enabled) EnumDescriptor() ([]byte, []int) {
	return file_launchconfig_proto_rawDescGZIP(), []int{0}
}

type LaunchConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	OrganisationConfig Enabled `protobuf:"varint,1,opt,name=organisation_config,json=organisationConfig,proto3,enum=gramophile.Enabled" json:"organisation_config,omitempty"`
	CleaningConfig     Enabled `protobuf:"varint,2,opt,name=cleaning_config,json=cleaningConfig,proto3,enum=gramophile.Enabled" json:"cleaning_config,omitempty"`
}

func (x *LaunchConfig) Reset() {
	*x = LaunchConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_launchconfig_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LaunchConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LaunchConfig) ProtoMessage() {}

func (x *LaunchConfig) ProtoReflect() protoreflect.Message {
	mi := &file_launchconfig_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LaunchConfig.ProtoReflect.Descriptor instead.
func (*LaunchConfig) Descriptor() ([]byte, []int) {
	return file_launchconfig_proto_rawDescGZIP(), []int{0}
}

func (x *LaunchConfig) GetOrganisationConfig() Enabled {
	if x != nil {
		return x.OrganisationConfig
	}
	return Enabled_DISABLED
}

func (x *LaunchConfig) GetCleaningConfig() Enabled {
	if x != nil {
		return x.CleaningConfig
	}
	return Enabled_DISABLED
}

var File_launchconfig_proto protoreflect.FileDescriptor

var file_launchconfig_proto_rawDesc = []byte{
	0x0a, 0x12, 0x6c, 0x61, 0x75, 0x6e, 0x63, 0x68, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65,
	0x22, 0x92, 0x01, 0x0a, 0x0c, 0x4c, 0x61, 0x75, 0x6e, 0x63, 0x68, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x12, 0x44, 0x0a, 0x13, 0x6f, 0x72, 0x67, 0x61, 0x6e, 0x69, 0x73, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x13,
	0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x45, 0x6e, 0x61, 0x62,
	0x6c, 0x65, 0x64, 0x52, 0x12, 0x6f, 0x72, 0x67, 0x61, 0x6e, 0x69, 0x73, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x3c, 0x0a, 0x0f, 0x63, 0x6c, 0x65, 0x61, 0x6e,
	0x69, 0x6e, 0x67, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e,
	0x32, 0x13, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x45, 0x6e,
	0x61, 0x62, 0x6c, 0x65, 0x64, 0x52, 0x0e, 0x63, 0x6c, 0x65, 0x61, 0x6e, 0x69, 0x6e, 0x67, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2a, 0x24, 0x0a, 0x07, 0x45, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64,
	0x12, 0x0c, 0x0a, 0x08, 0x44, 0x49, 0x53, 0x41, 0x42, 0x4c, 0x45, 0x44, 0x10, 0x00, 0x12, 0x0b,
	0x0a, 0x07, 0x45, 0x4e, 0x41, 0x42, 0x4c, 0x45, 0x44, 0x10, 0x01, 0x42, 0x2a, 0x5a, 0x28, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x72, 0x6f, 0x74, 0x68, 0x65,
	0x72, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x2f, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_launchconfig_proto_rawDescOnce sync.Once
	file_launchconfig_proto_rawDescData = file_launchconfig_proto_rawDesc
)

func file_launchconfig_proto_rawDescGZIP() []byte {
	file_launchconfig_proto_rawDescOnce.Do(func() {
		file_launchconfig_proto_rawDescData = protoimpl.X.CompressGZIP(file_launchconfig_proto_rawDescData)
	})
	return file_launchconfig_proto_rawDescData
}

var file_launchconfig_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_launchconfig_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_launchconfig_proto_goTypes = []interface{}{
	(Enabled)(0),         // 0: gramophile.Enabled
	(*LaunchConfig)(nil), // 1: gramophile.LaunchConfig
}
var file_launchconfig_proto_depIdxs = []int32{
	0, // 0: gramophile.LaunchConfig.organisation_config:type_name -> gramophile.Enabled
	0, // 1: gramophile.LaunchConfig.cleaning_config:type_name -> gramophile.Enabled
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_launchconfig_proto_init() }
func file_launchconfig_proto_init() {
	if File_launchconfig_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_launchconfig_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LaunchConfig); i {
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
			RawDescriptor: file_launchconfig_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_launchconfig_proto_goTypes,
		DependencyIndexes: file_launchconfig_proto_depIdxs,
		EnumInfos:         file_launchconfig_proto_enumTypes,
		MessageInfos:      file_launchconfig_proto_msgTypes,
	}.Build()
	File_launchconfig_proto = out.File
	file_launchconfig_proto_rawDesc = nil
	file_launchconfig_proto_goTypes = nil
	file_launchconfig_proto_depIdxs = nil
}
