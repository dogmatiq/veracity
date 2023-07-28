// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.2
// source: github.com/dogmatiq/veracity/internal/integration/internal/journalpb/record.proto

package journalpb

import (
	envelopepb "github.com/dogmatiq/enginekit/protobuf/envelopepb"
	uuidpb "github.com/dogmatiq/enginekit/protobuf/uuidpb"
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

// Record is a container for a journal record.
type Record struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to OneOf:
	//
	//	*Record_CommandEnqueued
	//	*Record_CommandHandled
	//	*Record_CommandHandlerFailed
	//	*Record_EventStreamSelected
	//	*Record_EventsAppendedToStream
	OneOf isRecord_OneOf `protobuf_oneof:"one_of"`
}

func (x *Record) Reset() {
	*x = Record{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Record) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Record) ProtoMessage() {}

func (x *Record) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Record.ProtoReflect.Descriptor instead.
func (*Record) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDescGZIP(), []int{0}
}

func (m *Record) GetOneOf() isRecord_OneOf {
	if m != nil {
		return m.OneOf
	}
	return nil
}

func (x *Record) GetCommandEnqueued() *CommandEnqueued {
	if x, ok := x.GetOneOf().(*Record_CommandEnqueued); ok {
		return x.CommandEnqueued
	}
	return nil
}

func (x *Record) GetCommandHandled() *CommandHandled {
	if x, ok := x.GetOneOf().(*Record_CommandHandled); ok {
		return x.CommandHandled
	}
	return nil
}

func (x *Record) GetCommandHandlerFailed() *CommandHandlerFailed {
	if x, ok := x.GetOneOf().(*Record_CommandHandlerFailed); ok {
		return x.CommandHandlerFailed
	}
	return nil
}

func (x *Record) GetEventStreamSelected() *EventStreamSelected {
	if x, ok := x.GetOneOf().(*Record_EventStreamSelected); ok {
		return x.EventStreamSelected
	}
	return nil
}

func (x *Record) GetEventsAppendedToStream() *EventsAppendedToStream {
	if x, ok := x.GetOneOf().(*Record_EventsAppendedToStream); ok {
		return x.EventsAppendedToStream
	}
	return nil
}

type isRecord_OneOf interface {
	isRecord_OneOf()
}

type Record_CommandEnqueued struct {
	CommandEnqueued *CommandEnqueued `protobuf:"bytes,1,opt,name=command_enqueued,json=commandEnqueued,proto3,oneof"`
}

type Record_CommandHandled struct {
	CommandHandled *CommandHandled `protobuf:"bytes,2,opt,name=command_handled,json=commandHandled,proto3,oneof"`
}

type Record_CommandHandlerFailed struct {
	CommandHandlerFailed *CommandHandlerFailed `protobuf:"bytes,3,opt,name=command_handler_failed,json=commandHandlerFailed,proto3,oneof"`
}

type Record_EventStreamSelected struct {
	EventStreamSelected *EventStreamSelected `protobuf:"bytes,4,opt,name=event_stream_selected,json=eventStreamSelected,proto3,oneof"`
}

type Record_EventsAppendedToStream struct {
	EventsAppendedToStream *EventsAppendedToStream `protobuf:"bytes,5,opt,name=events_appended_to_stream,json=eventsAppendedToStream,proto3,oneof"`
}

func (*Record_CommandEnqueued) isRecord_OneOf() {}

func (*Record_CommandHandled) isRecord_OneOf() {}

func (*Record_CommandHandlerFailed) isRecord_OneOf() {}

func (*Record_EventStreamSelected) isRecord_OneOf() {}

func (*Record_EventsAppendedToStream) isRecord_OneOf() {}

// CommandEnqueued is a journal record that indicates a command has been
// enqueued for handling.
type CommandEnqueued struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Command is the envelope containing the command to be handled.
	Command *envelopepb.Envelope `protobuf:"bytes,1,opt,name=command,proto3" json:"command,omitempty"`
}

func (x *CommandEnqueued) Reset() {
	*x = CommandEnqueued{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CommandEnqueued) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CommandEnqueued) ProtoMessage() {}

func (x *CommandEnqueued) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[1]
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
	return file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDescGZIP(), []int{1}
}

func (x *CommandEnqueued) GetCommand() *envelopepb.Envelope {
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

	// CommandId is the ID of the command that was handled.
	CommandId *uuidpb.UUID `protobuf:"bytes,1,opt,name=command_id,json=commandId,proto3" json:"command_id,omitempty"`
	// Events is the list of events produced while handling the command, in
	// chronological order.
	Events []*envelopepb.Envelope `protobuf:"bytes,2,rep,name=events,proto3" json:"events,omitempty"`
}

