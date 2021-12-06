// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.19.1
// source: github.com/dogmatiq/veracity/journal/record.proto

package journal

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

// RecordContainer is a container for a record. It the singular Protocol Buffers
// message type that is appended to the journal's underlying log.
type RecordContainer struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Elem:
	//	*RecordContainer_CommandEnqueued
	//	*RecordContainer_CommandHandledByAggregate
	//	*RecordContainer_CommandHandledByIntegration
	//	*RecordContainer_EventHandledByProcess
	//	*RecordContainer_TimeoutHandledByProcess
	Elem isRecordContainer_Elem `protobuf_oneof:"elem"`
}

func (x *RecordContainer) Reset() {
	*x = RecordContainer{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RecordContainer) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RecordContainer) ProtoMessage() {}

func (x *RecordContainer) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RecordContainer.ProtoReflect.Descriptor instead.
func (*RecordContainer) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_journal_record_proto_rawDescGZIP(), []int{0}
}

func (m *RecordContainer) GetElem() isRecordContainer_Elem {
	if m != nil {
		return m.Elem
	}
	return nil
}

func (x *RecordContainer) GetCommandEnqueued() *CommandEnqueued {
	if x, ok := x.GetElem().(*RecordContainer_CommandEnqueued); ok {
		return x.CommandEnqueued
	}
	return nil
}

func (x *RecordContainer) GetCommandHandledByAggregate() *CommandHandledByAggregate {
	if x, ok := x.GetElem().(*RecordContainer_CommandHandledByAggregate); ok {
		return x.CommandHandledByAggregate
	}
	return nil
}

func (x *RecordContainer) GetCommandHandledByIntegration() *CommandHandledByIntegration {
	if x, ok := x.GetElem().(*RecordContainer_CommandHandledByIntegration); ok {
		return x.CommandHandledByIntegration
	}
	return nil
}

func (x *RecordContainer) GetEventHandledByProcess() *EventHandledByProcess {
	if x, ok := x.GetElem().(*RecordContainer_EventHandledByProcess); ok {
		return x.EventHandledByProcess
	}
	return nil
}

func (x *RecordContainer) GetTimeoutHandledByProcess() *TimeoutHandledByProcess {
	if x, ok := x.GetElem().(*RecordContainer_TimeoutHandledByProcess); ok {
		return x.TimeoutHandledByProcess
	}
	return nil
}

type isRecordContainer_Elem interface {
	isRecordContainer_Elem()
}

type RecordContainer_CommandEnqueued struct {
	CommandEnqueued *CommandEnqueued `protobuf:"bytes,1,opt,name=command_enqueued,json=commandEnqueued,proto3,oneof"`
}

type RecordContainer_CommandHandledByAggregate struct {
	CommandHandledByAggregate *CommandHandledByAggregate `protobuf:"bytes,2,opt,name=command_handled_by_aggregate,json=commandHandledByAggregate,proto3,oneof"`
}

type RecordContainer_CommandHandledByIntegration struct {
	CommandHandledByIntegration *CommandHandledByIntegration `protobuf:"bytes,3,opt,name=command_handled_by_integration,json=commandHandledByIntegration,proto3,oneof"`
}

type RecordContainer_EventHandledByProcess struct {
	EventHandledByProcess *EventHandledByProcess `protobuf:"bytes,4,opt,name=event_handled_by_process,json=eventHandledByProcess,proto3,oneof"`
}

type RecordContainer_TimeoutHandledByProcess struct {
	TimeoutHandledByProcess *TimeoutHandledByProcess `protobuf:"bytes,5,opt,name=timeout_handled_by_process,json=timeoutHandledByProcess,proto3,oneof"`
}

func (*RecordContainer_CommandEnqueued) isRecordContainer_Elem() {}

func (*RecordContainer_CommandHandledByAggregate) isRecordContainer_Elem() {}

func (*RecordContainer_CommandHandledByIntegration) isRecordContainer_Elem() {}

func (*RecordContainer_EventHandledByProcess) isRecordContainer_Elem() {}

func (*RecordContainer_TimeoutHandledByProcess) isRecordContainer_Elem() {}

type CommandEnqueued struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Envelope *envelopespec.Envelope `protobuf:"bytes,1,opt,name=Envelope,proto3" json:"Envelope,omitempty"`
}

