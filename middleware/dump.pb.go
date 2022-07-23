// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.12.4
// source: middleware/dump.proto

package middleware

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

type HeaderDump struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name   string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Values []string `protobuf:"bytes,2,rep,name=values,proto3" json:"values,omitempty"`
}

func (x *HeaderDump) Reset() {
	*x = HeaderDump{}
	if protoimpl.UnsafeEnabled {
		mi := &file_middleware_dump_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HeaderDump) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HeaderDump) ProtoMessage() {}

func (x *HeaderDump) ProtoReflect() protoreflect.Message {
	mi := &file_middleware_dump_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HeaderDump.ProtoReflect.Descriptor instead.
func (*HeaderDump) Descriptor() ([]byte, []int) {
	return file_middleware_dump_proto_rawDescGZIP(), []int{0}
}

func (x *HeaderDump) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *HeaderDump) GetValues() []string {
	if x != nil {
		return x.Values
	}
	return nil
}

type RequestDump struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Method  string        `protobuf:"bytes,1,opt,name=method,proto3" json:"method,omitempty"`
	Uri     string        `protobuf:"bytes,2,opt,name=uri,proto3" json:"uri,omitempty"`
	Proto   string        `protobuf:"bytes,3,opt,name=proto,proto3" json:"proto,omitempty"`
	Host    string        `protobuf:"bytes,4,opt,name=host,proto3" json:"host,omitempty"`
	Headers []*HeaderDump `protobuf:"bytes,5,rep,name=headers,proto3" json:"headers,omitempty"`
}

func (x *RequestDump) Reset() {
	*x = RequestDump{}
	if protoimpl.UnsafeEnabled {
		mi := &file_middleware_dump_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RequestDump) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RequestDump) ProtoMessage() {}

func (x *RequestDump) ProtoReflect() protoreflect.Message {
	mi := &file_middleware_dump_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RequestDump.ProtoReflect.Descriptor instead.
func (*RequestDump) Descriptor() ([]byte, []int) {
	return file_middleware_dump_proto_rawDescGZIP(), []int{1}
}

func (x *RequestDump) GetMethod() string {
	if x != nil {
		return x.Method
	}
	return ""
}

func (x *RequestDump) GetUri() string {
	if x != nil {
		return x.Uri
	}
	return ""
}

func (x *RequestDump) GetProto() string {
	if x != nil {
		return x.Proto
	}
	return ""
}

func (x *RequestDump) GetHost() string {
	if x != nil {
		return x.Host
	}
	return ""
}

func (x *RequestDump) GetHeaders() []*HeaderDump {
	if x != nil {
		return x.Headers
	}
	return nil
}

type ResponseDump struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Code    int32         `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Headers []*HeaderDump `protobuf:"bytes,3,rep,name=headers,proto3" json:"headers,omitempty"`
}

func (x *ResponseDump) Reset() {
	*x = ResponseDump{}
	if protoimpl.UnsafeEnabled {
		mi := &file_middleware_dump_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ResponseDump) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResponseDump) ProtoMessage() {}

func (x *ResponseDump) ProtoReflect() protoreflect.Message {
	mi := &file_middleware_dump_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResponseDump.ProtoReflect.Descriptor instead.
func (*ResponseDump) Descriptor() ([]byte, []int) {
	return file_middleware_dump_proto_rawDescGZIP(), []int{2}
}

func (x *ResponseDump) GetCode() int32 {
	if x != nil {
		return x.Code
	}
	return 0
}

func (x *ResponseDump) GetHeaders() []*HeaderDump {
	if x != nil {
		return x.Headers
	}
	return nil
}

var File_middleware_dump_proto protoreflect.FileDescriptor

var file_middleware_dump_proto_rawDesc = []byte{
	0x0a, 0x15, 0x6d, 0x69, 0x64, 0x64, 0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2f, 0x64, 0x75, 0x6d,
	0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x6d, 0x69, 0x64, 0x64, 0x6c, 0x65, 0x77,
	0x61, 0x72, 0x65, 0x22, 0x38, 0x0a, 0x0a, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x44, 0x75, 0x6d,
	0x70, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x18,
	0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x06, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x22, 0x93, 0x01,
	0x0a, 0x0b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x44, 0x75, 0x6d, 0x70, 0x12, 0x16, 0x0a,
	0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x6d,
	0x65, 0x74, 0x68, 0x6f, 0x64, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x72, 0x69, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x03, 0x75, 0x72, 0x69, 0x12, 0x14, 0x0a, 0x05, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x12, 0x0a,
	0x04, 0x68, 0x6f, 0x73, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x6f, 0x73,
	0x74, 0x12, 0x30, 0x0a, 0x07, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x18, 0x05, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x16, 0x2e, 0x6d, 0x69, 0x64, 0x64, 0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2e,
	0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x44, 0x75, 0x6d, 0x70, 0x52, 0x07, 0x68, 0x65, 0x61, 0x64,
	0x65, 0x72, 0x73, 0x22, 0x54, 0x0a, 0x0c, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x44,
	0x75, 0x6d, 0x70, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x12, 0x30, 0x0a, 0x07, 0x68, 0x65, 0x61, 0x64, 0x65,
	0x72, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x6d, 0x69, 0x64, 0x64, 0x6c,
	0x65, 0x77, 0x61, 0x72, 0x65, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x44, 0x75, 0x6d, 0x70,
	0x52, 0x07, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x42, 0x0e, 0x5a, 0x0c, 0x2e, 0x2f, 0x6d,
	0x69, 0x64, 0x64, 0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_middleware_dump_proto_rawDescOnce sync.Once
	file_middleware_dump_proto_rawDescData = file_middleware_dump_proto_rawDesc
)

func file_middleware_dump_proto_rawDescGZIP() []byte {
	file_middleware_dump_proto_rawDescOnce.Do(func() {
		file_middleware_dump_proto_rawDescData = protoimpl.X.CompressGZIP(file_middleware_dump_proto_rawDescData)
	})
	return file_middleware_dump_proto_rawDescData
}

var file_middleware_dump_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_middleware_dump_proto_goTypes = []interface{}{
	(*HeaderDump)(nil),   // 0: middleware.HeaderDump
	(*RequestDump)(nil),  // 1: middleware.RequestDump
	(*ResponseDump)(nil), // 2: middleware.ResponseDump
}
var file_middleware_dump_proto_depIdxs = []int32{
	0, // 0: middleware.RequestDump.headers:type_name -> middleware.HeaderDump
	0, // 1: middleware.ResponseDump.headers:type_name -> middleware.HeaderDump
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_middleware_dump_proto_init() }
func file_middleware_dump_proto_init() {
	if File_middleware_dump_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_middleware_dump_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HeaderDump); i {
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
		file_middleware_dump_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RequestDump); i {
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
		file_middleware_dump_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ResponseDump); i {
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
			RawDescriptor: file_middleware_dump_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_middleware_dump_proto_goTypes,
		DependencyIndexes: file_middleware_dump_proto_depIdxs,
		MessageInfos:      file_middleware_dump_proto_msgTypes,
	}.Build()
	File_middleware_dump_proto = out.File
	file_middleware_dump_proto_rawDesc = nil
	file_middleware_dump_proto_goTypes = nil
	file_middleware_dump_proto_depIdxs = nil
}
