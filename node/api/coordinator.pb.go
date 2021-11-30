// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.19.1
// source: github.com/dogmatiq/veracity/node/api/coordinator.proto

package api

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

type EnqueueCommandRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Command *envelopespec.Envelope `protobuf:"bytes,1,opt,name=command,proto3" json:"command,omitempty"`
}

func (x *EnqueueCommandRequest) Reset() {
	*x = EnqueueCommandRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_node_api_coordinator_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EnqueueCommandRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnqueueCommandRequest) ProtoMessage() {}

func (x *EnqueueCommandRequest) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_node_api_coordinator_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnqueueCommandRequest.ProtoReflect.Descriptor instead.
func (*EnqueueCommandRequest) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_node_api_coordinator_proto_rawDescGZIP(), []int{0}
}

func (x *EnqueueCommandRequest) GetCommand() *envelopespec.Envelope {
	if x != nil {
		return x.Command
	}
	return nil
}

type EnqueueCommandResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *EnqueueCommandResponse) Reset() {
	*x = EnqueueCommandResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_node_api_coordinator_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EnqueueCommandResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnqueueCommandResponse) ProtoMessage() {}

func (x *EnqueueCommandResponse) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_node_api_coordinator_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnqueueCommandResponse.ProtoReflect.Descriptor instead.
func (*EnqueueCommandResponse) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_node_api_coordinator_proto_rawDescGZIP(), []int{1}
}

var File_github_com_dogmatiq_veracity_node_api_coordinator_proto protoreflect.FileDescriptor

var file_github_com_dogmatiq_veracity_node_api_coordinator_proto_rawDesc = []byte{
	0x0a, 0x37, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x67,
	0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2f, 0x6e,
	0x6f, 0x64, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x63, 0x6f, 0x6f, 0x72, 0x64, 0x69, 0x6e, 0x61,
	0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x10, 0x76, 0x65, 0x72, 0x61, 0x63,
	0x69, 0x74, 0x79, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x76, 0x31, 0x1a, 0x3b, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x74, 0x69, 0x71,
	0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6f, 0x70, 0x73, 0x70, 0x65, 0x63, 0x2f, 0x65, 0x6e, 0x76,
	0x65, 0x6c, 0x6f, 0x70, 0x65, 0x73, 0x70, 0x65, 0x63, 0x2f, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f,
	0x70, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x56, 0x0a, 0x15, 0x45, 0x6e, 0x71, 0x75,
	0x65, 0x75, 0x65, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x3d, 0x0a, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x6f, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x2e, 0x45,
	0x6e, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x52, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64,
	0x22, 0x18, 0x0a, 0x16, 0x45, 0x6e, 0x71, 0x75, 0x65, 0x75, 0x65, 0x43, 0x6f, 0x6d, 0x6d, 0x61,
	0x6e, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x32, 0x75, 0x0a, 0x0e, 0x43, 0x6f,
	0x6f, 0x72, 0x64, 0x69, 0x6e, 0x61, 0x74, 0x6f, 0x72, 0x41, 0x50, 0x49, 0x12, 0x63, 0x0a, 0x0e,
	0x45, 0x6e, 0x71, 0x75, 0x65, 0x75, 0x65, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x12, 0x27,
	0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x76,
	0x31, 0x2e, 0x45, 0x6e, 0x71, 0x75, 0x65, 0x75, 0x65, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x28, 0x2e, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69,
	0x74, 0x79, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6e, 0x71, 0x75, 0x65,
	0x75, 0x65, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x42, 0x27, 0x5a, 0x25, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x64, 0x6f, 0x67, 0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74,
	0x79, 0x2f, 0x6e, 0x6f, 0x64, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_github_com_dogmatiq_veracity_node_api_coordinator_proto_rawDescOnce sync.Once
	file_github_com_dogmatiq_veracity_node_api_coordinator_proto_rawDescData = file_github_com_dogmatiq_veracity_node_api_coordinator_proto_rawDesc
)

func file_github_com_dogmatiq_veracity_node_api_coordinator_proto_rawDescGZIP() []byte {
	file_github_com_dogmatiq_veracity_node_api_coordinator_proto_rawDescOnce.Do(func() {
		file_github_com_dogmatiq_veracity_node_api_coordinator_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_dogmatiq_veracity_node_api_coordinator_proto_rawDescData)
	})
	return file_github_com_dogmatiq_veracity_node_api_coordinator_proto_rawDescData
}

var file_github_com_dogmatiq_veracity_node_api_coordinator_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_github_com_dogmatiq_veracity_node_api_coordinator_proto_goTypes = []interface{}{
	(*EnqueueCommandRequest)(nil),  // 0: veracity.node.v1.EnqueueCommandRequest
	(*EnqueueCommandResponse)(nil), // 1: veracity.node.v1.EnqueueCommandResponse
	(*envelopespec.Envelope)(nil),  // 2: dogma.interop.v1.envelope.Envelope
}
var file_github_com_dogmatiq_veracity_node_api_coordinator_proto_depIdxs = []int32{
	2, // 0: veracity.node.v1.EnqueueCommandRequest.command:type_name -> dogma.interop.v1.envelope.Envelope
	0, // 1: veracity.node.v1.CoordinatorAPI.EnqueueCommand:input_type -> veracity.node.v1.EnqueueCommandRequest
	1, // 2: veracity.node.v1.CoordinatorAPI.EnqueueCommand:output_type -> veracity.node.v1.EnqueueCommandResponse
	2, // [2:3] is the sub-list for method output_type
	1, // [1:2] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_github_com_dogmatiq_veracity_node_api_coordinator_proto_init() }
func file_github_com_dogmatiq_veracity_node_api_coordinator_proto_init() {
	if File_github_com_dogmatiq_veracity_node_api_coordinator_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_dogmatiq_veracity_node_api_coordinator_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EnqueueCommandRequest); i {
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
		file_github_com_dogmatiq_veracity_node_api_coordinator_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EnqueueCommandResponse); i {
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
			RawDescriptor: file_github_com_dogmatiq_veracity_node_api_coordinator_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_github_com_dogmatiq_veracity_node_api_coordinator_proto_goTypes,
		DependencyIndexes: file_github_com_dogmatiq_veracity_node_api_coordinator_proto_depIdxs,
		MessageInfos:      file_github_com_dogmatiq_veracity_node_api_coordinator_proto_msgTypes,
	}.Build()
	File_github_com_dogmatiq_veracity_node_api_coordinator_proto = out.File
	file_github_com_dogmatiq_veracity_node_api_coordinator_proto_rawDesc = nil
	file_github_com_dogmatiq_veracity_node_api_coordinator_proto_goTypes = nil
	file_github_com_dogmatiq_veracity_node_api_coordinator_proto_depIdxs = nil
}