func (x *CommandEnqueued) Reset() {
	*x = CommandEnqueued{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CommandEnqueued) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CommandEnqueued) ProtoMessage() {}

func (x *CommandEnqueued) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CommandEnqueued.ProtoReflect.Descriptor instead.
func (*CommandEnqueued) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_journal_record_proto_rawDescGZIP(), []int{1}
}

func (x *CommandEnqueued) GetEnvelope() *envelopespec.Envelope {
	if x != nil {
		return x.Envelope
	}
	return nil
}

type AggregateInstance struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id        string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Destroyed bool   `protobuf:"varint,2,opt,name=destroyed,proto3" json:"destroyed,omitempty"`
}

func (x *AggregateInstance) Reset() {
	*x = AggregateInstance{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AggregateInstance) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AggregateInstance) ProtoMessage() {}

func (x *AggregateInstance) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AggregateInstance.ProtoReflect.Descriptor instead.
func (*AggregateInstance) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_journal_record_proto_rawDescGZIP(), []int{2}
}

func (x *AggregateInstance) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *AggregateInstance) GetDestroyed() bool {
	if x != nil {
		return x.Destroyed
	}
	return false
}

type CommandHandledByAggregate struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Handler   *envelopespec.Identity   `protobuf:"bytes,1,opt,name=handler,proto3" json:"handler,omitempty"`
	MessageId string                   `protobuf:"bytes,2,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	Instance  *AggregateInstance       `protobuf:"bytes,3,opt,name=instance,proto3" json:"instance,omitempty"`
	Events    []*envelopespec.Envelope `protobuf:"bytes,4,rep,name=events,proto3" json:"events,omitempty"`
}

func (x *CommandHandledByAggregate) Reset() {
	*x = CommandHandledByAggregate{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CommandHandledByAggregate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CommandHandledByAggregate) ProtoMessage() {}

func (x *CommandHandledByAggregate) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CommandHandledByAggregate.ProtoReflect.Descriptor instead.
func (*CommandHandledByAggregate) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_journal_record_proto_rawDescGZIP(), []int{3}
}

func (x *CommandHandledByAggregate) GetHandler() *envelopespec.Identity {
	if x != nil {
		return x.Handler
	}
	return nil
}

func (x *CommandHandledByAggregate) GetMessageId() string {
	if x != nil {
		return x.MessageId
	}
	return ""
}

func (x *CommandHandledByAggregate) GetInstance() *AggregateInstance {
	if x != nil {
		return x.Instance
	}
	return nil
}

func (x *CommandHandledByAggregate) GetEvents() []*envelopespec.Envelope {
	if x != nil {
		return x.Events
	}
	return nil
}

type CommandHandledByIntegration struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Handler   *envelopespec.Identity   `protobuf:"bytes,1,opt,name=handler,proto3" json:"handler,omitempty"`
	MessageId string                   `protobuf:"bytes,2,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	Events    []*envelopespec.Envelope `protobuf:"bytes,3,rep,name=events,proto3" json:"events,omitempty"`
}

func (x *CommandHandledByIntegration) Reset() {
	*x = CommandHandledByIntegration{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CommandHandledByIntegration) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CommandHandledByIntegration) ProtoMessage() {}

func (x *CommandHandledByIntegration) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CommandHandledByIntegration.ProtoReflect.Descriptor instead.
func (*CommandHandledByIntegration) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_journal_record_proto_rawDescGZIP(), []int{4}
}

func (x *CommandHandledByIntegration) GetHandler() *envelopespec.Identity {
	if x != nil {
		return x.Handler
	}
	return nil
}

func (x *CommandHandledByIntegration) GetMessageId() string {
	if x != nil {
		return x.MessageId
	}
	return ""
}

func (x *CommandHandledByIntegration) GetEvents() []*envelopespec.Envelope {
	if x != nil {
		return x.Events
	}
	return nil
}

