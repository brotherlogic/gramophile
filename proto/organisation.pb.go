// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.12.4
// source: organisation.proto

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

type Sort int32

const (
	Sort_ARTIST_YEAR Sort = 0
)

// Enum value maps for Sort.
var (
	Sort_name = map[int32]string{
		0: "ARTIST_YEAR",
	}
	Sort_value = map[string]int32{
		"ARTIST_YEAR": 0,
	}
)

func (x Sort) Enum() *Sort {
	p := new(Sort)
	*p = x
	return p
}

func (x Sort) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Sort) Descriptor() protoreflect.EnumDescriptor {
	return file_organisation_proto_enumTypes[0].Descriptor()
}

func (Sort) Type() protoreflect.EnumType {
	return &file_organisation_proto_enumTypes[0]
}

func (x Sort) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Sort.Descriptor instead.
func (Sort) EnumDescriptor() ([]byte, []int) {
	return file_organisation_proto_rawDescGZIP(), []int{0}
}

type Organisation struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Foldersets          []*FolderSet         `protobuf:"bytes,1,rep,name=foldersets,proto3" json:"foldersets,omitempty"`
	Spaces              []*Space             `protobuf:"bytes,2,rep,name=spaces,proto3" json:"spaces,omitempty"`
	LabelDeciders       []*LabelDecider      `protobuf:"bytes,3,rep,name=label_deciders,json=labelDeciders,proto3" json:"label_deciders,omitempty"`
	ArtistTranslation   []*ArtistTranslation `protobuf:"bytes,4,rep,name=artist_translation,json=artistTranslation,proto3" json:"artist_translation,omitempty"`
	AutoArtistTranslate bool                 `protobuf:"varint,5,opt,name=auto_artist_translate,json=autoArtistTranslate,proto3" json:"auto_artist_translate,omitempty"`
}

func (x *Organisation) Reset() {
	*x = Organisation{}
	if protoimpl.UnsafeEnabled {
		mi := &file_organisation_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Organisation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Organisation) ProtoMessage() {}

func (x *Organisation) ProtoReflect() protoreflect.Message {
	mi := &file_organisation_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Organisation.ProtoReflect.Descriptor instead.
func (*Organisation) Descriptor() ([]byte, []int) {
	return file_organisation_proto_rawDescGZIP(), []int{0}
}

func (x *Organisation) GetFoldersets() []*FolderSet {
	if x != nil {
		return x.Foldersets
	}
	return nil
}

func (x *Organisation) GetSpaces() []*Space {
	if x != nil {
		return x.Spaces
	}
	return nil
}

func (x *Organisation) GetLabelDeciders() []*LabelDecider {
	if x != nil {
		return x.LabelDeciders
	}
	return nil
}

func (x *Organisation) GetArtistTranslation() []*ArtistTranslation {
	if x != nil {
		return x.ArtistTranslation
	}
	return nil
}

func (x *Organisation) GetAutoArtistTranslate() bool {
	if x != nil {
		return x.AutoArtistTranslate
	}
	return false
}

type Space struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Index int32 `protobuf:"varint,1,opt,name=index,proto3" json:"index,omitempty"`
	Units int32 `protobuf:"varint,2,opt,name=units,proto3" json:"units,omitempty"`
	// Effective one_of
	RecordsWidth int32    `protobuf:"varint,3,opt,name=records_width,json=recordsWidth,proto3" json:"records_width,omitempty"`
	DisksWidth   int32    `protobuf:"varint,4,opt,name=disks_width,json=disksWidth,proto3" json:"disks_width,omitempty"`
	Width        float32  `protobuf:"fixed32,5,opt,name=width,proto3" json:"width,omitempty"`
	FolderSets   []string `protobuf:"bytes,6,rep,name=folder_sets,json=folderSets,proto3" json:"folder_sets,omitempty"`
}

func (x *Space) Reset() {
	*x = Space{}
	if protoimpl.UnsafeEnabled {
		mi := &file_organisation_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Space) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Space) ProtoMessage() {}

func (x *Space) ProtoReflect() protoreflect.Message {
	mi := &file_organisation_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Space.ProtoReflect.Descriptor instead.
func (*Space) Descriptor() ([]byte, []int) {
	return file_organisation_proto_rawDescGZIP(), []int{1}
}

func (x *Space) GetIndex() int32 {
	if x != nil {
		return x.Index
	}
	return 0
}

func (x *Space) GetUnits() int32 {
	if x != nil {
		return x.Units
	}
	return 0
}

func (x *Space) GetRecordsWidth() int32 {
	if x != nil {
		return x.RecordsWidth
	}
	return 0
}

func (x *Space) GetDisksWidth() int32 {
	if x != nil {
		return x.DisksWidth
	}
	return 0
}

func (x *Space) GetWidth() float32 {
	if x != nil {
		return x.Width
	}
	return 0
}

func (x *Space) GetFolderSets() []string {
	if x != nil {
		return x.FolderSets
	}
	return nil
}

type FolderSet struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name   string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Index  int32  `protobuf:"varint,2,opt,name=index,proto3" json:"index,omitempty"`
	Folder int32  `protobuf:"varint,3,opt,name=folder,proto3" json:"folder,omitempty"`
	Sort   Sort   `protobuf:"varint,4,opt,name=sort,proto3,enum=gramophile.Sort" json:"sort,omitempty"`
}

func (x *FolderSet) Reset() {
	*x = FolderSet{}
	if protoimpl.UnsafeEnabled {
		mi := &file_organisation_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FolderSet) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FolderSet) ProtoMessage() {}

func (x *FolderSet) ProtoReflect() protoreflect.Message {
	mi := &file_organisation_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FolderSet.ProtoReflect.Descriptor instead.
func (*FolderSet) Descriptor() ([]byte, []int) {
	return file_organisation_proto_rawDescGZIP(), []int{2}
}

func (x *FolderSet) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *FolderSet) GetIndex() int32 {
	if x != nil {
		return x.Index
	}
	return 0
}

func (x *FolderSet) GetFolder() int32 {
	if x != nil {
		return x.Folder
	}
	return 0
}

func (x *FolderSet) GetSort() Sort {
	if x != nil {
		return x.Sort
	}
	return Sort_ARTIST_YEAR
}

type LabelDecider struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Index       int32  `protobuf:"varint,1,opt,name=index,proto3" json:"index,omitempty"`
	LabelPrefix string `protobuf:"bytes,2,opt,name=label_prefix,json=labelPrefix,proto3" json:"label_prefix,omitempty"`
}

