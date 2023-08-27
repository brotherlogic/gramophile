// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.12.4
// source: config.proto

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

type Basis int32

const (
	Basis_DISCOGS    Basis = 0
	Basis_GRAMOPHILE Basis = 1
)

// Enum value maps for Basis.
var (
	Basis_name = map[int32]string{
		0: "DISCOGS",
		1: "GRAMOPHILE",
	}
	Basis_value = map[string]int32{
		"DISCOGS":    0,
		"GRAMOPHILE": 1,
	}
)

func (x Basis) Enum() *Basis {
	p := new(Basis)
	*p = x
	return p
}

func (x Basis) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Basis) Descriptor() protoreflect.EnumDescriptor {
	return file_config_proto_enumTypes[0].Descriptor()
}

func (Basis) Type() protoreflect.EnumType {
	return &file_config_proto_enumTypes[0]
}

func (x Basis) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Basis.Descriptor instead.
func (Basis) EnumDescriptor() ([]byte, []int) {
	return file_config_proto_rawDescGZIP(), []int{0}
}

type Mandate int32

const (
	Mandate_NONE        Mandate = 0
	Mandate_RECOMMENDED Mandate = 1
	Mandate_REQUIRED    Mandate = 2
)

// Enum value maps for Mandate.
var (
	Mandate_name = map[int32]string{
		0: "NONE",
		1: "RECOMMENDED",
		2: "REQUIRED",
	}
	Mandate_value = map[string]int32{
		"NONE":        0,
		"RECOMMENDED": 1,
		"REQUIRED":    2,
	}
)

func (x Mandate) Enum() *Mandate {
	p := new(Mandate)
	*p = x
	return p
}

func (x Mandate) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Mandate) Descriptor() protoreflect.EnumDescriptor {
	return file_config_proto_enumTypes[1].Descriptor()
}

func (Mandate) Type() protoreflect.EnumType {
	return &file_config_proto_enumTypes[1]
}

func (x Mandate) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Mandate.Descriptor instead.
func (Mandate) EnumDescriptor() ([]byte, []int) {
	return file_config_proto_rawDescGZIP(), []int{1}
}

type Order_Ordering int32

const (
	Order_ORDER_RANDOM     Order_Ordering = 0
	Order_ORDER_ADDED_DATE Order_Ordering = 1
)

// Enum value maps for Order_Ordering.
var (
	Order_Ordering_name = map[int32]string{
		0: "ORDER_RANDOM",
		1: "ORDER_ADDED_DATE",
	}
	Order_Ordering_value = map[string]int32{
		"ORDER_RANDOM":     0,
		"ORDER_ADDED_DATE": 1,
	}
)

func (x Order_Ordering) Enum() *Order_Ordering {
	p := new(Order_Ordering)
	*p = x
	return p
}

func (x Order_Ordering) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Order_Ordering) Descriptor() protoreflect.EnumDescriptor {
	return file_config_proto_enumTypes[2].Descriptor()
}

func (Order_Ordering) Type() protoreflect.EnumType {
	return &file_config_proto_enumTypes[2]
}

func (x Order_Ordering) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Order_Ordering.Descriptor instead.
func (Order_Ordering) EnumDescriptor() ([]byte, []int) {
	return file_config_proto_rawDescGZIP(), []int{3, 0}
}

type Filter struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Formats       []string `protobuf:"bytes,1,rep,name=formats,proto3" json:"formats,omitempty"`
	ExcludeFolder []int32  `protobuf:"varint,2,rep,packed,name=exclude_folder,json=excludeFolder,proto3" json:"exclude_folder,omitempty"`
	IncludeFolder []int32  `protobuf:"varint,3,rep,packed,name=include_folder,json=includeFolder,proto3" json:"include_folder,omitempty"`
}

func (x *Filter) Reset() {
	*x = Filter{}
	if protoimpl.UnsafeEnabled {
		mi := &file_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Filter) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Filter) ProtoMessage() {}