type ProcessInstance struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id        string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Ended     bool   `protobuf:"varint,2,opt,name=ended,proto3" json:"ended,omitempty"`
	MediaType string `protobuf:"bytes,3,opt,name=media_type,json=mediaType,proto3" json:"media_type,omitempty"`
	Data      []byte `protobuf:"bytes,4,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *ProcessInstance) Reset() {
	*x = ProcessInstance{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProcessInstance) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProcessInstance) ProtoMessage() {}

func (x *ProcessInstance) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProcessInstance.ProtoReflect.Descriptor instead.
func (*ProcessInstance) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_journal_record_proto_rawDescGZIP(), []int{5}
}

func (x *ProcessInstance) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *ProcessInstance) GetEnded() bool {
	if x != nil {
		return x.Ended
	}
	return false
}

func (x *ProcessInstance) GetMediaType() string {
	if x != nil {
		return x.MediaType
	}
	return ""
}

func (x *ProcessInstance) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

type EventHandledByProcess struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Handler           *envelopespec.Identity   `protobuf:"bytes,1,opt,name=handler,proto3" json:"handler,omitempty"`
	SourceApplication *envelopespec.Identity   `protobuf:"bytes,2,opt,name=source_application,json=sourceApplication,proto3" json:"source_application,omitempty"`
	SourceOffset      uint64                   `protobuf:"varint,3,opt,name=source_offset,json=sourceOffset,proto3" json:"source_offset,omitempty"`
	Instance          *ProcessInstance         `protobuf:"bytes,4,opt,name=instance,proto3" json:"instance,omitempty"`
	Commands          []*envelopespec.Envelope `protobuf:"bytes,5,rep,name=commands,proto3" json:"commands,omitempty"`
	Timeouts          []*envelopespec.Envelope `protobuf:"bytes,6,rep,name=timeouts,proto3" json:"timeouts,omitempty"`
}

func (x *EventHandledByProcess) Reset() {
	*x = EventHandledByProcess{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventHandledByProcess) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventHandledByProcess) ProtoMessage() {}

func (x *EventHandledByProcess) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventHandledByProcess.ProtoReflect.Descriptor instead.
func (*EventHandledByProcess) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_journal_record_proto_rawDescGZIP(), []int{6}
}

func (x *EventHandledByProcess) GetHandler() *envelopespec.Identity {
	if x != nil {
		return x.Handler
	}
	return nil
}

func (x *EventHandledByProcess) GetSourceApplication() *envelopespec.Identity {
	if x != nil {
		return x.SourceApplication
	}
	return nil
}

func (x *EventHandledByProcess) GetSourceOffset() uint64 {
	if x != nil {
		return x.SourceOffset
	}
	return 0
}

func (x *EventHandledByProcess) GetInstance() *ProcessInstance {
	if x != nil {
		return x.Instance
	}
	return nil
}

func (x *EventHandledByProcess) GetCommands() []*envelopespec.Envelope {
	if x != nil {
		return x.Commands
	}
	return nil
}

func (x *EventHandledByProcess) GetTimeouts() []*envelopespec.Envelope {
	if x != nil {
		return x.Timeouts
	}
	return nil
}

type TimeoutHandledByProcess struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Handler   *envelopespec.Identity   `protobuf:"bytes,1,opt,name=handler,proto3" json:"handler,omitempty"`
	MessageId string                   `protobuf:"bytes,2,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	Instance  *ProcessInstance         `protobuf:"bytes,3,opt,name=instance,proto3" json:"instance,omitempty"`
	Commands  []*envelopespec.Envelope `protobuf:"bytes,4,rep,name=commands,proto3" json:"commands,omitempty"`
	Timeouts  []*envelopespec.Envelope `protobuf:"bytes,5,rep,name=timeouts,proto3" json:"timeouts,omitempty"`
}

func (x *TimeoutHandledByProcess) Reset() {
	*x = TimeoutHandledByProcess{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TimeoutHandledByProcess) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TimeoutHandledByProcess) ProtoMessage() {}

func (x *TimeoutHandledByProcess) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TimeoutHandledByProcess.ProtoReflect.Descriptor instead.
func (*TimeoutHandledByProcess) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_journal_record_proto_rawDescGZIP(), []int{7}
}

func (x *TimeoutHandledByProcess) GetHandler() *envelopespec.Identity {
	if x != nil {
		return x.Handler
	}
	return nil
}

func (x *TimeoutHandledByProcess) GetMessageId() string {
	if x != nil {
		return x.MessageId
	}
	return ""
}

