// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v4.23.1
// source: github.com/dogmatiq/veracity/internal/integration/internal/journalpb/journal.proto

package journalpb

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

// CommandEnqueued is a journal record that indicates a command has been
// enqueued for handling.
type CommandEnqueued struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Command *envelopespec.Envelope `protobuf:"bytes,1,opt,name=command,proto3" json:"command,omitempty"`
}

func (x *CommandEnqueued) Reset() {
	*x = CommandEnqueued{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CommandEnqueued) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CommandEnqueued) ProtoMessage() {}

func (x *CommandEnqueued) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes[0]
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
	return file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDescGZIP(), []int{0}
}

func (x *CommandEnqueued) GetCommand() *envelopespec.Envelope {
	if x != nil {
		return x.Command
	}
	return nil
}

// CommandHandled is a journal record that indicates a command has been
// handled successfully.
type CommandHandled struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CommandId string                   `protobuf:"bytes,1,opt,name=command_id,json=commandId,proto3" json:"command_id,omitempty"`
	Events    []*envelopespec.Envelope `protobuf:"bytes,2,rep,name=events,proto3" json:"events,omitempty"`
}

func (x *CommandHandled) Reset() {
	*x = CommandHandled{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CommandHandled) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CommandHandled) ProtoMessage() {}

func (x *CommandHandled) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CommandHandled.ProtoReflect.Descriptor instead.
func (*CommandHandled) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDescGZIP(), []int{1}
}

func (x *CommandHandled) GetCommandId() string {
	if x != nil {
		return x.CommandId
	}
	return ""
}

func (x *CommandHandled) GetEvents() []*envelopespec.Envelope {
	if x != nil {
		return x.Events
	}
	return nil
}

// CommandHandlerFailed is a journal record that the handler returned an error
// when attempting to handle a command.
type CommandHandlerFailed struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CommandId string `protobuf:"bytes,1,opt,name=command_id,json=commandId,proto3" json:"command_id,omitempty"`
	Error     string `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
}

func (x *CommandHandlerFailed) Reset() {
	*x = CommandHandlerFailed{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CommandHandlerFailed) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CommandHandlerFailed) ProtoMessage() {}

func (x *CommandHandlerFailed) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CommandHandlerFailed.ProtoReflect.Descriptor instead.
func (*CommandHandlerFailed) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDescGZIP(), []int{2}
}

func (x *CommandHandlerFailed) GetCommandId() string {
	if x != nil {
		return x.CommandId
	}
	return ""
}

func (x *CommandHandlerFailed) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

// EventStreamSelected is a journal record that indicates that a specific event
// stream has been selected as the target for the events produced by the
// handler.
type EventStreamSelected struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StreamId   string `protobuf:"bytes,1,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	NextOffset uint64 `protobuf:"varint,2,opt,name=next_offset,json=nextOffset,proto3" json:"next_offset,omitempty"`
}

func (x *EventStreamSelected) Reset() {
	*x = EventStreamSelected{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventStreamSelected) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventStreamSelected) ProtoMessage() {}

func (x *EventStreamSelected) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventStreamSelected.ProtoReflect.Descriptor instead.
func (*EventStreamSelected) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDescGZIP(), []int{3}
}

func (x *EventStreamSelected) GetStreamId() string {
	if x != nil {
		return x.StreamId
	}
	return ""
}

func (x *EventStreamSelected) GetNextOffset() uint64 {
	if x != nil {
		return x.NextOffset
	}
	return 0
}

// EventAppendedToStream is a journal record that indicates that an event that
// was produced by the handler has been appended to an event stream.
type EventAppendedToStream struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StreamId string `protobuf:"bytes,1,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	Offset   uint64 `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"`
	EventId  string `protobuf:"bytes,3,opt,name=event_id,json=eventId,proto3" json:"event_id,omitempty"`
}

func (x *EventAppendedToStream) Reset() {
	*x = EventAppendedToStream{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventAppendedToStream) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventAppendedToStream) ProtoMessage() {}

func (x *EventAppendedToStream) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventAppendedToStream.ProtoReflect.Descriptor instead.
func (*EventAppendedToStream) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDescGZIP(), []int{4}
}

func (x *EventAppendedToStream) GetStreamId() string {
	if x != nil {
		return x.StreamId
	}
	return ""
}

func (x *EventAppendedToStream) GetOffset() uint64 {
	if x != nil {
		return x.Offset
	}
	return 0
}

func (x *EventAppendedToStream) GetEventId() string {
	if x != nil {
		return x.EventId
	}
	return ""
}

var File_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto protoreflect.FileDescriptor

