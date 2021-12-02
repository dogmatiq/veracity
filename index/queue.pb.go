// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.19.1
// source: github.com/dogmatiq/veracity/index/queue.proto

package index

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

type Queue struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	MessageIds []string `protobuf:"bytes,1,rep,name=message_ids,json=messageIds,proto3" json:"message_ids,omitempty"`
}

func (x *Queue) Reset() {
	*x = Queue{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_index_queue_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Queue) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Queue) ProtoMessage() {}

func (x *Queue) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_index_queue_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Queue.ProtoReflect.Descriptor instead.
func (*Queue) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_index_queue_proto_rawDescGZIP(), []int{0}
}

func (x *Queue) GetMessageIds() []string {
	if x != nil {
		return x.MessageIds
	}
	return nil
}

var File_github_com_dogmatiq_veracity_index_queue_proto protoreflect.FileDescriptor

var file_github_com_dogmatiq_veracity_index_queue_proto_rawDesc = []byte{
	0x0a, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x67,
	0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2f, 0x69,
	0x6e, 0x64, 0x65, 0x78, 0x2f, 0x71, 0x75, 0x65, 0x75, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x11, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x69, 0x6e, 0x64, 0x65, 0x78,
	0x2e, 0x76, 0x31, 0x22, 0x28, 0x0a, 0x05, 0x51, 0x75, 0x65, 0x75, 0x65, 0x12, 0x1f, 0x0a, 0x0b,
	0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x5f, 0x69, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x09, 0x52, 0x0a, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x49, 0x64, 0x73, 0x42, 0x24, 0x5a,
	0x22, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x67, 0x6d,
	0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2f, 0x69, 0x6e,
	0x64, 0x65, 0x78, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_github_com_dogmatiq_veracity_index_queue_proto_rawDescOnce sync.Once
	file_github_com_dogmatiq_veracity_index_queue_proto_rawDescData = file_github_com_dogmatiq_veracity_index_queue_proto_rawDesc
)

func file_github_com_dogmatiq_veracity_index_queue_proto_rawDescGZIP() []byte {
	file_github_com_dogmatiq_veracity_index_queue_proto_rawDescOnce.Do(func() {
		file_github_com_dogmatiq_veracity_index_queue_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_dogmatiq_veracity_index_queue_proto_rawDescData)
	})
	return file_github_com_dogmatiq_veracity_index_queue_proto_rawDescData
}

var file_github_com_dogmatiq_veracity_index_queue_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_dogmatiq_veracity_index_queue_proto_goTypes = []interface{}{
	(*Queue)(nil), // 0: veracity.index.v1.Queue
}
var file_github_com_dogmatiq_veracity_index_queue_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_github_com_dogmatiq_veracity_index_queue_proto_init() }
func file_github_com_dogmatiq_veracity_index_queue_proto_init() {
	if File_github_com_dogmatiq_veracity_index_queue_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_dogmatiq_veracity_index_queue_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Queue); i {
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
			RawDescriptor: file_github_com_dogmatiq_veracity_index_queue_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_dogmatiq_veracity_index_queue_proto_goTypes,
		DependencyIndexes: file_github_com_dogmatiq_veracity_index_queue_proto_depIdxs,
		MessageInfos:      file_github_com_dogmatiq_veracity_index_queue_proto_msgTypes,
	}.Build()
	File_github_com_dogmatiq_veracity_index_queue_proto = out.File
	file_github_com_dogmatiq_veracity_index_queue_proto_rawDesc = nil
	file_github_com_dogmatiq_veracity_index_queue_proto_goTypes = nil
	file_github_com_dogmatiq_veracity_index_queue_proto_depIdxs = nil
}
