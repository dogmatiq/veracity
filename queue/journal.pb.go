// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.5
// source: github.com/dogmatiq/veracity/queue/journal.proto

package queue

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
	//	*JournalRecord_Enqueue
	//	*JournalRecord_Acquire
	//	*JournalRecord_Ack
	//	*JournalRecord_Nack
	OneOf isJournalRecord_OneOf `protobuf_oneof:"OneOf"`
}

func (x *JournalRecord) Reset() {
	*x = JournalRecord{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_queue_journal_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *JournalRecord) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JournalRecord) ProtoMessage() {}

func (x *JournalRecord) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_queue_journal_proto_msgTypes[0]
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
	return file_github_com_dogmatiq_veracity_queue_journal_proto_rawDescGZIP(), []int{0}
}

func (m *JournalRecord) GetOneOf() isJournalRecord_OneOf {
	if m != nil {
		return m.OneOf
	}
	return nil
}

func (x *JournalRecord) GetEnqueue() *envelopespec.Envelope {
	if x, ok := x.GetOneOf().(*JournalRecord_Enqueue); ok {
		return x.Enqueue
	}
	return nil
}

func (x *JournalRecord) GetAcquire() string {
	if x, ok := x.GetOneOf().(*JournalRecord_Acquire); ok {
		return x.Acquire
	}
	return ""
}

func (x *JournalRecord) GetAck() string {
	if x, ok := x.GetOneOf().(*JournalRecord_Ack); ok {
		return x.Ack
	}
	return ""
}

func (x *JournalRecord) GetNack() string {
	if x, ok := x.GetOneOf().(*JournalRecord_Nack); ok {
		return x.Nack
	}
	return ""
}

type isJournalRecord_OneOf interface {
	isJournalRecord_OneOf()
}

type JournalRecord_Enqueue struct {
	Enqueue *envelopespec.Envelope `protobuf:"bytes,1,opt,name=enqueue,proto3,oneof"`
}

type JournalRecord_Acquire struct {
	Acquire string `protobuf:"bytes,2,opt,name=acquire,proto3,oneof"`
}

type JournalRecord_Ack struct {
	Ack string `protobuf:"bytes,3,opt,name=ack,proto3,oneof"`
}

type JournalRecord_Nack struct {
	Nack string `protobuf:"bytes,4,opt,name=nack,proto3,oneof"`
}

func (*JournalRecord_Enqueue) isJournalRecord_OneOf() {}

func (*JournalRecord_Acquire) isJournalRecord_OneOf() {}

func (*JournalRecord_Ack) isJournalRecord_OneOf() {}

func (*JournalRecord_Nack) isJournalRecord_OneOf() {}

var File_github_com_dogmatiq_veracity_queue_journal_proto protoreflect.FileDescriptor

var file_github_com_dogmatiq_veracity_queue_journal_proto_rawDesc = []byte{
	0x0a, 0x30, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x67,
	0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2f, 0x71,
	0x75, 0x65, 0x75, 0x65, 0x2f, 0x6a, 0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x0e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x71, 0x75, 0x65,
	0x75, 0x65, 0x1a, 0x3b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64,
	0x6f, 0x67, 0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6f, 0x70, 0x73,
	0x70, 0x65, 0x63, 0x2f, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x73, 0x70, 0x65, 0x63,
	0x2f, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0x9f, 0x01, 0x0a, 0x0d, 0x4a, 0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x52, 0x65, 0x63, 0x6f, 0x72,
	0x64, 0x12, 0x3f, 0x0a, 0x07, 0x65, 0x6e, 0x71, 0x75, 0x65, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x2e, 0x45,
	0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x48, 0x00, 0x52, 0x07, 0x65, 0x6e, 0x71, 0x75, 0x65,
	0x75, 0x65, 0x12, 0x1a, 0x0a, 0x07, 0x61, 0x63, 0x71, 0x75, 0x69, 0x72, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x07, 0x61, 0x63, 0x71, 0x75, 0x69, 0x72, 0x65, 0x12, 0x12,
	0x0a, 0x03, 0x61, 0x63, 0x6b, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x03, 0x61,
	0x63, 0x6b, 0x12, 0x14, 0x0a, 0x04, 0x6e, 0x61, 0x63, 0x6b, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09,
	0x48, 0x00, 0x52, 0x04, 0x6e, 0x61, 0x63, 0x6b, 0x42, 0x07, 0x0a, 0x05, 0x4f, 0x6e, 0x65, 0x4f,
	0x66, 0x42, 0x24, 0x5a, 0x22, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x64, 0x6f, 0x67, 0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74,
	0x79, 0x2f, 0x71, 0x75, 0x65, 0x75, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_github_com_dogmatiq_veracity_queue_journal_proto_rawDescOnce sync.Once
	file_github_com_dogmatiq_veracity_queue_journal_proto_rawDescData = file_github_com_dogmatiq_veracity_queue_journal_proto_rawDesc
)

func file_github_com_dogmatiq_veracity_queue_journal_proto_rawDescGZIP() []byte {
	file_github_com_dogmatiq_veracity_queue_journal_proto_rawDescOnce.Do(func() {
		file_github_com_dogmatiq_veracity_queue_journal_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_dogmatiq_veracity_queue_journal_proto_rawDescData)
	})
	return file_github_com_dogmatiq_veracity_queue_journal_proto_rawDescData
}

var file_github_com_dogmatiq_veracity_queue_journal_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_dogmatiq_veracity_queue_journal_proto_goTypes = []interface{}{
	(*JournalRecord)(nil),         // 0: veracity.queue.JournalRecord
	(*envelopespec.Envelope)(nil), // 1: dogma.interop.v1.envelope.Envelope
}
var file_github_com_dogmatiq_veracity_queue_journal_proto_depIdxs = []int32{
	1, // 0: veracity.queue.JournalRecord.enqueue:type_name -> dogma.interop.v1.envelope.Envelope
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_github_com_dogmatiq_veracity_queue_journal_proto_init() }
func file_github_com_dogmatiq_veracity_queue_journal_proto_init() {
	if File_github_com_dogmatiq_veracity_queue_journal_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_dogmatiq_veracity_queue_journal_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
	}
	file_github_com_dogmatiq_veracity_queue_journal_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*JournalRecord_Enqueue)(nil),
		(*JournalRecord_Acquire)(nil),
		(*JournalRecord_Ack)(nil),
		(*JournalRecord_Nack)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_dogmatiq_veracity_queue_journal_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_dogmatiq_veracity_queue_journal_proto_goTypes,
		DependencyIndexes: file_github_com_dogmatiq_veracity_queue_journal_proto_depIdxs,
		MessageInfos:      file_github_com_dogmatiq_veracity_queue_journal_proto_msgTypes,
	}.Build()
	File_github_com_dogmatiq_veracity_queue_journal_proto = out.File
	file_github_com_dogmatiq_veracity_queue_journal_proto_rawDesc = nil
	file_github_com_dogmatiq_veracity_queue_journal_proto_goTypes = nil
	file_github_com_dogmatiq_veracity_queue_journal_proto_depIdxs = nil
}