var file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDesc = []byte{
	0x0a, 0x52, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x67,
	0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x67, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6a, 0x6f, 0x75,
	0x72, 0x6e, 0x61, 0x6c, 0x70, 0x62, 0x2f, 0x6a, 0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x69,
	0x6e, 0x74, 0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x6a, 0x6f, 0x75, 0x72, 0x6e,
	0x61, 0x6c, 0x2e, 0x76, 0x31, 0x1a, 0x3b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x6f, 0x70, 0x73, 0x70, 0x65, 0x63, 0x2f, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x73,
	0x70, 0x65, 0x63, 0x2f, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x50, 0x0a, 0x0f, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x45, 0x6e, 0x71,
	0x75, 0x65, 0x75, 0x65, 0x64, 0x12, 0x3d, 0x0a, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f,
	0x70, 0x65, 0x2e, 0x45, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x52, 0x07, 0x63, 0x6f, 0x6d,
	0x6d, 0x61, 0x6e, 0x64, 0x22, 0x6c, 0x0a, 0x0e, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x48,
	0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e,
	0x64, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x63, 0x6f, 0x6d, 0x6d,
	0x61, 0x6e, 0x64, 0x49, 0x64, 0x12, 0x3b, 0x0a, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x18,
	0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x69, 0x6e,
	0x74, 0x65, 0x72, 0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70,
	0x65, 0x2e, 0x45, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x52, 0x06, 0x65, 0x76, 0x65, 0x6e,
	0x74, 0x73, 0x22, 0x4b, 0x0a, 0x14, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x48, 0x61, 0x6e,
	0x64, 0x6c, 0x65, 0x72, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x63, 0x6f,
	0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09,
	0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x49, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x72, 0x72,
	0x6f, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x22,
	0x53, 0x0a, 0x13, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x53, 0x65,
	0x6c, 0x65, 0x63, 0x74, 0x65, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d,
	0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x73, 0x74, 0x72, 0x65, 0x61,
	0x6d, 0x49, 0x64, 0x12, 0x1f, 0x0a, 0x0b, 0x6e, 0x65, 0x78, 0x74, 0x5f, 0x6f, 0x66, 0x66, 0x73,
	0x65, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0a, 0x6e, 0x65, 0x78, 0x74, 0x4f, 0x66,
	0x66, 0x73, 0x65, 0x74, 0x22, 0x67, 0x0a, 0x15, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x41, 0x70, 0x70,
	0x65, 0x6e, 0x64, 0x65, 0x64, 0x54, 0x6f, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x12, 0x1b, 0x0a,
	0x09, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x08, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x49, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x6f, 0x66,
	0x66, 0x73, 0x65, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x6f, 0x66, 0x66, 0x73,
	0x65, 0x74, 0x12, 0x19, 0x0a, 0x08, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x42, 0x3d, 0x5a,
	0x3b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x67, 0x6d,
	0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2f, 0x69, 0x6e,
	0x74, 0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e,
	0x61, 0x6c, 0x2f, 0x6a, 0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDescOnce sync.Once
	file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDescData = file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDesc
)

func file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDescGZIP() []byte {
	file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDescOnce.Do(func() {
		file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDescData)
	})
	return file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDescData
}

var file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_goTypes = []interface{}{
	(*CommandEnqueued)(nil),       // 0: veracity.integration.journal.v1.CommandEnqueued
	(*CommandHandled)(nil),        // 1: veracity.integration.journal.v1.CommandHandled
	(*CommandHandlerFailed)(nil),  // 2: veracity.integration.journal.v1.CommandHandlerFailed
	(*EventStreamSelected)(nil),   // 3: veracity.integration.journal.v1.EventStreamSelected
	(*EventAppendedToStream)(nil), // 4: veracity.integration.journal.v1.EventAppendedToStream
	(*envelopespec.Envelope)(nil), // 5: dogma.interop.v1.envelope.Envelope
}
var file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_depIdxs = []int32{
	5, // 0: veracity.integration.journal.v1.CommandEnqueued.command:type_name -> dogma.interop.v1.envelope.Envelope
	5, // 1: veracity.integration.journal.v1.CommandHandled.events:type_name -> dogma.interop.v1.envelope.Envelope
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() {
	file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_init()
}
func file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_init() {
	if File_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
		file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CommandHandled); i {
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
		file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CommandHandlerFailed); i {
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
		file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventStreamSelected); i {
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
		file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventAppendedToStream); i {
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
			RawDescriptor: file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_goTypes,
		DependencyIndexes: file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_depIdxs,
		MessageInfos:      file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_msgTypes,
	}.Build()
	File_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto = out.File
	file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_rawDesc = nil
	file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_goTypes = nil
	file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_journal_proto_depIdxs = nil
}