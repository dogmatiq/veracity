// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        v5.26.1
// source: github.com/dogmatiq/veracity/internal/cluster/internal/registrypb/registration.proto

package registrypb

import (
	uuidpb "github.com/dogmatiq/enginekit/protobuf/uuidpb"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Node struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Id is a unique identifier for the node.
	Id *uuidpb.UUID `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// Addresses is a list of network addresses at which the node's gRPC API can
	// be contacted.
	Addresses []string `protobuf:"bytes,2,rep,name=addresses,proto3" json:"addresses,omitempty"`
}

func (x *Node) Reset() {
	*x = Node{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Node) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Node) ProtoMessage() {}

func (x *Node) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Node.ProtoReflect.Descriptor instead.
func (*Node) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_rawDescGZIP(), []int{0}
}

func (x *Node) GetId() *uuidpb.UUID {
	if x != nil {
		return x.Id
	}
	return nil
}

func (x *Node) GetAddresses() []string {
	if x != nil {
		return x.Addresses
	}
	return nil
}

type Registration struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Node is the node that is registered.
	Node *Node `protobuf:"bytes,1,opt,name=node,proto3" json:"node,omitempty"`
	// ExpiresAt is the time at which the registration expires if not renewed.
	ExpiresAt *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=expires_at,json=expiresAt,proto3" json:"expires_at,omitempty"`
}

func (x *Registration) Reset() {
	*x = Registration{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Registration) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Registration) ProtoMessage() {}

func (x *Registration) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Registration.ProtoReflect.Descriptor instead.
func (*Registration) Descriptor() ([]byte, []int) {
	return file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_rawDescGZIP(), []int{1}
}

func (x *Registration) GetNode() *Node {
	if x != nil {
		return x.Node
	}
	return nil
}

func (x *Registration) GetExpiresAt() *timestamppb.Timestamp {
	if x != nil {
		return x.ExpiresAt
	}
	return nil
}

var File_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto protoreflect.FileDescriptor

var file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_rawDesc = []byte{
	0x0a, 0x54, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x67,
	0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2f,
	0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x70, 0x62, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1c, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79,
	0x2e, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x2e, 0x76, 0x31, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x38, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x65, 0x6e, 0x67, 0x69, 0x6e,
	0x65, 0x6b, 0x69, 0x74, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x75, 0x75,
	0x69, 0x64, 0x70, 0x62, 0x2f, 0x75, 0x75, 0x69, 0x64, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0x4a, 0x0a, 0x04, 0x4e, 0x6f, 0x64, 0x65, 0x12, 0x24, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x64, 0x6f, 0x67, 0x6d, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x55, 0x49, 0x44, 0x52, 0x02, 0x69, 0x64, 0x12, 0x1c, 0x0a,
	0x09, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x09, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x65, 0x73, 0x22, 0x81, 0x01, 0x0a, 0x0c,
	0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x36, 0x0a, 0x04,
	0x6e, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x76, 0x65, 0x72,
	0x61, 0x63, 0x69, 0x74, 0x79, 0x2e, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2e, 0x72, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x2e, 0x4e, 0x6f, 0x64, 0x65, 0x52, 0x04,
	0x6e, 0x6f, 0x64, 0x65, 0x12, 0x39, 0x0a, 0x0a, 0x65, 0x78, 0x70, 0x69, 0x72, 0x65, 0x73, 0x5f,
	0x61, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x65, 0x78, 0x70, 0x69, 0x72, 0x65, 0x73, 0x41, 0x74, 0x42,
	0x43, 0x5a, 0x41, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f,
	0x67, 0x6d, 0x61, 0x74, 0x69, 0x71, 0x2f, 0x76, 0x65, 0x72, 0x61, 0x63, 0x69, 0x74, 0x79, 0x2f,
	0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72,
	0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74,
	0x72, 0x79, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_rawDescOnce sync.Once
	file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_rawDescData = file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_rawDesc
)

func file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_rawDescGZIP() []byte {
	file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_rawDescOnce.Do(func() {
		file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_rawDescData)
	})
	return file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_rawDescData
}

var file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_goTypes = []any{
	(*Node)(nil),                  // 0: veracity.cluster.registry.v1.Node
	(*Registration)(nil),          // 1: veracity.cluster.registry.v1.Registration
	(*uuidpb.UUID)(nil),           // 2: dogma.protobuf.UUID
	(*timestamppb.Timestamp)(nil), // 3: google.protobuf.Timestamp
}
var file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_depIdxs = []int32{
	2, // 0: veracity.cluster.registry.v1.Node.id:type_name -> dogma.protobuf.UUID
	0, // 1: veracity.cluster.registry.v1.Registration.node:type_name -> veracity.cluster.registry.v1.Node
	3, // 2: veracity.cluster.registry.v1.Registration.expires_at:type_name -> google.protobuf.Timestamp
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() {
	file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_init()
}
func file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_init() {
	if File_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*Node); i {
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
		file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*Registration); i {
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
			RawDescriptor: file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_goTypes,
		DependencyIndexes: file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_depIdxs,
		MessageInfos:      file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_msgTypes,
	}.Build()
	File_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto = out.File
	file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_rawDesc = nil
	file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_goTypes = nil
	file_github_com_dogmatiq_veracity_internal_cluster_internal_registrypb_registration_proto_depIdxs = nil
}