func (x *Filter) ProtoReflect() protoreflect.Message {
	mi := &file_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Filter.ProtoReflect.Descriptor instead.
func (*Filter) Descriptor() ([]byte, []int) {
	return file_config_proto_rawDescGZIP(), []int{0}
}

func (x *Filter) GetFormats() []string {
	if x != nil {
		return x.Formats
	}
	return nil
}

func (x *Filter) GetExcludeFolder() []int32 {
	if x != nil {
		return x.ExcludeFolder
	}
	return nil
}

func (x *Filter) GetIncludeFolder() []int32 {
	if x != nil {
		return x.IncludeFolder
	}
	return nil
}

type CleaningConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Cleaning             Mandate `protobuf:"varint,1,opt,name=cleaning,proto3,enum=gramophile.Mandate" json:"cleaning,omitempty"`
	AppliesTo            *Filter `protobuf:"bytes,2,opt,name=applies_to,json=appliesTo,proto3" json:"applies_to,omitempty"`
	CleaningGapInSeconds int64   `protobuf:"varint,3,opt,name=cleaning_gap_in_seconds,json=cleaningGapInSeconds,proto3" json:"cleaning_gap_in_seconds,omitempty"`
	CleaningGapInPlays   int32   `protobuf:"varint,4,opt,name=cleaning_gap_in_plays,json=cleaningGapInPlays,proto3" json:"cleaning_gap_in_plays,omitempty"`
}

func (x *CleaningConfig) Reset() {
	*x = CleaningConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_config_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CleaningConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CleaningConfig) ProtoMessage() {}

func (x *CleaningConfig) ProtoReflect() protoreflect.Message {
	mi := &file_config_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CleaningConfig.ProtoReflect.Descriptor instead.
func (*CleaningConfig) Descriptor() ([]byte, []int) {
	return file_config_proto_rawDescGZIP(), []int{1}
}

func (x *CleaningConfig) GetCleaning() Mandate {
	if x != nil {
		return x.Cleaning
	}
	return Mandate_NONE
}

func (x *CleaningConfig) GetAppliesTo() *Filter {
	if x != nil {
		return x.AppliesTo
	}
	return nil
}

func (x *CleaningConfig) GetCleaningGapInSeconds() int64 {
	if x != nil {
		return x.CleaningGapInSeconds
	}
	return 0
}

func (x *CleaningConfig) GetCleaningGapInPlays() int32 {
	if x != nil {
		return x.CleaningGapInPlays
	}
	return 0
}

type ListenConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Enabled bool            `protobuf:"varint,1,opt,name=enabled,proto3" json:"enabled,omitempty"`
	Filters []*ListenFilter `protobuf:"bytes,2,rep,name=filters,proto3" json:"filters,omitempty"`
}

func (x *ListenConfig) Reset() {
	*x = ListenConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_config_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListenConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListenConfig) ProtoMessage() {}

func (x *ListenConfig) ProtoReflect() protoreflect.Message {
	mi := &file_config_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListenConfig.ProtoReflect.Descriptor instead.
func (*ListenConfig) Descriptor() ([]byte, []int) {
	return file_config_proto_rawDescGZIP(), []int{2}
}

func (x *ListenConfig) GetEnabled() bool {
	if x != nil {
		return x.Enabled
	}
	return false
}

func (x *ListenConfig) GetFilters() []*ListenFilter {
	if x != nil {
		return x.Filters
	}
	return nil
}

type Order struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ordering Order_Ordering `protobuf:"varint,1,opt,name=ordering,proto3,enum=gramophile.Order_Ordering" json:"ordering,omitempty"`
	Reverse  bool           `protobuf:"varint,2,opt,name=reverse,proto3" json:"reverse,omitempty"`
}

func (x *Order) Reset() {
	*x = Order{}
	if protoimpl.UnsafeEnabled {
		mi := &file_config_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Order) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order) ProtoMessage() {}