func (x *TimeoutHandledByProcess) GetInstance() *ProcessInstance {
	if x != nil {
		return x.Instance
	}
	return nil
}

func (x *TimeoutHandledByProcess) GetCommands() []*envelopespec.Envelope {
	if x != nil {
		return x.Commands
	}
	return nil
}

func (x *TimeoutHandledByProcess) GetTimeouts() []*envelopespec.Envelope {
	if x != nil {
		return x.Timeouts
	}
	return nil
}

var File_github_com_dogmatiq_veracity_journal_record_proto protoreflect.FileDescriptor

var file_github_com_dogmatiq_veracity_journal_record_proto_rawDesc = []byte{
	0x0a, 0x31, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x67,
	0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2f, 0x6a,
	0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x13, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x6a, 0x6f,
	0x75, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x76, 0x31, 0x1a, 0x3b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x69, 0x6e,
	0x74, 0x65, 0x72, 0x6f, 0x70, 0x73, 0x70, 0x65, 0x63, 0x2f, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f,
	0x70, 0x65, 0x73, 0x70, 0x65, 0x63, 0x2f, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xac, 0x04, 0x0a, 0x0f, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64,
	0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x12, 0x51, 0x0a, 0x10, 0x63, 0x6f, 0x6d,
	0x6d, 0x61, 0x6e, 0x64, 0x5f, 0x65, 0x6e, 0x71, 0x75, 0x65, 0x75, 0x65, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x6a,
	0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e,
	0x64, 0x45, 0x6e, 0x71, 0x75, 0x65, 0x75, 0x65, 0x64, 0x48, 0x00, 0x52, 0x0f, 0x63, 0x6f, 0x6d,
	0x6d, 0x61, 0x6e, 0x64, 0x45, 0x6e, 0x71, 0x75, 0x65, 0x75, 0x65, 0x64, 0x12, 0x71, 0x0a, 0x1c,
	0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x5f, 0x68, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64, 0x5f,
	0x62, 0x79, 0x5f, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x6a, 0x6f,
	0x75, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64,
	0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64, 0x42, 0x79, 0x41, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61,
	0x74, 0x65, 0x48, 0x00, 0x52, 0x19, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x48, 0x61, 0x6e,
	0x64, 0x6c, 0x65, 0x64, 0x42, 0x79, 0x41, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x12,
	0x77, 0x0a, 0x1e, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x5f, 0x68, 0x61, 0x6e, 0x64, 0x6c,
	0x65, 0x64, 0x5f, 0x62, 0x79, 0x5f, 0x69, 0x6e, 0x74, 0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x30, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69,
	0x74, 0x79, 0x2e, 0x6a, 0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f,
	0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64, 0x42, 0x79, 0x49, 0x6e,
	0x74, 0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x48, 0x00, 0x52, 0x1b, 0x63, 0x6f, 0x6d,
	0x6d, 0x61, 0x6e, 0x64, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64, 0x42, 0x79, 0x49, 0x6e, 0x74,
	0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x65, 0x0a, 0x18, 0x65, 0x76, 0x65, 0x6e,
	0x74, 0x5f, 0x68, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64, 0x5f, 0x62, 0x79, 0x5f, 0x70, 0x72, 0x6f,
	0x63, 0x65, 0x73, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2a, 0x2e, 0x76, 0x65, 0x72,
	0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x6a, 0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x76, 0x31,
	0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64, 0x42, 0x79, 0x50,
	0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x48, 0x00, 0x52, 0x15, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x48,
	0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64, 0x42, 0x79, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x12,
	0x6b, 0x0a, 0x1a, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x5f, 0x68, 0x61, 0x6e, 0x64, 0x6c,
	0x65, 0x64, 0x5f, 0x62, 0x79, 0x5f, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x6a,
	0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x6f, 0x75,
	0x74, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64, 0x42, 0x79, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73,
	0x73, 0x48, 0x00, 0x52, 0x17, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x48, 0x61, 0x6e, 0x64,
	0x6c, 0x65, 0x64, 0x42, 0x79, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x42, 0x06, 0x0a, 0x04,
	0x65, 0x6c, 0x65, 0x6d, 0x22, 0x52, 0x0a, 0x0f, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x45,
	0x6e, 0x71, 0x75, 0x65, 0x75, 0x65, 0x64, 0x12, 0x3f, 0x0a, 0x08, 0x45, 0x6e, 0x76, 0x65, 0x6c,
	0x6f, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f, 0x67, 0x6d,
	0x61, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x6e, 0x76,
	0x65, 0x6c, 0x6f, 0x70, 0x65, 0x2e, 0x45, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x52, 0x08,
	0x45, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x22, 0x41, 0x0a, 0x11, 0x41, 0x67, 0x67, 0x72,
	0x65, 0x67, 0x61, 0x74, 0x65, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x12, 0x0e, 0x0a,
	0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x1c, 0x0a,
	0x09, 0x64, 0x65, 0x73, 0x74, 0x72, 0x6f, 0x79, 0x65, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08,
	0x52, 0x09, 0x64, 0x65, 0x73, 0x74, 0x72, 0x6f, 0x79, 0x65, 0x64, 0x22, 0xfa, 0x01, 0x0a, 0x19,
	0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64, 0x42, 0x79,
	0x41, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x12, 0x3d, 0x0a, 0x07, 0x68, 0x61, 0x6e,
	0x64, 0x6c, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f, 0x67,
	0x6d, 0x61, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x6e,
	0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x2e, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x52,
	0x07, 0x68, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x72, 0x12, 0x1d, 0x0a, 0x0a, 0x6d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x49, 0x64, 0x12, 0x42, 0x0a, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61,
	0x6e, 0x63, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x76, 0x65, 0x72, 0x61,
	0x63, 0x69, 0x74, 0x79, 0x2e, 0x6a, 0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x76, 0x31, 0x2e,
	0x41, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63,
	0x65, 0x52, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x12, 0x3b, 0x0a, 0x06, 0x65,
	0x76, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f,
	0x67, 0x6d, 0x61, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65,
	0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x2e, 0x45, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65,
	0x52, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x22, 0xb8, 0x01, 0x0a, 0x1b, 0x43, 0x6f, 0x6d,
	0x6d, 0x61, 0x6e, 0x64, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64, 0x42, 0x79, 0x49, 0x6e, 0x74,
	0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x3d, 0x0a, 0x07, 0x68, 0x61, 0x6e, 0x64,
	0x6c, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f, 0x67, 0x6d,
	0x61, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x6e, 0x76,
	0x65, 0x6c, 0x6f, 0x70, 0x65, 0x2e, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x52, 0x07,
	0x68, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x72, 0x12, 0x1d, 0x0a, 0x0a, 0x6d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x49, 0x64, 0x12, 0x3b, 0x0a, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73,
	0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f,
	0x70, 0x65, 0x2e, 0x45, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x52, 0x06, 0x65, 0x76, 0x65,
	0x6e, 0x74, 0x73, 0x22, 0x6a, 0x0a, 0x0f, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x49, 0x6e,
	0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x05, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x12, 0x1d, 0x0a, 0x0a,
	0x6d, 0x65, 0x64, 0x69, 0x61, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x09, 0x6d, 0x65, 0x64, 0x69, 0x61, 0x54, 0x79, 0x70, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x64,
	0x61, 0x74, 0x61, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22,
	0x93, 0x03, 0x0a, 0x15, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64,
	0x42, 0x79, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x12, 0x3d, 0x0a, 0x07, 0x68, 0x61, 0x6e,
	0x64, 0x6c, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f, 0x67,
	0x6d, 0x61, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x6e,
	0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x2e, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x52,
	0x07, 0x68, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x72, 0x12, 0x52, 0x0a, 0x12, 0x73, 0x6f, 0x75, 0x72,
	0x63, 0x65, 0x5f, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x69, 0x6e, 0x74,
	0x65, 0x72, 0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65,
	0x2e, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x52, 0x11, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x41, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x23, 0x0a, 0x0d,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x5f, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x04, 0x52, 0x0c, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x4f, 0x66, 0x66, 0x73, 0x65,
	0x74, 0x12, 0x40, 0x0a, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x6a,
	0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73,
	0x73, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x52, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61,
	0x6e, 0x63, 0x65, 0x12, 0x3f, 0x0a, 0x08, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x73, 0x18,
	0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x69, 0x6e,
	0x74, 0x65, 0x72, 0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70,
	0x65, 0x2e, 0x45, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x52, 0x08, 0x63, 0x6f, 0x6d, 0x6d,
	0x61, 0x6e, 0x64, 0x73, 0x12, 0x3f, 0x0a, 0x08, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x73,
	0x18, 0x06, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f,
	0x70, 0x65, 0x2e, 0x45, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x52, 0x08, 0x74, 0x69, 0x6d,
	0x65, 0x6f, 0x75, 0x74, 0x73, 0x22, 0xbb, 0x02, 0x0a, 0x17, 0x54, 0x69, 0x6d, 0x65, 0x6f, 0x75,
	0x74, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64, 0x42, 0x79, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73,
	0x73, 0x12, 0x3d, 0x0a, 0x07, 0x68, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x2e, 0x49,
	0x64, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x52, 0x07, 0x68, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x72,
	0x12, 0x1d, 0x0a, 0x0a, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x49, 0x64, 0x12,
	0x40, 0x0a, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x24, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x6a, 0x6f, 0x75,
	0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x49,
	0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x52, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63,
	0x65, 0x12, 0x3f, 0x0a, 0x08, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x73, 0x18, 0x04, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x69, 0x6e, 0x74, 0x65,
	0x72, 0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x2e,
	0x45, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x52, 0x08, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e,
	0x64, 0x73, 0x12, 0x3f, 0x0a, 0x08, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x73, 0x18, 0x05,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x69, 0x6e, 0x74,
	0x65, 0x72, 0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65,
	0x2e, 0x45, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x52, 0x08, 0x74, 0x69, 0x6d, 0x65, 0x6f,
	0x75, 0x74, 0x73, 0x42, 0x26, 0x5a, 0x24, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63,
	0x69, 0x74, 0x79, 0x2f, 0x6a, 0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_github_com_dogmatiq_veracity_journal_record_proto_rawDescOnce sync.Once
	file_github_com_dogmatiq_veracity_journal_record_proto_rawDescData = file_github_com_dogmatiq_veracity_journal_record_proto_rawDesc
)