func (x *LabelDecider) Reset() {
	*x = LabelDecider{}
	if protoimpl.UnsafeEnabled {
		mi := &file_organisation_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LabelDecider) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LabelDecider) ProtoMessage() {}

func (x *LabelDecider) ProtoReflect() protoreflect.Message {
	mi := &file_organisation_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LabelDecider.ProtoReflect.Descriptor instead.
func (*LabelDecider) Descriptor() ([]byte, []int) {
	return file_organisation_proto_rawDescGZIP(), []int{3}
}

func (x *LabelDecider) GetIndex() int32 {
	if x != nil {
		return x.Index
	}
	return 0
}

func (x *LabelDecider) GetLabelPrefix() string {
	if x != nil {
		return x.LabelPrefix
	}
	return ""
}

type ArtistTranslation struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ArtistPrefix  string `protobuf:"bytes,1,opt,name=artist_prefix,json=artistPrefix,proto3" json:"artist_prefix,omitempty"`
	OrderedArtist string `protobuf:"bytes,2,opt,name=ordered_artist,json=orderedArtist,proto3" json:"ordered_artist,omitempty"`
}

func (x *ArtistTranslation) Reset() {
	*x = ArtistTranslation{}
	if protoimpl.UnsafeEnabled {
		mi := &file_organisation_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ArtistTranslation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ArtistTranslation) ProtoMessage() {}

func (x *ArtistTranslation) ProtoReflect() protoreflect.Message {
	mi := &file_organisation_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ArtistTranslation.ProtoReflect.Descriptor instead.
func (*ArtistTranslation) Descriptor() ([]byte, []int) {
	return file_organisation_proto_rawDescGZIP(), []int{4}
}

func (x *ArtistTranslation) GetArtistPrefix() string {
	if x != nil {
		return x.ArtistPrefix
	}
	return ""
}

func (x *ArtistTranslation) GetOrderedArtist() string {
	if x != nil {
		return x.OrderedArtist
	}
	return ""
}

var File_organisation_proto protoreflect.FileDescriptor

var file_organisation_proto_rawDesc = []byte{
	0x0a, 0x12, 0x6f, 0x72, 0x67, 0x61, 0x6e, 0x69, 0x73, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65,
	0x22, 0xb3, 0x02, 0x0a, 0x0c, 0x4f, 0x72, 0x67, 0x61, 0x6e, 0x69, 0x73, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x12, 0x35, 0x0a, 0x0a, 0x66, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x73, 0x65, 0x74, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69,
	0x6c, 0x65, 0x2e, 0x46, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x53, 0x65, 0x74, 0x52, 0x0a, 0x66, 0x6f,
	0x6c, 0x64, 0x65, 0x72, 0x73, 0x65, 0x74, 0x73, 0x12, 0x29, 0x0a, 0x06, 0x73, 0x70, 0x61, 0x63,
	0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f,
	0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x53, 0x70, 0x61, 0x63, 0x65, 0x52, 0x06, 0x73, 0x70, 0x61,
	0x63, 0x65, 0x73, 0x12, 0x3f, 0x0a, 0x0e, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x5f, 0x64, 0x65, 0x63,
	0x69, 0x64, 0x65, 0x72, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x67, 0x72,
	0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x44, 0x65,
	0x63, 0x69, 0x64, 0x65, 0x72, 0x52, 0x0d, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x44, 0x65, 0x63, 0x69,
	0x64, 0x65, 0x72, 0x73, 0x12, 0x4c, 0x0a, 0x12, 0x61, 0x72, 0x74, 0x69, 0x73, 0x74, 0x5f, 0x74,
	0x72, 0x61, 0x6e, 0x73, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x1d, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x41, 0x72,
	0x74, 0x69, 0x73, 0x74, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52,
	0x11, 0x61, 0x72, 0x74, 0x69, 0x73, 0x74, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x6c, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x12, 0x32, 0x0a, 0x15, 0x61, 0x75, 0x74, 0x6f, 0x5f, 0x61, 0x72, 0x74, 0x69, 0x73,
	0x74, 0x5f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x6c, 0x61, 0x74, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x08, 0x52, 0x13, 0x61, 0x75, 0x74, 0x6f, 0x41, 0x72, 0x74, 0x69, 0x73, 0x74, 0x54, 0x72, 0x61,
	0x6e, 0x73, 0x6c, 0x61, 0x74, 0x65, 0x22, 0xb0, 0x01, 0x0a, 0x05, 0x53, 0x70, 0x61, 0x63, 0x65,
	0x12, 0x14, 0x0a, 0x05, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x05, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x12, 0x14, 0x0a, 0x05, 0x75, 0x6e, 0x69, 0x74, 0x73, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x75, 0x6e, 0x69, 0x74, 0x73, 0x12, 0x23, 0x0a, 0x0d,
	0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x73, 0x5f, 0x77, 0x69, 0x64, 0x74, 0x68, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x0c, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x73, 0x57, 0x69, 0x64, 0x74,
	0x68, 0x12, 0x1f, 0x0a, 0x0b, 0x64, 0x69, 0x73, 0x6b, 0x73, 0x5f, 0x77, 0x69, 0x64, 0x74, 0x68,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0a, 0x64, 0x69, 0x73, 0x6b, 0x73, 0x57, 0x69, 0x64,
	0x74, 0x68, 0x12, 0x14, 0x0a, 0x05, 0x77, 0x69, 0x64, 0x74, 0x68, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x02, 0x52, 0x05, 0x77, 0x69, 0x64, 0x74, 0x68, 0x12, 0x1f, 0x0a, 0x0b, 0x66, 0x6f, 0x6c, 0x64,
	0x65, 0x72, 0x5f, 0x73, 0x65, 0x74, 0x73, 0x18, 0x06, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0a, 0x66,
	0x6f, 0x6c, 0x64, 0x65, 0x72, 0x53, 0x65, 0x74, 0x73, 0x22, 0x73, 0x0a, 0x09, 0x46, 0x6f, 0x6c,
	0x64, 0x65, 0x72, 0x53, 0x65, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x69, 0x6e,
	0x64, 0x65, 0x78, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x69, 0x6e, 0x64, 0x65, 0x78,
	0x12, 0x16, 0x0a, 0x06, 0x66, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x06, 0x66, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x12, 0x24, 0x0a, 0x04, 0x73, 0x6f, 0x72, 0x74,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x10, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68,
	0x69, 0x6c, 0x65, 0x2e, 0x53, 0x6f, 0x72, 0x74, 0x52, 0x04, 0x73, 0x6f, 0x72, 0x74, 0x22, 0x47,
	0x0a, 0x0c, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x44, 0x65, 0x63, 0x69, 0x64, 0x65, 0x72, 0x12, 0x14,
	0x0a, 0x05, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x69,
	0x6e, 0x64, 0x65, 0x78, 0x12, 0x21, 0x0a, 0x0c, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x5f, 0x70, 0x72,
	0x65, 0x66, 0x69, 0x78, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x6c, 0x61, 0x62, 0x65,
	0x6c, 0x50, 0x72, 0x65, 0x66, 0x69, 0x78, 0x22, 0x5f, 0x0a, 0x11, 0x41, 0x72, 0x74, 0x69, 0x73,
	0x74, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x23, 0x0a, 0x0d,
	0x61, 0x72, 0x74, 0x69, 0x73, 0x74, 0x5f, 0x70, 0x72, 0x65, 0x66, 0x69, 0x78, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0c, 0x61, 0x72, 0x74, 0x69, 0x73, 0x74, 0x50, 0x72, 0x65, 0x66, 0x69,
	0x78, 0x12, 0x25, 0x0a, 0x0e, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x65, 0x64, 0x5f, 0x61, 0x72, 0x74,
	0x69, 0x73, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x6f, 0x72, 0x64, 0x65, 0x72,
	0x65, 0x64, 0x41, 0x72, 0x74, 0x69, 0x73, 0x74, 0x2a, 0x17, 0x0a, 0x04, 0x53, 0x6f, 0x72, 0x74,
	0x12, 0x0f, 0x0a, 0x0b, 0x41, 0x52, 0x54, 0x49, 0x53, 0x54, 0x5f, 0x59, 0x45, 0x41, 0x52, 0x10,
	0x00, 0x42, 0x2a, 0x5a, 0x28, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x62, 0x72, 0x6f, 0x74, 0x68, 0x65, 0x72, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x2f, 0x67, 0x72, 0x61,
	0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_organisation_proto_rawDescOnce sync.Once
	file_organisation_proto_rawDescData = file_organisation_proto_rawDesc
)

func file_organisation_proto_rawDescGZIP() []byte {
	file_organisation_proto_rawDescOnce.Do(func() {
		file_organisation_proto_rawDescData = protoimpl.X.CompressGZIP(file_organisation_proto_rawDescData)
	})
	return file_organisation_proto_rawDescData
}

var file_organisation_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_organisation_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_organisation_proto_goTypes = []interface{}{
	(Sort)(0),                 // 0: gramophile.Sort
	(*Organisation)(nil),      // 1: gramophile.Organisation
	(*Space)(nil),             // 2: gramophile.Space
	(*FolderSet)(nil),         // 3: gramophile.FolderSet
	(*LabelDecider)(nil),      // 4: gramophile.LabelDecider
	(*ArtistTranslation)(nil), // 5: gramophile.ArtistTranslation
}
var file_organisation_proto_depIdxs = []int32{
	3, // 0: gramophile.Organisation.foldersets:type_name -> gramophile.FolderSet
	2, // 1: gramophile.Organisation.spaces:type_name -> gramophile.Space
	4, // 2: gramophile.Organisation.label_deciders:type_name -> gramophile.LabelDecider
	5, // 3: gramophile.Organisation.artist_translation:type_name -> gramophile.ArtistTranslation
	0, // 4: gramophile.FolderSet.sort:type_name -> gramophile.Sort
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_organisation_proto_init() }
func file_organisation_proto_init() {
	if File_organisation_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_organisation_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Organisation); i {
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
		file_organisation_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Space); i {
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
		file_organisation_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FolderSet); i {
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
		file_organisation_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LabelDecider); i {
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
		file_organisation_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ArtistTranslation); i {
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
			RawDescriptor: file_organisation_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_organisation_proto_goTypes,
		DependencyIndexes: file_organisation_proto_depIdxs,
		EnumInfos:         file_organisation_proto_enumTypes,
		MessageInfos:      file_organisation_proto_msgTypes,
	}.Build()
	File_organisation_proto = out.File
	file_organisation_proto_rawDesc = nil
	file_organisation_proto_goTypes = nil
	file_organisation_proto_depIdxs = nil
}