func (x *Order) ProtoReflect() protoreflect.Message {
	mi := &file_config_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order.ProtoReflect.Descriptor instead.
func (*Order) Descriptor() ([]byte, []int) {
	return file_config_proto_rawDescGZIP(), []int{3}
}

func (x *Order) GetOrdering() Order_Ordering {
	if x != nil {
		return x.Ordering
	}
	return Order_ORDER_RANDOM
}

func (x *Order) GetReverse() bool {
	if x != nil {
		return x.Reverse
	}
	return false
}

type ListenFilter struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name   string  `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Filter *Filter `protobuf:"bytes,2,opt,name=filter,proto3" json:"filter,omitempty"`
	Order  *Order  `protobuf:"bytes,3,opt,name=order,proto3" json:"order,omitempty"`
}

func (x *ListenFilter) Reset() {
	*x = ListenFilter{}
	if protoimpl.UnsafeEnabled {
		mi := &file_config_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListenFilter) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListenFilter) ProtoMessage() {}

func (x *ListenFilter) ProtoReflect() protoreflect.Message {
	mi := &file_config_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListenFilter.ProtoReflect.Descriptor instead.
func (*ListenFilter) Descriptor() ([]byte, []int) {
	return file_config_proto_rawDescGZIP(), []int{4}
}

func (x *ListenFilter) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ListenFilter) GetFilter() *Filter {
	if x != nil {
		return x.Filter
	}
	return nil
}

func (x *ListenFilter) GetOrder() *Order {
	if x != nil {
		return x.Order
	}
	return nil
}

type GramophileConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Basis              Basis               `protobuf:"varint,2,opt,name=basis,proto3,enum=gramophile.Basis" json:"basis,omitempty"`
	CleaningConfig     *CleaningConfig     `protobuf:"bytes,1,opt,name=cleaning_config,json=cleaningConfig,proto3" json:"cleaning_config,omitempty"`
	ListenConfig       *ListenConfig       `protobuf:"bytes,3,opt,name=listen_config,json=listenConfig,proto3" json:"listen_config,omitempty"`
	WidthConfig        *WidthConfig        `protobuf:"bytes,4,opt,name=width_config,json=widthConfig,proto3" json:"width_config,omitempty"`
	OrganisationConfig *OrganisationConfig `protobuf:"bytes,5,opt,name=organisation_config,json=organisationConfig,proto3" json:"organisation_config,omitempty"`
	WeightConfig       *WeightConfig       `protobuf:"bytes,6,opt,name=weight_config,json=weightConfig,proto3" json:"weight_config,omitempty"`
	GoalFolderConfig   *GoalFolderConfig   `protobuf:"bytes,7,opt,name=goal_folder_config,json=goalFolderConfig,proto3" json:"goal_folder_config,omitempty"`
}

func (x *GramophileConfig) Reset() {
	*x = GramophileConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_config_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GramophileConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GramophileConfig) ProtoMessage() {}

func (x *GramophileConfig) ProtoReflect() protoreflect.Message {
	mi := &file_config_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GramophileConfig.ProtoReflect.Descriptor instead.
func (*GramophileConfig) Descriptor() ([]byte, []int) {
	return file_config_proto_rawDescGZIP(), []int{5}
}

func (x *GramophileConfig) GetBasis() Basis {
	if x != nil {
		return x.Basis
	}
	return Basis_DISCOGS
}

func (x *GramophileConfig) GetCleaningConfig() *CleaningConfig {
	if x != nil {
		return x.CleaningConfig
	}
	return nil
}

func (x *GramophileConfig) GetListenConfig() *ListenConfig {
	if x != nil {
		return x.ListenConfig
	}
	return nil
}

func (x *GramophileConfig) GetWidthConfig() *WidthConfig {
	if x != nil {
		return x.WidthConfig
	}
	return nil
}

func (x *GramophileConfig) GetOrganisationConfig() *OrganisationConfig {
	if x != nil {
		return x.OrganisationConfig
	}
	return nil
}

func (x *GramophileConfig) GetWeightConfig() *WeightConfig {
	if x != nil {
		return x.WeightConfig
	}
	return nil
}

func (x *GramophileConfig) GetGoalFolderConfig() *GoalFolderConfig {
	if x != nil {
		return x.GoalFolderConfig
	}
	return nil
}

type WeightConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Mandate Mandate `protobuf:"varint,1,opt,name=mandate,proto3,enum=gramophile.Mandate" json:"mandate,omitempty"`
}

func (x *WeightConfig) Reset() {
	*x = WeightConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_config_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WeightConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WeightConfig) ProtoMessage() {}

func (x *WeightConfig) ProtoReflect() protoreflect.Message {
	mi := &file_config_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WeightConfig.ProtoReflect.Descriptor instead.
func (*WeightConfig) Descriptor() ([]byte, []int) {
	return file_config_proto_rawDescGZIP(), []int{6}
}

func (x *WeightConfig) GetMandate() Mandate {
	if x != nil {
		return x.Mandate
	}
	return Mandate_NONE
}

type WidthConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Mandate Mandate `protobuf:"varint,1,opt,name=mandate,proto3,enum=gramophile.Mandate" json:"mandate,omitempty"`
}

func (x *WidthConfig) Reset() {
	*x = WidthConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_config_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WidthConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WidthConfig) ProtoMessage() {}

func (x *WidthConfig) ProtoReflect() protoreflect.Message {
	mi := &file_config_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WidthConfig.ProtoReflect.Descriptor instead.
func (*WidthConfig) Descriptor() ([]byte, []int) {
	return file_config_proto_rawDescGZIP(), []int{7}
}

func (x *WidthConfig) GetMandate() Mandate {
	if x != nil {
		return x.Mandate
	}
	return Mandate_NONE
}

type GoalFolderConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Mandate Mandate `protobuf:"varint,1,opt,name=mandate,proto3,enum=gramophile.Mandate" json:"mandate,omitempty"`
}

func (x *GoalFolderConfig) Reset() {
	*x = GoalFolderConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_config_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GoalFolderConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GoalFolderConfig) ProtoMessage() {}

func (x *GoalFolderConfig) ProtoReflect() protoreflect.Message {
	mi := &file_config_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GoalFolderConfig.ProtoReflect.Descriptor instead.
func (*GoalFolderConfig) Descriptor() ([]byte, []int) {
	return file_config_proto_rawDescGZIP(), []int{8}
}

func (x *GoalFolderConfig) GetMandate() Mandate {
	if x != nil {
		return x.Mandate
	}
	return Mandate_NONE
}

var File_config_proto protoreflect.FileDescriptor

var file_config_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a,
	0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x1a, 0x12, 0x6f, 0x72, 0x67, 0x61,
	0x6e, 0x69, 0x73, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x70,
	0x0a, 0x06, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x12, 0x18, 0x0a, 0x07, 0x66, 0x6f, 0x72, 0x6d,
	0x61, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x66, 0x6f, 0x72, 0x6d, 0x61,
	0x74, 0x73, 0x12, 0x25, 0x0a, 0x0e, 0x65, 0x78, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x5f, 0x66, 0x6f,
	0x6c, 0x64, 0x65, 0x72, 0x18, 0x02, 0x20, 0x03, 0x28, 0x05, 0x52, 0x0d, 0x65, 0x78, 0x63, 0x6c,
	0x75, 0x64, 0x65, 0x46, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x12, 0x25, 0x0a, 0x0e, 0x69, 0x6e, 0x63,
	0x6c, 0x75, 0x64, 0x65, 0x5f, 0x66, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x18, 0x03, 0x20, 0x03, 0x28,
	0x05, 0x52, 0x0d, 0x69, 0x6e, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x46, 0x6f, 0x6c, 0x64, 0x65, 0x72,
	0x22, 0xde, 0x01, 0x0a, 0x0e, 0x43, 0x6c, 0x65, 0x61, 0x6e, 0x69, 0x6e, 0x67, 0x43, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x12, 0x2f, 0x0a, 0x08, 0x63, 0x6c, 0x65, 0x61, 0x6e, 0x69, 0x6e, 0x67, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x13, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69,
	0x6c, 0x65, 0x2e, 0x4d, 0x61, 0x6e, 0x64, 0x61, 0x74, 0x65, 0x52, 0x08, 0x63, 0x6c, 0x65, 0x61,
	0x6e, 0x69, 0x6e, 0x67, 0x12, 0x31, 0x0a, 0x0a, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x65, 0x73, 0x5f,
	0x74, 0x6f, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f,
	0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x52, 0x09, 0x61, 0x70,
	0x70, 0x6c, 0x69, 0x65, 0x73, 0x54, 0x6f, 0x12, 0x35, 0x0a, 0x17, 0x63, 0x6c, 0x65, 0x61, 0x6e,
	0x69, 0x6e, 0x67, 0x5f, 0x67, 0x61, 0x70, 0x5f, 0x69, 0x6e, 0x5f, 0x73, 0x65, 0x63, 0x6f, 0x6e,
	0x64, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x14, 0x63, 0x6c, 0x65, 0x61, 0x6e, 0x69,
	0x6e, 0x67, 0x47, 0x61, 0x70, 0x49, 0x6e, 0x53, 0x65, 0x63, 0x6f, 0x6e, 0x64, 0x73, 0x12, 0x31,
	0x0a, 0x15, 0x63, 0x6c, 0x65, 0x61, 0x6e, 0x69, 0x6e, 0x67, 0x5f, 0x67, 0x61, 0x70, 0x5f, 0x69,
	0x6e, 0x5f, 0x70, 0x6c, 0x61, 0x79, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x12, 0x63,
	0x6c, 0x65, 0x61, 0x6e, 0x69, 0x6e, 0x67, 0x47, 0x61, 0x70, 0x49, 0x6e, 0x50, 0x6c, 0x61, 0x79,
	0x73, 0x22, 0x5c, 0x0a, 0x0c, 0x4c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x12, 0x18, 0x0a, 0x07, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x07, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x12, 0x32, 0x0a, 0x07, 0x66,
	0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x67,
	0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x65, 0x6e,
	0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x52, 0x07, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x22,
	0x8d, 0x01, 0x0a, 0x05, 0x4f, 0x72, 0x64, 0x65, 0x72, 0x12, 0x36, 0x0a, 0x08, 0x6f, 0x72, 0x64,
	0x65, 0x72, 0x69, 0x6e, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1a, 0x2e, 0x67, 0x72,
	0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x4f, 0x72, 0x64, 0x65, 0x72, 0x2e, 0x4f,
	0x72, 0x64, 0x65, 0x72, 0x69, 0x6e, 0x67, 0x52, 0x08, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x69, 0x6e,
	0x67, 0x12, 0x18, 0x0a, 0x07, 0x72, 0x65, 0x76, 0x65, 0x72, 0x73, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x07, 0x72, 0x65, 0x76, 0x65, 0x72, 0x73, 0x65, 0x22, 0x32, 0x0a, 0x08, 0x4f,
	0x72, 0x64, 0x65, 0x72, 0x69, 0x6e, 0x67, 0x12, 0x10, 0x0a, 0x0c, 0x4f, 0x52, 0x44, 0x45, 0x52,
	0x5f, 0x52, 0x41, 0x4e, 0x44, 0x4f, 0x4d, 0x10, 0x00, 0x12, 0x14, 0x0a, 0x10, 0x4f, 0x52, 0x44,
	0x45, 0x52, 0x5f, 0x41, 0x44, 0x44, 0x45, 0x44, 0x5f, 0x44, 0x41, 0x54, 0x45, 0x10, 0x01, 0x22,
	0x77, 0x0a, 0x0c, 0x4c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x12,
	0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x12, 0x2a, 0x0a, 0x06, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65,
	0x2e, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x52, 0x06, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x12,
	0x27, 0x0a, 0x05, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x11,
	0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x4f, 0x72, 0x64, 0x65,
	0x72, 0x52, 0x05, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x22, 0xd7, 0x03, 0x0a, 0x10, 0x47, 0x72, 0x61,
	0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x27, 0x0a,
	0x05, 0x62, 0x61, 0x73, 0x69, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x11, 0x2e, 0x67,
	0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x42, 0x61, 0x73, 0x69, 0x73, 0x52,
	0x05, 0x62, 0x61, 0x73, 0x69, 0x73, 0x12, 0x43, 0x0a, 0x0f, 0x63, 0x6c, 0x65, 0x61, 0x6e, 0x69,
	0x6e, 0x67, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1a, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x43, 0x6c, 0x65,
	0x61, 0x6e, 0x69, 0x6e, 0x67, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x0e, 0x63, 0x6c, 0x65,
	0x61, 0x6e, 0x69, 0x6e, 0x67, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x3d, 0x0a, 0x0d, 0x6c,
	0x69, 0x73, 0x74, 0x65, 0x6e, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x18, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e,
	0x4c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x0c, 0x6c, 0x69,
	0x73, 0x74, 0x65, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x3a, 0x0a, 0x0c, 0x77, 0x69,
	0x64, 0x74, 0x68, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x17, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x57, 0x69,
	0x64, 0x74, 0x68, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x0b, 0x77, 0x69, 0x64, 0x74, 0x68,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x4f, 0x0a, 0x13, 0x6f, 0x72, 0x67, 0x61, 0x6e, 0x69,
	0x73, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65,
	0x2e, 0x4f, 0x72, 0x67, 0x61, 0x6e, 0x69, 0x73, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x52, 0x12, 0x6f, 0x72, 0x67, 0x61, 0x6e, 0x69, 0x73, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x3d, 0x0a, 0x0d, 0x77, 0x65, 0x69, 0x67, 0x68,
	0x74, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18,
	0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x57, 0x65, 0x69, 0x67,
	0x68, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x0c, 0x77, 0x65, 0x69, 0x67, 0x68, 0x74,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x4a, 0x0a, 0x12, 0x67, 0x6f, 0x61, 0x6c, 0x5f, 0x66,
	0x6f, 0x6c, 0x64, 0x65, 0x72, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x07, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e,
	0x47, 0x6f, 0x61, 0x6c, 0x46, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x52, 0x10, 0x67, 0x6f, 0x61, 0x6c, 0x46, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x22, 0x3d, 0x0a, 0x0c, 0x57, 0x65, 0x69, 0x67, 0x68, 0x74, 0x43, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x12, 0x2d, 0x0a, 0x07, 0x6d, 0x61, 0x6e, 0x64, 0x61, 0x74, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0e, 0x32, 0x13, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65,
	0x2e, 0x4d, 0x61, 0x6e, 0x64, 0x61, 0x74, 0x65, 0x52, 0x07, 0x6d, 0x61, 0x6e, 0x64, 0x61, 0x74,
	0x65, 0x22, 0x3c, 0x0a, 0x0b, 0x57, 0x69, 0x64, 0x74, 0x68, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x12, 0x2d, 0x0a, 0x07, 0x6d, 0x61, 0x6e, 0x64, 0x61, 0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x13, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x4d,
	0x61, 0x6e, 0x64, 0x61, 0x74, 0x65, 0x52, 0x07, 0x6d, 0x61, 0x6e, 0x64, 0x61, 0x74, 0x65, 0x22,
	0x41, 0x0a, 0x10, 0x47, 0x6f, 0x61, 0x6c, 0x46, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x43, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x12, 0x2d, 0x0a, 0x07, 0x6d, 0x61, 0x6e, 0x64, 0x61, 0x74, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0e, 0x32, 0x13, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c,
	0x65, 0x2e, 0x4d, 0x61, 0x6e, 0x64, 0x61, 0x74, 0x65, 0x52, 0x07, 0x6d, 0x61, 0x6e, 0x64, 0x61,
	0x74, 0x65, 0x2a, 0x24, 0x0a, 0x05, 0x42, 0x61, 0x73, 0x69, 0x73, 0x12, 0x0b, 0x0a, 0x07, 0x44,
	0x49, 0x53, 0x43, 0x4f, 0x47, 0x53, 0x10, 0x00, 0x12, 0x0e, 0x0a, 0x0a, 0x47, 0x52, 0x41, 0x4d,
	0x4f, 0x50, 0x48, 0x49, 0x4c, 0x45, 0x10, 0x01, 0x2a, 0x32, 0x0a, 0x07, 0x4d, 0x61, 0x6e, 0x64,
	0x61, 0x74, 0x65, 0x12, 0x08, 0x0a, 0x04, 0x4e, 0x4f, 0x4e, 0x45, 0x10, 0x00, 0x12, 0x0f, 0x0a,
	0x0b, 0x52, 0x45, 0x43, 0x4f, 0x4d, 0x4d, 0x45, 0x4e, 0x44, 0x45, 0x44, 0x10, 0x01, 0x12, 0x0c,
	0x0a, 0x08, 0x52, 0x45, 0x51, 0x55, 0x49, 0x52, 0x45, 0x44, 0x10, 0x02, 0x42, 0x2a, 0x5a, 0x28,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x72, 0x6f, 0x74, 0x68,
	0x65, 0x72, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x2f, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69,
	0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_config_proto_rawDescOnce sync.Once
	file_config_proto_rawDescData = file_config_proto_rawDesc
)

func file_config_proto_rawDescGZIP() []byte {
	file_config_proto_rawDescOnce.Do(func() {
		file_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_config_proto_rawDescData)
	})
	return file_config_proto_rawDescData
}

var file_config_proto_enumTypes = make([]protoimpl.EnumInfo, 3)
var file_config_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_config_proto_goTypes = []interface{}{
	(Basis)(0),                 // 0: gramophile.Basis
	(Mandate)(0),               // 1: gramophile.Mandate
	(Order_Ordering)(0),        // 2: gramophile.Order.Ordering
	(*Filter)(nil),             // 3: gramophile.Filter
	(*CleaningConfig)(nil),     // 4: gramophile.CleaningConfig
	(*ListenConfig)(nil),       // 5: gramophile.ListenConfig
	(*Order)(nil),              // 6: gramophile.Order
	(*ListenFilter)(nil),       // 7: gramophile.ListenFilter
	(*GramophileConfig)(nil),   // 8: gramophile.GramophileConfig
	(*WeightConfig)(nil),       // 9: gramophile.WeightConfig
	(*WidthConfig)(nil),        // 10: gramophile.WidthConfig
	(*GoalFolderConfig)(nil),   // 11: gramophile.GoalFolderConfig
	(*OrganisationConfig)(nil), // 12: gramophile.OrganisationConfig
}
var file_config_proto_depIdxs = []int32{
	1,  // 0: gramophile.CleaningConfig.cleaning:type_name -> gramophile.Mandate
	3,  // 1: gramophile.CleaningConfig.applies_to:type_name -> gramophile.Filter
	7,  // 2: gramophile.ListenConfig.filters:type_name -> gramophile.ListenFilter
	2,  // 3: gramophile.Order.ordering:type_name -> gramophile.Order.Ordering
	3,  // 4: gramophile.ListenFilter.filter:type_name -> gramophile.Filter
	6,  // 5: gramophile.ListenFilter.order:type_name -> gramophile.Order
	0,  // 6: gramophile.GramophileConfig.basis:type_name -> gramophile.Basis
	4,  // 7: gramophile.GramophileConfig.cleaning_config:type_name -> gramophile.CleaningConfig
	5,  // 8: gramophile.GramophileConfig.listen_config:type_name -> gramophile.ListenConfig
	10, // 9: gramophile.GramophileConfig.width_config:type_name -> gramophile.WidthConfig
	12, // 10: gramophile.GramophileConfig.organisation_config:type_name -> gramophile.OrganisationConfig
	9,  // 11: gramophile.GramophileConfig.weight_config:type_name -> gramophile.WeightConfig
	11, // 12: gramophile.GramophileConfig.goal_folder_config:type_name -> gramophile.GoalFolderConfig
	1,  // 13: gramophile.WeightConfig.mandate:type_name -> gramophile.Mandate
	1,  // 14: gramophile.WidthConfig.mandate:type_name -> gramophile.Mandate
	1,  // 15: gramophile.GoalFolderConfig.mandate:type_name -> gramophile.Mandate
	16, // [16:16] is the sub-list for method output_type
	16, // [16:16] is the sub-list for method input_type
	16, // [16:16] is the sub-list for extension type_name
	16, // [16:16] is the sub-list for extension extendee
	0,  // [0:16] is the sub-list for field type_name
}

func init() { file_config_proto_init() }
func file_config_proto_init() {
	if File_config_proto != nil {
		return
	}
	file_organisation_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Filter); i {
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
		file_config_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CleaningConfig); i {
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
		file_config_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListenConfig); i {
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
		file_config_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Order); i {
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
		file_config_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListenFilter); i {
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
		file_config_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GramophileConfig); i {
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
		file_config_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WeightConfig); i {
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
		file_config_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WidthConfig); i {
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
		file_config_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GoalFolderConfig); i {
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
			RawDescriptor: file_config_proto_rawDesc,
			NumEnums:      3,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_config_proto_goTypes,
		DependencyIndexes: file_config_proto_depIdxs,
		EnumInfos:         file_config_proto_enumTypes,
		MessageInfos:      file_config_proto_msgTypes,
	}.Build()
	File_config_proto = out.File
	file_config_proto_rawDesc = nil
	file_config_proto_goTypes = nil
	file_config_proto_depIdxs = nil
}