func file_github_com_dogmatiq_veracity_journal_record_proto_rawDescGZIP() []byte {
	file_github_com_dogmatiq_veracity_journal_record_proto_rawDescOnce.Do(func() {
		file_github_com_dogmatiq_veracity_journal_record_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_dogmatiq_veracity_journal_record_proto_rawDescData)
	})
	return file_github_com_dogmatiq_veracity_journal_record_proto_rawDescData
}

var file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_github_com_dogmatiq_veracity_journal_record_proto_goTypes = []interface{}{
	(*RecordContainer)(nil),             // 0: veracity.journal.v1.RecordContainer
	(*CommandEnqueued)(nil),             // 1: veracity.journal.v1.CommandEnqueued
	(*AggregateInstance)(nil),           // 2: veracity.journal.v1.AggregateInstance
	(*CommandHandledByAggregate)(nil),   // 3: veracity.journal.v1.CommandHandledByAggregate
	(*CommandHandledByIntegration)(nil), // 4: veracity.journal.v1.CommandHandledByIntegration
	(*ProcessInstance)(nil),             // 5: veracity.journal.v1.ProcessInstance
	(*EventHandledByProcess)(nil),       // 6: veracity.journal.v1.EventHandledByProcess
	(*TimeoutHandledByProcess)(nil),     // 7: veracity.journal.v1.TimeoutHandledByProcess
	(*envelopespec.Envelope)(nil),       // 8: dogma.interop.v1.envelope.Envelope
	(*envelopespec.Identity)(nil),       // 9: dogma.interop.v1.envelope.Identity
}
var file_github_com_dogmatiq_veracity_journal_record_proto_depIdxs = []int32{
	1,  // 0: veracity.journal.v1.RecordContainer.command_enqueued:type_name -> veracity.journal.v1.CommandEnqueued
	3,  // 1: veracity.journal.v1.RecordContainer.command_handled_by_aggregate:type_name -> veracity.journal.v1.CommandHandledByAggregate
	4,  // 2: veracity.journal.v1.RecordContainer.command_handled_by_integration:type_name -> veracity.journal.v1.CommandHandledByIntegration
	6,  // 3: veracity.journal.v1.RecordContainer.event_handled_by_process:type_name -> veracity.journal.v1.EventHandledByProcess
	7,  // 4: veracity.journal.v1.RecordContainer.timeout_handled_by_process:type_name -> veracity.journal.v1.TimeoutHandledByProcess
	8,  // 5: veracity.journal.v1.CommandEnqueued.Envelope:type_name -> dogma.interop.v1.envelope.Envelope
	9,  // 6: veracity.journal.v1.CommandHandledByAggregate.handler:type_name -> dogma.interop.v1.envelope.Identity
	2,  // 7: veracity.journal.v1.CommandHandledByAggregate.instance:type_name -> veracity.journal.v1.AggregateInstance
	8,  // 8: veracity.journal.v1.CommandHandledByAggregate.events:type_name -> dogma.interop.v1.envelope.Envelope
	9,  // 9: veracity.journal.v1.CommandHandledByIntegration.handler:type_name -> dogma.interop.v1.envelope.Identity
	8,  // 10: veracity.journal.v1.CommandHandledByIntegration.events:type_name -> dogma.interop.v1.envelope.Envelope
	9,  // 11: veracity.journal.v1.EventHandledByProcess.handler:type_name -> dogma.interop.v1.envelope.Identity
	9,  // 12: veracity.journal.v1.EventHandledByProcess.source_application:type_name -> dogma.interop.v1.envelope.Identity
	5,  // 13: veracity.journal.v1.EventHandledByProcess.instance:type_name -> veracity.journal.v1.ProcessInstance
	8,  // 14: veracity.journal.v1.EventHandledByProcess.commands:type_name -> dogma.interop.v1.envelope.Envelope
	8,  // 15: veracity.journal.v1.EventHandledByProcess.timeouts:type_name -> dogma.interop.v1.envelope.Envelope
	9,  // 16: veracity.journal.v1.TimeoutHandledByProcess.handler:type_name -> dogma.interop.v1.envelope.Identity
	5,  // 17: veracity.journal.v1.TimeoutHandledByProcess.instance:type_name -> veracity.journal.v1.ProcessInstance
	8,  // 18: veracity.journal.v1.TimeoutHandledByProcess.commands:type_name -> dogma.interop.v1.envelope.Envelope
	8,  // 19: veracity.journal.v1.TimeoutHandledByProcess.timeouts:type_name -> dogma.interop.v1.envelope.Envelope
	20, // [20:20] is the sub-list for method output_type
	20, // [20:20] is the sub-list for method input_type
	20, // [20:20] is the sub-list for extension type_name
	20, // [20:20] is the sub-list for extension extendee
	0,  // [0:20] is the sub-list for field type_name
}

