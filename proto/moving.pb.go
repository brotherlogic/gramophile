// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v3.21.12
// source: moving.proto

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

type FormatSelector struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Format      string   `protobuf:"bytes,1,opt,name=format,proto3" json:"format,omitempty"`
	Description []string `protobuf:"bytes,2,rep,name=description,proto3" json:"description,omitempty"`
	Contains    []string `protobuf:"bytes,3,rep,name=contains,proto3" json:"contains,omitempty"`
	Order       int32    `protobuf:"varint,4,opt,name=order,proto3" json:"order,omitempty"`
}

func (x *FormatSelector) Reset() {
	*x = FormatSelector{}
	if protoimpl.UnsafeEnabled {
		mi := &file_moving_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FormatSelector) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FormatSelector) ProtoMessage() {}

func (x *FormatSelector) ProtoReflect() protoreflect.Message {
	mi := &file_moving_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FormatSelector.ProtoReflect.Descriptor instead.
func (*FormatSelector) Descriptor() ([]byte, []int) {
	return file_moving_proto_rawDescGZIP(), []int{0}
}

func (x *FormatSelector) GetFormat() string {
	if x != nil {
		return x.Format
	}
	return ""
}

func (x *FormatSelector) GetDescription() []string {
	if x != nil {
		return x.Description
	}
	return nil
}

func (x *FormatSelector) GetContains() []string {
	if x != nil {
		return x.Contains
	}
	return nil
}

func (x *FormatSelector) GetOrder() int32 {
	if x != nil {
		return x.Order
	}
	return 0
}

type FormatClassifier struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Formats       []*FormatSelector `protobuf:"bytes,1,rep,name=formats,proto3" json:"formats,omitempty"`
	DefaultFormat string            `protobuf:"bytes,2,opt,name=default_format,json=defaultFormat,proto3" json:"default_format,omitempty"`
}

func (x *FormatClassifier) Reset() {
	*x = FormatClassifier{}
	if protoimpl.UnsafeEnabled {
		mi := &file_moving_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FormatClassifier) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FormatClassifier) ProtoMessage() {}

func (x *FormatClassifier) ProtoReflect() protoreflect.Message {
	mi := &file_moving_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FormatClassifier.ProtoReflect.Descriptor instead.
func (*FormatClassifier) Descriptor() ([]byte, []int) {
	return file_moving_proto_rawDescGZIP(), []int{1}
}

func (x *FormatClassifier) GetFormats() []*FormatSelector {
	if x != nil {
		return x.Formats
	}
	return nil
}

func (x *FormatClassifier) GetDefaultFormat() string {
	if x != nil {
		return x.DefaultFormat
	}
	return ""
}

type RecordMove struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Classification []string `protobuf:"bytes,1,rep,name=classification,proto3" json:"classification,omitempty"`
	Format         []string `protobuf:"bytes,2,rep,name=format,proto3" json:"format,omitempty"`
	Folder         string   `protobuf:"bytes,3,opt,name=folder,proto3" json:"folder,omitempty"`
}

func (x *RecordMove) Reset() {
	*x = RecordMove{}
	if protoimpl.UnsafeEnabled {
		mi := &file_moving_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RecordMove) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RecordMove) ProtoMessage() {}

func (x *RecordMove) ProtoReflect() protoreflect.Message {
	mi := &file_moving_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RecordMove.ProtoReflect.Descriptor instead.
func (*RecordMove) Descriptor() ([]byte, []int) {
	return file_moving_proto_rawDescGZIP(), []int{2}
}

func (x *RecordMove) GetClassification() []string {
	if x != nil {
		return x.Classification
	}
	return nil
}

func (x *RecordMove) GetFormat() []string {
	if x != nil {
		return x.Format
	}
	return nil
}

func (x *RecordMove) GetFolder() string {
	if x != nil {
		return x.Folder
	}
	return ""
}

type MovingConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	FormatClassifier *FormatClassifier `protobuf:"bytes,1,opt,name=format_classifier,json=formatClassifier,proto3" json:"format_classifier,omitempty"`
	Moves            []*RecordMove     `protobuf:"bytes,2,rep,name=moves,proto3" json:"moves,omitempty"`
}

func (x *MovingConfig) Reset() {
	*x = MovingConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_moving_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MovingConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MovingConfig) ProtoMessage() {}

