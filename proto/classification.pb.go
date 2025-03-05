// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v3.21.12
// source: classification.proto

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

type Comparator int32

const (
	Comparator_COMPARATOR_UNKNOWN                Comparator = 0
	Comparator_COMPARATOR_GREATER_THAN           Comparator = 1
	Comparator_COMPARATOR_GREATER_THAN_OR_EQUALS Comparator = 2
	Comparator_COMPARATOR_LESS_THAN              Comparator = 3
	Comparator_COMPARATOR_LESS_THAN_OR_EQUALS    Comparator = 4
)

// Enum value maps for Comparator.
var (
	Comparator_name = map[int32]string{
		0: "COMPARATOR_UNKNOWN",
		1: "COMPARATOR_GREATER_THAN",
		2: "COMPARATOR_GREATER_THAN_OR_EQUALS",
		3: "COMPARATOR_LESS_THAN",
		4: "COMPARATOR_LESS_THAN_OR_EQUALS",
	}
	Comparator_value = map[string]int32{
		"COMPARATOR_UNKNOWN":                0,
		"COMPARATOR_GREATER_THAN":           1,
		"COMPARATOR_GREATER_THAN_OR_EQUALS": 2,
		"COMPARATOR_LESS_THAN":              3,
		"COMPARATOR_LESS_THAN_OR_EQUALS":    4,
	}
)

func (x Comparator) Enum() *Comparator {
	p := new(Comparator)
	*p = x
	return p
}

func (x Comparator) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Comparator) Descriptor() protoreflect.EnumDescriptor {
	return file_classification_proto_enumTypes[0].Descriptor()
}

func (Comparator) Type() protoreflect.EnumType {
	return &file_classification_proto_enumTypes[0]
}

func (x Comparator) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Comparator.Descriptor instead.
func (Comparator) EnumDescriptor() ([]byte, []int) {
	return file_classification_proto_rawDescGZIP(), []int{0}
}

type BooleanSelector struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *BooleanSelector) Reset() {
	*x = BooleanSelector{}
	if protoimpl.UnsafeEnabled {
		mi := &file_classification_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BooleanSelector) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BooleanSelector) ProtoMessage() {}

