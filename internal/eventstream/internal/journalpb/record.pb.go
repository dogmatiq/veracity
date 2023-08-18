// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.0
// source: github.com/dogmatiq/veracity/internal/eventstream/internal/journalpb/record.proto

package journalpb

import (
	envelopepb "github.com/dogmatiq/enginekit/protobuf/envelopepb"
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

// Record is a journal record that stores an operation performed on an event
// stream.
type Record struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StreamOffsetBefore uint64 `protobuf:"varint,1,opt,name=stream_offset_before,json=streamOffsetBefore,proto3" json:"stream_offset_before,omitempty"`
	StreamOffsetAfter  uint64 `protobuf:"varint,2,opt,name=stream_offset_after,json=streamOffsetAfter,proto3" json:"stream_offset_after,omitempty"`
	// Types that are assignable to Operation:
	//
	//	*Record_Append
	Operation isRecord_Operation `protobuf_oneof:"operation"`
}

func (x *Record) Reset() {
	*x = Record{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Record) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Record) ProtoMessage() {}

func (x *Record) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_msgTypes[0]
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
	return file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_rawDescGZIP(), []int{0}
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

func (m *Record) GetOperation() isRecord_Operation {
	if m != nil {
		return m.Operation
	}
	return nil
}

func (x *Record) GetAppend() *AppendOperation {
	if x, ok := x.GetOperation().(*Record_Append); ok {
		return x.Append
	}
	return nil
}

type isRecord_Operation interface {
	isRecord_Operation()
}

type Record_Append struct {
	Append *AppendOperation `protobuf:"bytes,3,opt,name=append,proto3,oneof"`
}

func (*Record_Append) isRecord_Operation() {}

// AppendOperation is an operation that indicates a set of events have been
// appended to the stream.
type AppendOperation struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Events []*envelopepb.Envelope `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty"`
}

func (x *AppendOperation) Reset() {
	*x = AppendOperation{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AppendOperation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AppendOperation) ProtoMessage() {}

func (x *AppendOperation) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AppendOperation.ProtoReflect.Descriptor instead.
func (*AppendOperation) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_rawDescGZIP(), []int{1}
}

func (x *AppendOperation) GetEvents() []*envelopepb.Envelope {
	if x != nil {
		return x.Events
	}
	return nil
}

var File_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto protoreflect.FileDescriptor

var file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_rawDesc = []byte{
	0x0a, 0x51, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x67,
	0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x74, 0x72,
	0x65, 0x61, 0x6d, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6a, 0x6f, 0x75,
	0x72, 0x6e, 0x61, 0x6c, 0x70, 0x62, 0x2f, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x1f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x65, 0x76,
	0x65, 0x6e, 0x74, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x2e, 0x6a, 0x6f, 0x75, 0x72, 0x6e, 0x61,
	0x6c, 0x2e, 0x76, 0x31, 0x1a, 0x40, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x65, 0x6e, 0x67, 0x69, 0x6e, 0x65,
	0x6b, 0x69, 0x74, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6e, 0x76,
	0x65, 0x6c, 0x6f, 0x70, 0x65, 0x70, 0x62, 0x2f, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xc3, 0x01, 0x0a, 0x06, 0x52, 0x65, 0x63, 0x6f, 0x72,
	0x64, 0x12, 0x30, 0x0a, 0x14, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x5f, 0x6f, 0x66, 0x66, 0x73,
	0x65, 0x74, 0x5f, 0x62, 0x65, 0x66, 0x6f, 0x72, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x12, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x4f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x42, 0x65, 0x66,
	0x6f, 0x72, 0x65, 0x12, 0x2e, 0x0a, 0x13, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x5f, 0x6f, 0x66,
	0x66, 0x73, 0x65, 0x74, 0x5f, 0x61, 0x66, 0x74, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x11, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x4f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x41, 0x66,
	0x74, 0x65, 0x72, 0x12, 0x4a, 0x0a, 0x06, 0x61, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x30, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x65,
	0x76, 0x65, 0x6e, 0x74, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x2e, 0x6a, 0x6f, 0x75, 0x72, 0x6e,
	0x61, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x4f, 0x70, 0x65, 0x72,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x48, 0x00, 0x52, 0x06, 0x61, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x42,
	0x0b, 0x0a, 0x09, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x43, 0x0a, 0x0f,
	0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12,
	0x30, 0x0a, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x18, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x45, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x52, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74,
	0x73, 0x42, 0x46, 0x5a, 0x44, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x64, 0x6f, 0x67, 0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74,
	0x79, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x65, 0x76, 0x65, 0x6e, 0x74,
	0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f,
	0x6a, 0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_rawDescOnce sync.Once
	file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_rawDescData = file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_rawDesc
)

func file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_rawDescGZIP() []byte {
	file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_rawDescOnce.Do(func() {
		file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_rawDescData)
	})
	return file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_rawDescData
}

var file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_goTypes = []interface{}{
	(*Record)(nil),              // 0: veracity.eventstream.journal.v1.Record
	(*AppendOperation)(nil),     // 1: veracity.eventstream.journal.v1.AppendOperation
	(*envelopepb.Envelope)(nil), // 2: dogma.protobuf.Envelope
}
var file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_depIdxs = []int32{
	1, // 0: veracity.eventstream.journal.v1.Record.append:type_name -> veracity.eventstream.journal.v1.AppendOperation
	2, // 1: veracity.eventstream.journal.v1.AppendOperation.events:type_name -> dogma.protobuf.Envelope
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() {
	file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_init()
}
func file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_init() {
	if File_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
		file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AppendOperation); i {
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
	file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*Record_Append)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_goTypes,
		DependencyIndexes: file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_depIdxs,
		MessageInfos:      file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_msgTypes,
	}.Build()
	File_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto = out.File
	file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_rawDesc = nil
	file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_goTypes = nil
	file_github_com_dogmatiq_veracity_internal_eventstream_internal_journalpb_record_proto_depIdxs = nil
}
