// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.5
// source: github.com/dogmatiq/veracity/internal/aggregate/journal.proto

package aggregate

import (
	envelopespec "github.com/dogmatiq/interopspec/envelopespec"
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

type JournalRecord struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to OneOf:
	//
	//	*JournalRecord_Revision
	OneOf isJournalRecord_OneOf `protobuf_oneof:"OneOf"`
}

func (x *JournalRecord) Reset() {
	*x = JournalRecord{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *JournalRecord) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JournalRecord) ProtoMessage() {}

func (x *JournalRecord) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JournalRecord.ProtoReflect.Descriptor instead.
func (*JournalRecord) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDescGZIP(), []int{0}
}

func (m *JournalRecord) GetOneOf() isJournalRecord_OneOf {
	if m != nil {
		return m.OneOf
	}
	return nil
}

func (x *JournalRecord) GetRevision() *RevisionRecord {
	if x, ok := x.GetOneOf().(*JournalRecord_Revision); ok {
		return x.Revision
	}
	return nil
}

type isJournalRecord_OneOf interface {
	isJournalRecord_OneOf()
}

type JournalRecord_Revision struct {
	Revision *RevisionRecord `protobuf:"bytes,1,opt,name=revision,proto3,oneof"`
}

func (*JournalRecord_Revision) isJournalRecord_OneOf() {}

type RevisionRecord struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CommandId string            `protobuf:"bytes,1,opt,name=command_id,json=commandId,proto3" json:"command_id,omitempty"`
	Actions   []*RevisionAction `protobuf:"bytes,2,rep,name=actions,proto3" json:"actions,omitempty"`
}

func (x *RevisionRecord) Reset() {
	*x = RevisionRecord{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RevisionRecord) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RevisionRecord) ProtoMessage() {}

func (x *RevisionRecord) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RevisionRecord.ProtoReflect.Descriptor instead.
func (*RevisionRecord) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDescGZIP(), []int{1}
}

func (x *RevisionRecord) GetCommandId() string {
	if x != nil {
		return x.CommandId
	}
	return ""
}

func (x *RevisionRecord) GetActions() []*RevisionAction {
	if x != nil {
		return x.Actions
	}
	return nil
}

type RevisionAction struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to OneOf:
	//
	//	*RevisionAction_RecordEvent
	//	*RevisionAction_Log
	//	*RevisionAction_Destroy
	OneOf isRevisionAction_OneOf `protobuf_oneof:"OneOf"`
}

func (x *RevisionAction) Reset() {
	*x = RevisionAction{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RevisionAction) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RevisionAction) ProtoMessage() {}

func (x *RevisionAction) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RevisionAction.ProtoReflect.Descriptor instead.
func (*RevisionAction) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDescGZIP(), []int{2}
}

func (m *RevisionAction) GetOneOf() isRevisionAction_OneOf {
	if m != nil {
		return m.OneOf
	}
	return nil
}

func (x *RevisionAction) GetRecordEvent() *RecordEventAction {
	if x, ok := x.GetOneOf().(*RevisionAction_RecordEvent); ok {
		return x.RecordEvent
	}
	return nil
}

func (x *RevisionAction) GetLog() *LogAction {
	if x, ok := x.GetOneOf().(*RevisionAction_Log); ok {
		return x.Log
	}
	return nil
}

func (x *RevisionAction) GetDestroy() *DestroyAction {
	if x, ok := x.GetOneOf().(*RevisionAction_Destroy); ok {
		return x.Destroy
	}
	return nil
}

type isRevisionAction_OneOf interface {
	isRevisionAction_OneOf()
}

type RevisionAction_RecordEvent struct {
	RecordEvent *RecordEventAction `protobuf:"bytes,1,opt,name=record_event,json=recordEvent,proto3,oneof"`
}

type RevisionAction_Log struct {
	Log *LogAction `protobuf:"bytes,2,opt,name=log,proto3,oneof"`
}

type RevisionAction_Destroy struct {
	Destroy *DestroyAction `protobuf:"bytes,3,opt,name=destroy,proto3,oneof"`
}