func (x *BooleanSelector) ProtoReflect() protoreflect.Message {
	mi := &file_classification_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BooleanSelector.ProtoReflect.Descriptor instead.
func (*BooleanSelector) Descriptor() ([]byte, []int) {
	return file_classification_proto_rawDescGZIP(), []int{0}
}

func (x *BooleanSelector) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type IntSelector struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name      string     `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Threshold int64      `protobuf:"varint,2,opt,name=threshold,proto3" json:"threshold,omitempty"`
	Comp      Comparator `protobuf:"varint,3,opt,name=comp,proto3,enum=gramophile.Comparator" json:"comp,omitempty"`
}

func (x *IntSelector) Reset() {
	*x = IntSelector{}
	if protoimpl.UnsafeEnabled {
		mi := &file_classification_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IntSelector) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IntSelector) ProtoMessage() {}

func (x *IntSelector) ProtoReflect() protoreflect.Message {
	mi := &file_classification_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IntSelector.ProtoReflect.Descriptor instead.
func (*IntSelector) Descriptor() ([]byte, []int) {
	return file_classification_proto_rawDescGZIP(), []int{1}
}

func (x *IntSelector) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *IntSelector) GetThreshold() int64 {
	if x != nil {
		return x.Threshold
	}
	return 0
}

func (x *IntSelector) GetComp() Comparator {
	if x != nil {
		return x.Comp
	}
	return Comparator_COMPARATOR_UNKNOWN
}

type ClassificationRule struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RuleName string `protobuf:"bytes,1,opt,name=rule_name,json=ruleName,proto3" json:"rule_name,omitempty"`
	Priority int32  `protobuf:"varint,2,opt,name=priority,proto3" json:"priority,omitempty"`
	// Types that are assignable to Selector:
	//
	//	*ClassificationRule_BooleanSelector
	//	*ClassificationRule_IntSelector
	Selector isClassificationRule_Selector `protobuf_oneof:"selector"`
}

func (x *ClassificationRule) Reset() {
	*x = ClassificationRule{}
	if protoimpl.UnsafeEnabled {
		mi := &file_classification_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ClassificationRule) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClassificationRule) ProtoMessage() {}

func (x *ClassificationRule) ProtoReflect() protoreflect.Message {
	mi := &file_classification_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClassificationRule.ProtoReflect.Descriptor instead.
func (*ClassificationRule) Descriptor() ([]byte, []int) {
	return file_classification_proto_rawDescGZIP(), []int{2}
}

func (x *ClassificationRule) GetRuleName() string {
	if x != nil {
		return x.RuleName
	}
	return ""
}

func (x *ClassificationRule) GetPriority() int32 {
	if x != nil {
		return x.Priority
	}
	return 0
}

func (m *ClassificationRule) GetSelector() isClassificationRule_Selector {
	if m != nil {
		return m.Selector
	}
	return nil
}

func (x *ClassificationRule) GetBooleanSelector() *BooleanSelector {
	if x, ok := x.GetSelector().(*ClassificationRule_BooleanSelector); ok {
		return x.BooleanSelector
	}
	return nil
}

func (x *ClassificationRule) GetIntSelector() *IntSelector {
	if x, ok := x.GetSelector().(*ClassificationRule_IntSelector); ok {
		return x.IntSelector
	}
	return nil
}

type isClassificationRule_Selector interface {
	isClassificationRule_Selector()
}

type ClassificationRule_BooleanSelector struct {
	BooleanSelector *BooleanSelector `protobuf:"bytes,3,opt,name=boolean_selector,json=booleanSelector,proto3,oneof"`
}

type ClassificationRule_IntSelector struct {
	IntSelector *IntSelector `protobuf:"bytes,4,opt,name=int_selector,json=intSelector,proto3,oneof"`
}

func (*ClassificationRule_BooleanSelector) isClassificationRule_Selector() {}

func (*ClassificationRule_IntSelector) isClassificationRule_Selector() {}

type Classifier struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ClassifierName string                `protobuf:"bytes,1,opt,name=classifier_name,json=classifierName,proto3" json:"classifier_name,omitempty"`
	Rule           []*ClassificationRule `protobuf:"bytes,2,rep,name=rule,proto3" json:"rule,omitempty"`
	Classification string                `protobuf:"bytes,3,opt,name=classification,proto3" json:"classification,omitempty"`
}

func (x *Classifier) Reset() {
	*x = Classifier{}
	if protoimpl.UnsafeEnabled {
		mi := &file_classification_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Classifier) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Classifier) ProtoMessage() {}

func (x *Classifier) ProtoReflect() protoreflect.Message {
	mi := &file_classification_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Classifier.ProtoReflect.Descriptor instead.
func (*Classifier) Descriptor() ([]byte, []int) {
	return file_classification_proto_rawDescGZIP(), []int{3}
}

func (x *Classifier) GetClassifierName() string {
	if x != nil {
		return x.ClassifierName
	}
	return ""
}

func (x *Classifier) GetRule() []*ClassificationRule {
	if x != nil {
		return x.Rule
	}
	return nil
}

func (x *Classifier) GetClassification() string {
	if x != nil {
		return x.Classification
	}
	return ""
}

type ClassificationConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Classifiers []*Classifier `protobuf:"bytes,1,rep,name=classifiers,proto3" json:"classifiers,omitempty"`
}

func (x *ClassificationConfig) Reset() {
	*x = ClassificationConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_classification_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ClassificationConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClassificationConfig) ProtoMessage() {}

func (x *ClassificationConfig) ProtoReflect() protoreflect.Message {
	mi := &file_classification_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClassificationConfig.ProtoReflect.Descriptor instead.
func (*ClassificationConfig) Descriptor() ([]byte, []int) {
	return file_classification_proto_rawDescGZIP(), []int{4}
}

func (x *ClassificationConfig) GetClassifiers() []*Classifier {
	if x != nil {
		return x.Classifiers
	}
	return nil
}

var File_classification_proto protoreflect.FileDescriptor

var file_classification_proto_rawDesc = []byte{
	0x0a, 0x14, 0x63, 0x6c, 0x61, 0x73, 0x73, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69,
	0x6c, 0x65, 0x22, 0x25, 0x0a, 0x0f, 0x42, 0x6f, 0x6f, 0x6c, 0x65, 0x61, 0x6e, 0x53, 0x65, 0x6c,
	0x65, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x6b, 0x0a, 0x0b, 0x49, 0x6e, 0x74,
	0x53, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1c, 0x0a, 0x09,
	0x74, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x09, 0x74, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x12, 0x2a, 0x0a, 0x04, 0x63, 0x6f,
	0x6d, 0x70, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x16, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f,
	0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x61, 0x72, 0x61, 0x74, 0x6f, 0x72,
	0x52, 0x04, 0x63, 0x6f, 0x6d, 0x70, 0x22, 0xe1, 0x01, 0x0a, 0x12, 0x43, 0x6c, 0x61, 0x73, 0x73,
	0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x75, 0x6c, 0x65, 0x12, 0x1b, 0x0a,
	0x09, 0x72, 0x75, 0x6c, 0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x08, 0x72, 0x75, 0x6c, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x72,
	0x69, 0x6f, 0x72, 0x69, 0x74, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x70, 0x72,
	0x69, 0x6f, 0x72, 0x69, 0x74, 0x79, 0x12, 0x48, 0x0a, 0x10, 0x62, 0x6f, 0x6f, 0x6c, 0x65, 0x61,
	0x6e, 0x5f, 0x73, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1b, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x42, 0x6f,
	0x6f, 0x6c, 0x65, 0x61, 0x6e, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x48, 0x00, 0x52,
	0x0f, 0x62, 0x6f, 0x6f, 0x6c, 0x65, 0x61, 0x6e, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72,
	0x12, 0x3c, 0x0a, 0x0c, 0x69, 0x6e, 0x74, 0x5f, 0x73, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68,
	0x69, 0x6c, 0x65, 0x2e, 0x49, 0x6e, 0x74, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x48,
	0x00, 0x52, 0x0b, 0x69, 0x6e, 0x74, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x42, 0x0a,
	0x0a, 0x08, 0x73, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x22, 0x91, 0x01, 0x0a, 0x0a, 0x43,
	0x6c, 0x61, 0x73, 0x73, 0x69, 0x66, 0x69, 0x65, 0x72, 0x12, 0x27, 0x0a, 0x0f, 0x63, 0x6c, 0x61,
	0x73, 0x73, 0x69, 0x66, 0x69, 0x65, 0x72, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0e, 0x63, 0x6c, 0x61, 0x73, 0x73, 0x69, 0x66, 0x69, 0x65, 0x72, 0x4e, 0x61,
	0x6d, 0x65, 0x12, 0x32, 0x0a, 0x04, 0x72, 0x75, 0x6c, 0x65, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x1e, 0x2e, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x43, 0x6c,
	0x61, 0x73, 0x73, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x75, 0x6c, 0x65,
	0x52, 0x04, 0x72, 0x75, 0x6c, 0x65, 0x12, 0x26, 0x0a, 0x0e, 0x63, 0x6c, 0x61, 0x73, 0x73, 0x69,
	0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e,
	0x63, 0x6c, 0x61, 0x73, 0x73, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x50,
	0x0a, 0x14, 0x43, 0x6c, 0x61, 0x73, 0x73, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x38, 0x0a, 0x0b, 0x63, 0x6c, 0x61, 0x73, 0x73, 0x69,
	0x66, 0x69, 0x65, 0x72, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x67, 0x72,
	0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2e, 0x43, 0x6c, 0x61, 0x73, 0x73, 0x69, 0x66,
	0x69, 0x65, 0x72, 0x52, 0x0b, 0x63, 0x6c, 0x61, 0x73, 0x73, 0x69, 0x66, 0x69, 0x65, 0x72, 0x73,
	0x2a, 0xa6, 0x01, 0x0a, 0x0a, 0x43, 0x6f, 0x6d, 0x70, 0x61, 0x72, 0x61, 0x74, 0x6f, 0x72, 0x12,
	0x16, 0x0a, 0x12, 0x43, 0x4f, 0x4d, 0x50, 0x41, 0x52, 0x41, 0x54, 0x4f, 0x52, 0x5f, 0x55, 0x4e,
	0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x1b, 0x0a, 0x17, 0x43, 0x4f, 0x4d, 0x50, 0x41,
	0x52, 0x41, 0x54, 0x4f, 0x52, 0x5f, 0x47, 0x52, 0x45, 0x41, 0x54, 0x45, 0x52, 0x5f, 0x54, 0x48,
	0x41, 0x4e, 0x10, 0x01, 0x12, 0x25, 0x0a, 0x21, 0x43, 0x4f, 0x4d, 0x50, 0x41, 0x52, 0x41, 0x54,
	0x4f, 0x52, 0x5f, 0x47, 0x52, 0x45, 0x41, 0x54, 0x45, 0x52, 0x5f, 0x54, 0x48, 0x41, 0x4e, 0x5f,
	0x4f, 0x52, 0x5f, 0x45, 0x51, 0x55, 0x41, 0x4c, 0x53, 0x10, 0x02, 0x12, 0x18, 0x0a, 0x14, 0x43,
	0x4f, 0x4d, 0x50, 0x41, 0x52, 0x41, 0x54, 0x4f, 0x52, 0x5f, 0x4c, 0x45, 0x53, 0x53, 0x5f, 0x54,
	0x48, 0x41, 0x4e, 0x10, 0x03, 0x12, 0x22, 0x0a, 0x1e, 0x43, 0x4f, 0x4d, 0x50, 0x41, 0x52, 0x41,
	0x54, 0x4f, 0x52, 0x5f, 0x4c, 0x45, 0x53, 0x53, 0x5f, 0x54, 0x48, 0x41, 0x4e, 0x5f, 0x4f, 0x52,
	0x5f, 0x45, 0x51, 0x55, 0x41, 0x4c, 0x53, 0x10, 0x04, 0x42, 0x2a, 0x5a, 0x28, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x72, 0x6f, 0x74, 0x68, 0x65, 0x72, 0x6c,
	0x6f, 0x67, 0x69, 0x63, 0x2f, 0x67, 0x72, 0x61, 0x6d, 0x6f, 0x70, 0x68, 0x69, 0x6c, 0x65, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_classification_proto_rawDescOnce sync.Once
	file_classification_proto_rawDescData = file_classification_proto_rawDesc
)

func file_classification_proto_rawDescGZIP() []byte {
	file_classification_proto_rawDescOnce.Do(func() {
		file_classification_proto_rawDescData = protoimpl.X.CompressGZIP(file_classification_proto_rawDescData)
	})
	return file_classification_proto_rawDescData
}

var file_classification_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_classification_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_classification_proto_goTypes = []interface{}{
	(Comparator)(0),              // 0: gramophile.Comparator
	(*BooleanSelector)(nil),      // 1: gramophile.BooleanSelector
	(*IntSelector)(nil),          // 2: gramophile.IntSelector
	(*ClassificationRule)(nil),   // 3: gramophile.ClassificationRule
	(*Classifier)(nil),           // 4: gramophile.Classifier
	(*ClassificationConfig)(nil), // 5: gramophile.ClassificationConfig
}
var file_classification_proto_depIdxs = []int32{
	0, // 0: gramophile.IntSelector.comp:type_name -> gramophile.Comparator
	1, // 1: gramophile.ClassificationRule.boolean_selector:type_name -> gramophile.BooleanSelector
	2, // 2: gramophile.ClassificationRule.int_selector:type_name -> gramophile.IntSelector
	3, // 3: gramophile.Classifier.rule:type_name -> gramophile.ClassificationRule
	4, // 4: gramophile.ClassificationConfig.classifiers:type_name -> gramophile.Classifier
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_classification_proto_init() }
func file_classification_proto_init() {
	if File_classification_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_classification_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BooleanSelector); i {
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
		file_classification_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IntSelector); i {
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
		file_classification_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ClassificationRule); i {
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
		file_classification_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Classifier); i {
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
		file_classification_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ClassificationConfig); i {
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
	file_classification_proto_msgTypes[2].OneofWrappers = []interface{}{
		(*ClassificationRule_BooleanSelector)(nil),
		(*ClassificationRule_IntSelector)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_classification_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_classification_proto_goTypes,
		DependencyIndexes: file_classification_proto_depIdxs,
		EnumInfos:         file_classification_proto_enumTypes,
		MessageInfos:      file_classification_proto_msgTypes,
	}.Build()
	File_classification_proto = out.File
	file_classification_proto_rawDesc = nil
	file_classification_proto_goTypes = nil
	file_classification_proto_depIdxs = nil
}
