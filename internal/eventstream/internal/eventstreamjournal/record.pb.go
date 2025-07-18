// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v6.31.1
// source: github.com/dogmatiq/veracity/internal/eventstream/internal/eventstreamjournal/record.proto

package eventstreamjournal

import (
	envelopepb "github.com/dogmatiq/enginekit/protobuf/envelopepb"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Record is a journal record that stores an operation performed on an event
// stream.
type Record struct {
	state              protoimpl.MessageState `protogen:"open.v1"`
	StreamOffsetBefore uint64                 `protobuf:"varint,1,opt,name=stream_offset_before,json=streamOffsetBefore,proto3" json:"stream_offset_before,omitempty"`
	StreamOffsetAfter  uint64                 `protobuf:"varint,2,opt,name=stream_offset_after,json=streamOffsetAfter,proto3" json:"stream_offset_after,omitempty"`
	// Types that are valid to be assigned to Operation:
	//
	//	*Record_EventsAppended
	Operation     isRecord_Operation `protobuf_oneof:"operation"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Record) Reset() {
	*x = Record{}
	mi := &file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Record) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Record) ProtoMessage() {}

func (x *Record) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_msgTypes[0]
	if x != nil {
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
	return file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_rawDescGZIP(), []int{0}
}

func (x *Record) GetStreamOffsetBefore() uint64 {
	if x != nil {
		return x.StreamOffsetBefore
	}
	return 0
}

func (x *Record) GetStreamOffsetAfter() uint64 {
	if x != nil {
		return x.StreamOffsetAfter
	}
	return 0
}

func (x *Record) GetOperation() isRecord_Operation {
	if x != nil {
		return x.Operation
	}
	return nil
}

func (x *Record) GetEventsAppended() *EventsAppended {
	if x != nil {
		if x, ok := x.Operation.(*Record_EventsAppended); ok {
			return x.EventsAppended
		}
	}
	return nil
}

type isRecord_Operation interface {
	isRecord_Operation()
}

type Record_EventsAppended struct {
	EventsAppended *EventsAppended `protobuf:"bytes,3,opt,name=events_appended,json=eventsAppended,proto3,oneof"`
}

func (*Record_EventsAppended) isRecord_Operation() {}

// AppendOperation is an operation that appends a set of events to a stream.
type EventsAppended struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Events        []*envelopepb.Envelope `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EventsAppended) Reset() {
	*x = EventsAppended{}
	mi := &file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EventsAppended) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventsAppended) ProtoMessage() {}

func (x *EventsAppended) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventsAppended.ProtoReflect.Descriptor instead.
func (*EventsAppended) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_rawDescGZIP(), []int{1}
}

func (x *EventsAppended) GetEvents() []*envelopepb.Envelope {
	if x != nil {
		return x.Events
	}
	return nil
}

var File_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto protoreflect.FileDescriptor

const file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_rawDesc = "" +
	"\n" +
	"Zgithub.com/dogmatiq/veracity/internal/eventstream/internal/eventstreamjournal/record.proto\x12\x1fveracity.eventstream.journal.v1\x1a@github.com/dogmatiq/enginekit/protobuf/envelopepb/envelope.proto\"\xd3\x01\n" +
	"\x06Record\x120\n" +
	"\x14stream_offset_before\x18\x01 \x01(\x04R\x12streamOffsetBefore\x12.\n" +
	"\x13stream_offset_after\x18\x02 \x01(\x04R\x11streamOffsetAfter\x12Z\n" +
	"\x0fevents_appended\x18\x03 \x01(\v2/.veracity.eventstream.journal.v1.EventsAppendedH\x00R\x0eeventsAppendedB\v\n" +
	"\toperation\"B\n" +
	"\x0eEventsAppended\x120\n" +
	"\x06events\x18\x01 \x03(\v2\x18.dogma.protobuf.EnvelopeR\x06eventsBOZMgithub.com/dogmatiq/veracity/internal/eventstream/internal/eventstreamjournalb\x06proto3"

var (
	file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_rawDescOnce sync.Once
	file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_rawDescData []byte
)

func file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_rawDescGZIP() []byte {
	file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_rawDescOnce.Do(func() {
		file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_rawDesc), len(file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_rawDesc)))
	})
	return file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_rawDescData
}

var file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_goTypes = []any{
	(*Record)(nil),              // 0: veracity.eventstream.journal.v1.Record
	(*EventsAppended)(nil),      // 1: veracity.eventstream.journal.v1.EventsAppended
	(*envelopepb.Envelope)(nil), // 2: dogma.protobuf.Envelope
}
var file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_depIdxs = []int32{
	1, // 0: veracity.eventstream.journal.v1.Record.events_appended:type_name -> veracity.eventstream.journal.v1.EventsAppended
	2, // 1: veracity.eventstream.journal.v1.EventsAppended.events:type_name -> dogma.protobuf.Envelope
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() {
	file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_init()
}
func file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_init() {
	if File_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto != nil {
		return
	}
	file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_msgTypes[0].OneofWrappers = []any{
		(*Record_EventsAppended)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_rawDesc), len(file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_goTypes,
		DependencyIndexes: file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_depIdxs,
		MessageInfos:      file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_msgTypes,
	}.Build()
	File_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto = out.File
	file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_goTypes = nil
	file_github_com_dogmatiq_veracity_internal_eventstream_internal_eventstreamjournal_record_proto_depIdxs = nil
}