func init() { file_github_com_dogmatiq_veracity_journal_record_proto_init() }
func file_github_com_dogmatiq_veracity_journal_record_proto_init() {
	if File_github_com_dogmatiq_veracity_journal_record_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RecordContainer); i {
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
		file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CommandEnqueued); i {
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
		file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AggregateInstance); i {
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
		file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CommandHandledByAggregate); i {
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
		file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CommandHandledByIntegration); i {
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
		file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProcessInstance); i {
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
		file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventHandledByProcess); i {
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
		file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TimeoutHandledByProcess); i {
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
	file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*RecordContainer_CommandEnqueued)(nil),
		(*RecordContainer_CommandHandledByAggregate)(nil),
		(*RecordContainer_CommandHandledByIntegration)(nil),
		(*RecordContainer_EventHandledByProcess)(nil),
		(*RecordContainer_TimeoutHandledByProcess)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_dogmatiq_veracity_journal_record_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_dogmatiq_veracity_journal_record_proto_goTypes,
		DependencyIndexes: file_github_com_dogmatiq_veracity_journal_record_proto_depIdxs,
		MessageInfos:      file_github_com_dogmatiq_veracity_journal_record_proto_msgTypes,
	}.Build()
	File_github_com_dogmatiq_veracity_journal_record_proto = out.File
	file_github_com_dogmatiq_veracity_journal_record_proto_rawDesc = nil
	file_github_com_dogmatiq_veracity_journal_record_proto_goTypes = nil
	file_github_com_dogmatiq_veracity_journal_record_proto_depIdxs = nil
}