func (x *CommandHandled) Reset() {
	*x = CommandHandled{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CommandHandled) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CommandHandled) ProtoMessage() {}

func (x *CommandHandled) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[2]
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
	return file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDescGZIP(), []int{2}
}

func (x *CommandHandled) GetCommandId() *uuidpb.UUID {
	if x != nil {
		return x.CommandId
	}
	return nil
}

func (x *CommandHandled) GetEvents() []*envelopepb.Envelope {
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

	// CommandId is the ID of the command that was being handled.
	CommandId *uuidpb.UUID `protobuf:"bytes,1,opt,name=command_id,json=commandId,proto3" json:"command_id,omitempty"`
	// Error is the message of the error that the handler returned.
	Error string `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
}

func (x *CommandHandlerFailed) Reset() {
	*x = CommandHandlerFailed{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CommandHandlerFailed) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CommandHandlerFailed) ProtoMessage() {}

func (x *CommandHandlerFailed) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[3]
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
	return file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDescGZIP(), []int{3}
}

func (x *CommandHandlerFailed) GetCommandId() *uuidpb.UUID {
	if x != nil {
		return x.CommandId
	}
	return nil
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

	// StreamId is the ID of the stream to which the events will be appended.
	StreamId *uuidpb.UUID `protobuf:"bytes,1,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	// Offset is the next offset of the stream, at the time it was selected.
	Offset uint64 `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"`
}

func (x *EventStreamSelected) Reset() {
	*x = EventStreamSelected{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventStreamSelected) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventStreamSelected) ProtoMessage() {}

func (x *EventStreamSelected) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[4]
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
	return file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDescGZIP(), []int{4}
}

func (x *EventStreamSelected) GetStreamId() *uuidpb.UUID {
	if x != nil {
		return x.StreamId
	}
	return nil
}

func (x *EventStreamSelected) GetOffset() uint64 {
	if x != nil {
		return x.Offset
	}
	return 0
}