func (x *MovingConfig) ProtoReflect() protoreflect.Message {
	mi := &file_moving_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MovingConfig.ProtoReflect.Descriptor instead.
func (*MovingConfig) Descriptor() ([]byte, []int) {
	return file_moving_proto_rawDescGZIP(), []int{3}
}

func (x *MovingConfig) GetFormatClassifier() *FormatClassifier {
	if x != nil {
		return x.FormatClassifier
	}
	return nil
}

func (x *MovingConfig) GetMoves() []*RecordMove {
	if x != nil {
		return x.Moves
	}
	return nil
}

var File_moving_proto protoreflect.FileDescriptor

var file_moving_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x6d, 0x6f, 0x76, 0x69, 0x6e, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a,
	0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x22, 0x7c, 0x0a, 0x0e, 0x46, 0x6f,
	0x72, 0x6d, 0x61, 0x74, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x16, 0x0a, 0x06,
	0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x66, 0x6f,
	0x72, 0x6d, 0x61, 0x74, 0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72,
	0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69,
	0x6e, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x08, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69,
	0x6e, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x05, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x22, 0x6f, 0x0a, 0x10, 0x46, 0x6f, 0x72, 0x6d,
	0x61, 0x74, 0x43, 0x6c, 0x61, 0x73, 0x73, 0x69, 0x66, 0x69, 0x65, 0x72, 0x12, 0x34, 0x0a, 0x07,
	0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1a, 0x2e,
	0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x46, 0x6f, 0x72, 0x6d, 0x61,
	0x74, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x52, 0x07, 0x66, 0x6f, 0x72, 0x6d, 0x61,
	0x74, 0x73, 0x12, 0x25, 0x0a, 0x0e, 0x64, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x5f, 0x66, 0x6f,
	0x72, 0x6d, 0x61, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x64, 0x65, 0x66, 0x61,
	0x75, 0x6c, 0x74, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x22, 0x64, 0x0a, 0x0a, 0x52, 0x65, 0x63,
	0x6f, 0x72, 0x64, 0x4d, 0x6f, 0x76, 0x65, 0x12, 0x26, 0x0a, 0x0e, 0x63, 0x6c, 0x61, 0x73, 0x73,
	0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52,
	0x0e, 0x63, 0x6c, 0x61, 0x73, 0x73, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12,
	0x16, 0x0a, 0x06, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52,
	0x06, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x66, 0x6f, 0x6c, 0x64, 0x65,
	0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x66, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x22,
	0x87, 0x01, 0x0a, 0x0c, 0x4d, 0x6f, 0x76, 0x69, 0x6e, 0x67, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x12, 0x49, 0x0a, 0x11, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x5f, 0x63, 0x6c, 0x61, 0x73, 0x73,
	0x69, 0x66, 0x69, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x72,
	0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x43,
	0x6c, 0x61, 0x73, 0x73, 0x69, 0x66, 0x69, 0x65, 0x72, 0x52, 0x10, 0x66, 0x6f, 0x72, 0x6d, 0x61,
	0x74, 0x43, 0x6c, 0x61, 0x73, 0x73, 0x69, 0x66, 0x69, 0x65, 0x72, 0x12, 0x2c, 0x0a, 0x05, 0x6d,
	0x6f, 0x76, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x67, 0x72, 0x61,
	0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x4d, 0x6f,
	0x76, 0x65, 0x52, 0x05, 0x6d, 0x6f, 0x76, 0x65, 0x73, 0x42, 0x2a, 0x5a, 0x28, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x72, 0x6f, 0x74, 0x68, 0x65, 0x72, 0x6c,
	0x6f, 0x67, 0x69, 0x63, 0x2f, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_moving_proto_rawDescOnce sync.Once
	file_moving_proto_rawDescData = file_moving_proto_rawDesc
)

func file_moving_proto_rawDescGZIP() []byte {
	file_moving_proto_rawDescOnce.Do(func() {
		file_moving_proto_rawDescData = protoimpl.X.CompressGZIP(file_moving_proto_rawDescData)
	})
	return file_moving_proto_rawDescData
}

var file_moving_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_moving_proto_goTypes = []interface{}{
	(*FormatSelector)(nil),   // 0: gramophile.FormatSelector
	(*FormatClassifier)(nil), // 1: gramophile.FormatClassifier
	(*RecordMove)(nil),       // 2: gramophile.RecordMove
	(*MovingConfig)(nil),     // 3: gramophile.MovingConfig
}
var file_moving_proto_depIdxs = []int32{
	0, // 0: gramophile.FormatClassifier.formats:type_name -> gramophile.FormatSelector
	1, // 1: gramophile.MovingConfig.format_classifier:type_name -> gramophile.FormatClassifier
	2, // 2: gramophile.MovingConfig.moves:type_name -> gramophile.RecordMove
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_moving_proto_init() }
func file_moving_proto_init() {
	if File_moving_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_moving_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FormatSelector); i {
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
		file_moving_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FormatClassifier); i {
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
		file_moving_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RecordMove); i {
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
		file_moving_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MovingConfig); i {
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
			RawDescriptor: file_moving_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_moving_proto_goTypes,
		DependencyIndexes: file_moving_proto_depIdxs,
		MessageInfos:      file_moving_proto_msgTypes,
	}.Build()
	File_moving_proto = out.File
	file_moving_proto_rawDesc = nil
	file_moving_proto_goTypes = nil
	file_moving_proto_depIdxs = nil
}
