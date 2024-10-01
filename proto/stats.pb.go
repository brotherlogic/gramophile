// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.12.4
// source: stats.proto

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

type CollectionStats struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	FolderToCount map[int32]int32 `protobuf:"bytes,1,rep,name=folder_to_count,json=folderToCount,proto3" json:"folder_to_count,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

func (x *CollectionStats) Reset() {
	*x = CollectionStats{}
	if protoimpl.UnsafeEnabled {
		mi := &file_stats_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CollectionStats) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CollectionStats) ProtoMessage() {}

func (x *CollectionStats) ProtoReflect() protoreflect.Message {
	mi := &file_stats_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CollectionStats.ProtoReflect.Descriptor instead.
func (*CollectionStats) Descriptor() ([]byte, []int) {
	return file_stats_proto_rawDescGZIP(), []int{0}
}

func (x *CollectionStats) GetFolderToCount() map[int32]int32 {
	if x != nil {
		return x.FolderToCount
	}
	return nil
}

type SaleStats struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	YearTotals map[int32]int32 `protobuf:"bytes,1,rep,name=year_totals,json=yearTotals,proto3" json:"year_totals,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

func (x *SaleStats) Reset() {
	*x = SaleStats{}
	if protoimpl.UnsafeEnabled {
		mi := &file_stats_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SaleStats) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SaleStats) ProtoMessage() {}

func (x *SaleStats) ProtoReflect() protoreflect.Message {
	mi := &file_stats_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SaleStats.ProtoReflect.Descriptor instead.
func (*SaleStats) Descriptor() ([]byte, []int) {
	return file_stats_proto_rawDescGZIP(), []int{1}
}

func (x *SaleStats) GetYearTotals() map[int32]int32 {
	if x != nil {
		return x.YearTotals
	}
	return nil
}

var File_stats_proto protoreflect.FileDescriptor

var file_stats_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x73, 0x74, 0x61, 0x74, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x67,
	0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x22, 0xab, 0x01, 0x0a, 0x0f, 0x43, 0x6f,
	0x6c, 0x6c, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x73, 0x12, 0x56, 0x0a,
	0x0f, 0x66, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x5f, 0x74, 0x6f, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68,
	0x69, 0x6c, 0x65, 0x2e, 0x43, 0x6f, 0x6c, 0x6c, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74,
	0x61, 0x74, 0x73, 0x2e, 0x46, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x54, 0x6f, 0x43, 0x6f, 0x75, 0x6e,
	0x74, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0d, 0x66, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x54, 0x6f,
	0x43, 0x6f, 0x75, 0x6e, 0x74, 0x1a, 0x40, 0x0a, 0x12, 0x46, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x54,
	0x6f, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b,
	0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x92, 0x01, 0x0a, 0x09, 0x53, 0x61, 0x6c, 0x65,
	0x53, 0x74, 0x61, 0x74, 0x73, 0x12, 0x46, 0x0a, 0x0b, 0x79, 0x65, 0x61, 0x72, 0x5f, 0x74, 0x6f,
	0x74, 0x61, 0x6c, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x67, 0x72, 0x61,
	0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x53, 0x61, 0x6c, 0x65, 0x53, 0x74, 0x61, 0x74,
	0x73, 0x2e, 0x59, 0x65, 0x61, 0x72, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x52, 0x0a, 0x79, 0x65, 0x61, 0x72, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x73, 0x1a, 0x3d, 0x0a,
	0x0f, 0x59, 0x65, 0x61, 0x72, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x42, 0x2a, 0x5a, 0x28,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x72, 0x6f, 0x74, 0x68,
	0x65, 0x72, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x2f, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69,
	0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_stats_proto_rawDescOnce sync.Once
	file_stats_proto_rawDescData = file_stats_proto_rawDesc
)

func file_stats_proto_rawDescGZIP() []byte {
	file_stats_proto_rawDescOnce.Do(func() {
		file_stats_proto_rawDescData = protoimpl.X.CompressGZIP(file_stats_proto_rawDescData)
	})
	return file_stats_proto_rawDescData
}

var file_stats_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_stats_proto_goTypes = []interface{}{
	(*CollectionStats)(nil), // 0: gramophile.CollectionStats
	(*SaleStats)(nil),       // 1: gramophile.SaleStats
	nil,                     // 2: gramophile.CollectionStats.FolderToCountEntry
	nil,                     // 3: gramophile.SaleStats.YearTotalsEntry
}
var file_stats_proto_depIdxs = []int32{
	2, // 0: gramophile.CollectionStats.folder_to_count:type_name -> gramophile.CollectionStats.FolderToCountEntry
	3, // 1: gramophile.SaleStats.year_totals:type_name -> gramophile.SaleStats.YearTotalsEntry
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_stats_proto_init() }
func file_stats_proto_init() {
	if File_stats_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_stats_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CollectionStats); i {
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
		file_stats_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SaleStats); i {
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
			RawDescriptor: file_stats_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_stats_proto_goTypes,
		DependencyIndexes: file_stats_proto_depIdxs,
		MessageInfos:      file_stats_proto_msgTypes,
	}.Build()
	File_stats_proto = out.File
	file_stats_proto_rawDesc = nil
	file_stats_proto_goTypes = nil
	file_stats_proto_depIdxs = nil
}