// EventsAppendedToStream is a journal record that indicates that the events
// produced by a specific command have been appended to an event stream.
type EventsAppendedToStream struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// CommandId is the ID of the command that produced the events.
	CommandId *uuidpb.UUID `protobuf:"bytes,1,opt,name=command_id,json=commandId,proto3" json:"command_id,omitempty"`
	// StreamId is the ID of the stream to which the events were appended.
	StreamId *uuidpb.UUID `protobuf:"bytes,2,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	// Offset is the offset of the first event.
	Offset uint64 `protobuf:"varint,3,opt,name=offset,proto3" json:"offset,omitempty"`
}

func (x *EventsAppendedToStream) Reset() {
	*x = EventsAppendedToStream{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventsAppendedToStream) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventsAppendedToStream) ProtoMessage() {}

func (x *EventsAppendedToStream) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventsAppendedToStream.ProtoReflect.Descriptor instead.
func (*EventsAppendedToStream) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDescGZIP(), []int{5}
}

func (x *EventsAppendedToStream) GetCommandId() *uuidpb.UUID {
	if x != nil {
		return x.CommandId
	}
	return nil
}

func (x *EventsAppendedToStream) GetStreamId() *uuidpb.UUID {
	if x != nil {
		return x.StreamId
	}
	return nil
}

func (x *EventsAppendedToStream) GetOffset() uint64 {
	if x != nil {
		return x.Offset
	}
	return 0
}

var File_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto protoreflect.FileDescriptor

var file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDesc = []byte{
	0x0a, 0x51, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x67,
	0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x67, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6a, 0x6f, 0x75,
	0x72, 0x6e, 0x61, 0x6c, 0x70, 0x62, 0x2f, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x1f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x69, 0x6e,
	0x74, 0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x6a, 0x6f, 0x75, 0x72, 0x6e, 0x61,
	0x6c, 0x2e, 0x76, 0x31, 0x1a, 0x38, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x65, 0x6e, 0x67, 0x69, 0x6e, 0x65,
	0x6b, 0x69, 0x74, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x75, 0x75, 0x69,
	0x64, 0x70, 0x62, 0x2f, 0x75, 0x75, 0x69, 0x64, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x40,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x67, 0x6d, 0x61,
	0x74, 0x69, 0x71, 0x2f, 0x65, 0x6e, 0x67, 0x69, 0x6e, 0x65, 0x6b, 0x69, 0x74, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x70,
	0x62, 0x2f, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0x9e, 0x04, 0x0a, 0x06, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x12, 0x5d, 0x0a, 0x10, 0x63,
	0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x5f, 0x65, 0x6e, 0x71, 0x75, 0x65, 0x75, 0x65, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x30, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79,
	0x2e, 0x69, 0x6e, 0x74, 0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x6a, 0x6f, 0x75,
	0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x45,
	0x6e, 0x71, 0x75, 0x65, 0x75, 0x65, 0x64, 0x48, 0x00, 0x52, 0x0f, 0x63, 0x6f, 0x6d, 0x6d, 0x61,
	0x6e, 0x64, 0x45, 0x6e, 0x71, 0x75, 0x65, 0x75, 0x65, 0x64, 0x12, 0x5a, 0x0a, 0x0f, 0x63, 0x6f,
	0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x5f, 0x68, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x2f, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x69,
	0x6e, 0x74, 0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x6a, 0x6f, 0x75, 0x72, 0x6e,
	0x61, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x48, 0x61, 0x6e,
	0x64, 0x6c, 0x65, 0x64, 0x48, 0x00, 0x52, 0x0e, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x48,
	0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64, 0x12, 0x6d, 0x0a, 0x16, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e,
	0x64, 0x5f, 0x68, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x72, 0x5f, 0x66, 0x61, 0x69, 0x6c, 0x65, 0x64,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x35, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74,
	0x79, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x6a, 0x6f,
	0x75, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64,
	0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x72, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64, 0x48, 0x00, 0x52,
	0x14, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x72, 0x46,
	0x61, 0x69, 0x6c, 0x65, 0x64, 0x12, 0x6a, 0x0a, 0x15, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x5f, 0x73,
	0x74, 0x72, 0x65, 0x61, 0x6d, 0x5f, 0x73, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x65, 0x64, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x34, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e,
	0x69, 0x6e, 0x74, 0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x6a, 0x6f, 0x75, 0x72,
	0x6e, 0x61, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x72, 0x65,
	0x61, 0x6d, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x65, 0x64, 0x48, 0x00, 0x52, 0x13, 0x65, 0x76,
	0x65, 0x6e, 0x74, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x65,
	0x64, 0x12, 0x74, 0x0a, 0x19, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x5f, 0x61, 0x70, 0x70, 0x65,
	0x6e, 0x64, 0x65, 0x64, 0x5f, 0x74, 0x6f, 0x5f, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x37, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e,
	0x69, 0x6e, 0x74, 0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x6a, 0x6f, 0x75, 0x72,
	0x6e, 0x61, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x41, 0x70, 0x70,
	0x65, 0x6e, 0x64, 0x65, 0x64, 0x54, 0x6f, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x48, 0x00, 0x52,
	0x16, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x54,
	0x6f, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x42, 0x08, 0x0a, 0x06, 0x6f, 0x6e, 0x65, 0x5f, 0x6f,
	0x66, 0x22, 0x45, 0x0a, 0x0f, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x45, 0x6e, 0x71, 0x75,
	0x65, 0x75, 0x65, 0x64, 0x12, 0x32, 0x0a, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x52,
	0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x22, 0x77, 0x0a, 0x0e, 0x43, 0x6f, 0x6d, 0x6d,
	0x61, 0x6e, 0x64, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x64, 0x12, 0x33, 0x0a, 0x0a, 0x63, 0x6f,
	0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14,
	0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x55, 0x55, 0x49, 0x44, 0x52, 0x09, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x49, 0x64, 0x12,
	0x30, 0x0a, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x18, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x45, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x52, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74,
	0x73, 0x22, 0x61, 0x0a, 0x14, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x48, 0x61, 0x6e, 0x64,
	0x6c, 0x65, 0x72, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64, 0x12, 0x33, 0x0a, 0x0a, 0x63, 0x6f, 0x6d,
	0x6d, 0x61, 0x6e, 0x64, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e,
	0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55,
	0x55, 0x49, 0x44, 0x52, 0x09, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x49, 0x64, 0x12, 0x14,
	0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65,
	0x72, 0x72, 0x6f, 0x72, 0x22, 0x60, 0x0a, 0x13, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x72,
	0x65, 0x61, 0x6d, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x65, 0x64, 0x12, 0x31, 0x0a, 0x09, 0x73,
	0x74, 0x72, 0x65, 0x61, 0x6d, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14,
	0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x55, 0x55, 0x49, 0x44, 0x52, 0x08, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x49, 0x64, 0x12, 0x16,
	0x0a, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06,
	0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x22, 0x98, 0x01, 0x0a, 0x16, 0x45, 0x76, 0x65, 0x6e, 0x74,
	0x73, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x54, 0x6f, 0x53, 0x74, 0x72, 0x65, 0x61,
	0x6d, 0x12, 0x33, 0x0a, 0x0a, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x5f, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x55, 0x49, 0x44, 0x52, 0x09, 0x63, 0x6f, 0x6d,
	0x6d, 0x61, 0x6e, 0x64, 0x49, 0x64, 0x12, 0x31, 0x0a, 0x09, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d,
	0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x64, 0x6f, 0x67, 0x6d,
	0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x55, 0x49, 0x44, 0x52,
	0x08, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x49, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x6f, 0x66, 0x66,
	0x73, 0x65, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65,
	0x74, 0x42, 0x46, 0x5a, 0x44, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x64, 0x6f, 0x67, 0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74,
	0x79, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x67,
	0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f,
	0x6a, 0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDescOnce sync.Once
	file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDescData = file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDesc
)

func file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDescGZIP() []byte {
	file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDescOnce.Do(func() {
		file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDescData)
	})
	return file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDescData
}

var file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_goTypes = []interface{}{
	(*Record)(nil),                 // 0: veracity.integration.journal.v1.Record
	(*CommandEnqueued)(nil),        // 1: veracity.integration.journal.v1.CommandEnqueued
	(*CommandHandled)(nil),         // 2: veracity.integration.journal.v1.CommandHandled
	(*CommandHandlerFailed)(nil),   // 3: veracity.integration.journal.v1.CommandHandlerFailed
	(*EventStreamSelected)(nil),    // 4: veracity.integration.journal.v1.EventStreamSelected
	(*EventsAppendedToStream)(nil), // 5: veracity.integration.journal.v1.EventsAppendedToStream
	(*envelopepb.Envelope)(nil),    // 6: dogma.protobuf.Envelope
	(*uuidpb.UUID)(nil),            // 7: dogma.protobuf.UUID
}
var file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_depIdxs = []int32{
	1,  // 0: veracity.integration.journal.v1.Record.command_enqueued:type_name -> veracity.integration.journal.v1.CommandEnqueued
	2,  // 1: veracity.integration.journal.v1.Record.command_handled:type_name -> veracity.integration.journal.v1.CommandHandled
	3,  // 2: veracity.integration.journal.v1.Record.command_handler_failed:type_name -> veracity.integration.journal.v1.CommandHandlerFailed
	4,  // 3: veracity.integration.journal.v1.Record.event_stream_selected:type_name -> veracity.integration.journal.v1.EventStreamSelected
	5,  // 4: veracity.integration.journal.v1.Record.events_appended_to_stream:type_name -> veracity.integration.journal.v1.EventsAppendedToStream
	6,  // 5: veracity.integration.journal.v1.CommandEnqueued.command:type_name -> dogma.protobuf.Envelope
	7,  // 6: veracity.integration.journal.v1.CommandHandled.command_id:type_name -> dogma.protobuf.UUID
	6,  // 7: veracity.integration.journal.v1.CommandHandled.events:type_name -> dogma.protobuf.Envelope
	7,  // 8: veracity.integration.journal.v1.CommandHandlerFailed.command_id:type_name -> dogma.protobuf.UUID
	7,  // 9: veracity.integration.journal.v1.EventStreamSelected.stream_id:type_name -> dogma.protobuf.UUID
	7,  // 10: veracity.integration.journal.v1.EventsAppendedToStream.command_id:type_name -> dogma.protobuf.UUID
	7,  // 11: veracity.integration.journal.v1.EventsAppendedToStream.stream_id:type_name -> dogma.protobuf.UUID
	12, // [12:12] is the sub-list for method output_type
	12, // [12:12] is the sub-list for method input_type
	12, // [12:12] is the sub-list for extension type_name
	12, // [12:12] is the sub-list for extension extendee
	0,  // [0:12] is the sub-list for field type_name
}

func init() {
	file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_init()
}
func file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_init() {
	if File_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Record); i {
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
		file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
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
		file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
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
		file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
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
		file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
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
		file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventsAppendedToStream); i {
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
	file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*Record_CommandEnqueued)(nil),
		(*Record_CommandHandled)(nil),
		(*Record_CommandHandlerFailed)(nil),
		(*Record_EventStreamSelected)(nil),
		(*Record_EventsAppendedToStream)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_goTypes,
		DependencyIndexes: file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_depIdxs,
		MessageInfos:      file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_msgTypes,
	}.Build()
	File_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto = out.File
	file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_rawDesc = nil
	file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_goTypes = nil
	file_github_com_dogmatiq_veracity_internal_integration_internal_journalpb_record_proto_depIdxs = nil
}