func (*RevisionAction_RecordEvent) isRevisionAction_OneOf() {}

func (*RevisionAction_Log) isRevisionAction_OneOf() {}

func (*RevisionAction_Destroy) isRevisionAction_OneOf() {}

type RecordEventAction struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Envelope *envelopespec.Envelope `protobuf:"bytes,1,opt,name=envelope,proto3" json:"envelope,omitempty"`
}

func (x *RecordEventAction) Reset() {
	*x = RecordEventAction{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RecordEventAction) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RecordEventAction) ProtoMessage() {}

func (x *RecordEventAction) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RecordEventAction.ProtoReflect.Descriptor instead.
func (*RecordEventAction) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDescGZIP(), []int{3}
}

func (x *RecordEventAction) GetEnvelope() *envelopespec.Envelope {
	if x != nil {
		return x.Envelope
	}
	return nil
}

type LogAction struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Message string `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *LogAction) Reset() {
	*x = LogAction{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LogAction) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LogAction) ProtoMessage() {}

func (x *LogAction) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LogAction.ProtoReflect.Descriptor instead.
func (*LogAction) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDescGZIP(), []int{4}
}

func (x *LogAction) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type DestroyAction struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	IsCancelled bool `protobuf:"varint,1,opt,name=is_cancelled,json=isCancelled,proto3" json:"is_cancelled,omitempty"`
}

func (x *DestroyAction) Reset() {
	*x = DestroyAction{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DestroyAction) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DestroyAction) ProtoMessage() {}

func (x *DestroyAction) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DestroyAction.ProtoReflect.Descriptor instead.
func (*DestroyAction) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDescGZIP(), []int{5}
}

func (x *DestroyAction) GetIsCancelled() bool {
	if x != nil {
		return x.IsCancelled
	}
	return false
}

var File_github_com_dogmatiq_veracity_internal_aggregate_journal_proto protoreflect.FileDescriptor

var file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDesc = []byte{
	0x0a, 0x3d, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x67,
	0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74,
	0x65, 0x2f, 0x6a, 0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x12, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67,
	0x61, 0x74, 0x65, 0x1a, 0x3b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x64, 0x6f, 0x67, 0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6f, 0x70,
	0x73, 0x70, 0x65, 0x63, 0x2f, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x73, 0x70, 0x65,
	0x63, 0x2f, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0x5a, 0x0a, 0x0d, 0x4a, 0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x52, 0x65, 0x63, 0x6f, 0x72,
	0x64, 0x12, 0x40, 0x0a, 0x08, 0x72, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x61,
	0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x2e, 0x52, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f,
	0x6e, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x48, 0x00, 0x52, 0x08, 0x72, 0x65, 0x76, 0x69, 0x73,
	0x69, 0x6f, 0x6e, 0x42, 0x07, 0x0a, 0x05, 0x4f, 0x6e, 0x65, 0x4f, 0x66, 0x22, 0x6d, 0x0a, 0x0e,
	0x52, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x12, 0x1d,
	0x0a, 0x0a, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x09, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x49, 0x64, 0x12, 0x3c, 0x0a,
	0x07, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x22,
	0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67,
	0x61, 0x74, 0x65, 0x2e, 0x52, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x41, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x52, 0x07, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x22, 0xd7, 0x01, 0x0a, 0x0e,
	0x52, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x4a,
	0x0a, 0x0c, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x5f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e,
	0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x2e, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64,
	0x45, 0x76, 0x65, 0x6e, 0x74, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x48, 0x00, 0x52, 0x0b, 0x72,
	0x65, 0x63, 0x6f, 0x72, 0x64, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x31, 0x0a, 0x03, 0x6c, 0x6f,
	0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69,
	0x74, 0x79, 0x2e, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x2e, 0x4c, 0x6f, 0x67,
	0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x48, 0x00, 0x52, 0x03, 0x6c, 0x6f, 0x67, 0x12, 0x3d, 0x0a,
	0x07, 0x64, 0x65, 0x73, 0x74, 0x72, 0x6f, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x21,
	0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67,
	0x61, 0x74, 0x65, 0x2e, 0x44, 0x65, 0x73, 0x74, 0x72, 0x6f, 0x79, 0x41, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x48, 0x00, 0x52, 0x07, 0x64, 0x65, 0x73, 0x74, 0x72, 0x6f, 0x79, 0x42, 0x07, 0x0a, 0x05,
	0x4f, 0x6e, 0x65, 0x4f, 0x66, 0x22, 0x54, 0x0a, 0x11, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x45,
	0x76, 0x65, 0x6e, 0x74, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x3f, 0x0a, 0x08, 0x65, 0x6e,
	0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64,
	0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e,
	0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x2e, 0x45, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70,
	0x65, 0x52, 0x08, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x22, 0x25, 0x0a, 0x09, 0x4c,
	0x6f, 0x67, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x22, 0x32, 0x0a, 0x0d, 0x44, 0x65, 0x73, 0x74, 0x72, 0x6f, 0x79, 0x41, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x12, 0x21, 0x0a, 0x0c, 0x69, 0x73, 0x5f, 0x63, 0x61, 0x6e, 0x63, 0x65, 0x6c,
	0x6c, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0b, 0x69, 0x73, 0x43, 0x61, 0x6e,
	0x63, 0x65, 0x6c, 0x6c, 0x65, 0x64, 0x42, 0x31, 0x5a, 0x2f, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65,
	0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f,
	0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDescOnce sync.Once
	file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDescData = file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDesc
)

func file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDescGZIP() []byte {
	file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDescOnce.Do(func() {
		file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDescData)
	})
	return file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDescData
}

var file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_goTypes = []interface{}{
	(*JournalRecord)(nil),         // 0: veracity.aggregate.JournalRecord
	(*RevisionRecord)(nil),        // 1: veracity.aggregate.RevisionRecord
	(*RevisionAction)(nil),        // 2: veracity.aggregate.RevisionAction
	(*RecordEventAction)(nil),     // 3: veracity.aggregate.RecordEventAction
	(*LogAction)(nil),             // 4: veracity.aggregate.LogAction
	(*DestroyAction)(nil),         // 5: veracity.aggregate.DestroyAction
	(*envelopespec.Envelope)(nil), // 6: dogma.interop.v1.envelope.Envelope
}
var file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_depIdxs = []int32{
	1, // 0: veracity.aggregate.JournalRecord.revision:type_name -> veracity.aggregate.RevisionRecord
	2, // 1: veracity.aggregate.RevisionRecord.actions:type_name -> veracity.aggregate.RevisionAction
	3, // 2: veracity.aggregate.RevisionAction.record_event:type_name -> veracity.aggregate.RecordEventAction
	4, // 3: veracity.aggregate.RevisionAction.log:type_name -> veracity.aggregate.LogAction
	5, // 4: veracity.aggregate.RevisionAction.destroy:type_name -> veracity.aggregate.DestroyAction
	6, // 5: veracity.aggregate.RecordEventAction.envelope:type_name -> dogma.interop.v1.envelope.Envelope
	6, // [6:6] is the sub-list for method output_type
	6, // [6:6] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_init() }
func file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_init() {
	if File_github_com_dogmatiq_veracity_internal_aggregate_journal_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*JournalRecord); i {
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
		file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RevisionRecord); i {
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
		file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RevisionAction); i {
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
		file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RecordEventAction); i {
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
		file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LogAction); i {
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
		file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DestroyAction); i {
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
	file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*JournalRecord_Revision)(nil),
	}
	file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes[2].OneofWrappers = []interface{}{
		(*RevisionAction_RecordEvent)(nil),
		(*RevisionAction_Log)(nil),
		(*RevisionAction_Destroy)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_goTypes,
		DependencyIndexes: file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_depIdxs,
		MessageInfos:      file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_msgTypes,
	}.Build()
	File_github_com_dogmatiq_veracity_internal_aggregate_journal_proto = out.File
	file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_rawDesc = nil
	file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_goTypes = nil
	file_github_com_dogmatiq_veracity_internal_aggregate_journal_proto_depIdxs = nil
}
