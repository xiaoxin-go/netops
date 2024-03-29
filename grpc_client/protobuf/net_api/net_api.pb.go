// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.8
// source: protobuf/net_api/net_api.proto

package net_api

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

type Command struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id     int32  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Cmd    string `protobuf:"bytes,2,opt,name=cmd,proto3" json:"cmd,omitempty"`
	Status string `protobuf:"bytes,3,opt,name=status,proto3" json:"status,omitempty"`
	Result string `protobuf:"bytes,4,opt,name=result,proto3" json:"result,omitempty"`
}

func (x *Command) Reset() {
	*x = Command{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protobuf_net_api_net_api_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Command) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Command) ProtoMessage() {}

func (x *Command) ProtoReflect() protoreflect.Message {
	mi := &file_protobuf_net_api_net_api_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Command.ProtoReflect.Descriptor instead.
func (*Command) Descriptor() ([]byte, []int) {
	return file_protobuf_net_api_net_api_proto_rawDescGZIP(), []int{0}
}

func (x *Command) GetId() int32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Command) GetCmd() string {
	if x != nil {
		return x.Cmd
	}
	return ""
}

func (x *Command) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

func (x *Command) GetResult() string {
	if x != nil {
		return x.Result
	}
	return ""
}

type ConfigRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	DeviceType     string     `protobuf:"bytes,1,opt,name=device_type,json=deviceType,proto3" json:"device_type,omitempty"`
	Host           string     `protobuf:"bytes,2,opt,name=host,proto3" json:"host,omitempty"`
	Username       string     `protobuf:"bytes,3,opt,name=username,proto3" json:"username,omitempty"`
	Password       string     `protobuf:"bytes,4,opt,name=password,proto3" json:"password,omitempty"`
	EnablePassword string     `protobuf:"bytes,5,opt,name=enable_password,json=enablePassword,proto3" json:"enable_password,omitempty"`
	Port           int32      `protobuf:"varint,6,opt,name=port,proto3" json:"port,omitempty"`
	Commands       []*Command `protobuf:"bytes,7,rep,name=commands,proto3" json:"commands,omitempty"`
}

func (x *ConfigRequest) Reset() {
	*x = ConfigRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protobuf_net_api_net_api_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConfigRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConfigRequest) ProtoMessage() {}

func (x *ConfigRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protobuf_net_api_net_api_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConfigRequest.ProtoReflect.Descriptor instead.
func (*ConfigRequest) Descriptor() ([]byte, []int) {
	return file_protobuf_net_api_net_api_proto_rawDescGZIP(), []int{1}
}

func (x *ConfigRequest) GetDeviceType() string {
	if x != nil {
		return x.DeviceType
	}
	return ""
}

func (x *ConfigRequest) GetHost() string {
	if x != nil {
		return x.Host
	}
	return ""
}

func (x *ConfigRequest) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *ConfigRequest) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

func (x *ConfigRequest) GetEnablePassword() string {
	if x != nil {
		return x.EnablePassword
	}
	return ""
}

func (x *ConfigRequest) GetPort() int32 {
	if x != nil {
		return x.Port
	}
	return 0
}

func (x *ConfigRequest) GetCommands() []*Command {
	if x != nil {
		return x.Commands
	}
	return nil
}

type ConfigResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Code    int32      `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Message string     `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
	Results []*Command `protobuf:"bytes,3,rep,name=results,proto3" json:"results,omitempty"`
}

func (x *ConfigResponse) Reset() {
	*x = ConfigResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protobuf_net_api_net_api_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConfigResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConfigResponse) ProtoMessage() {}

func (x *ConfigResponse) ProtoReflect() protoreflect.Message {
	mi := &file_protobuf_net_api_net_api_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConfigResponse.ProtoReflect.Descriptor instead.
func (*ConfigResponse) Descriptor() ([]byte, []int) {
	return file_protobuf_net_api_net_api_proto_rawDescGZIP(), []int{2}
}

func (x *ConfigResponse) GetCode() int32 {
	if x != nil {
		return x.Code
	}
	return 0
}

func (x *ConfigResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *ConfigResponse) GetResults() []*Command {
	if x != nil {
		return x.Results
	}
	return nil
}

type HttpRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Url      string `protobuf:"bytes,1,opt,name=url,proto3" json:"url,omitempty"`
	Method   string `protobuf:"bytes,2,opt,name=method,proto3" json:"method,omitempty"`
	Username string `protobuf:"bytes,3,opt,name=username,proto3" json:"username,omitempty"`
	Password string `protobuf:"bytes,4,opt,name=password,proto3" json:"password,omitempty"`
	Params   string `protobuf:"bytes,5,opt,name=params,proto3" json:"params,omitempty"`
}

func (x *HttpRequest) Reset() {
	*x = HttpRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protobuf_net_api_net_api_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HttpRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HttpRequest) ProtoMessage() {}

func (x *HttpRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protobuf_net_api_net_api_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HttpRequest.ProtoReflect.Descriptor instead.
func (*HttpRequest) Descriptor() ([]byte, []int) {
	return file_protobuf_net_api_net_api_proto_rawDescGZIP(), []int{3}
}

func (x *HttpRequest) GetUrl() string {
	if x != nil {
		return x.Url
	}
	return ""
}

func (x *HttpRequest) GetMethod() string {
	if x != nil {
		return x.Method
	}
	return ""
}

func (x *HttpRequest) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *HttpRequest) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

func (x *HttpRequest) GetParams() string {
	if x != nil {
		return x.Params
	}
	return ""
}

type HttpResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Code    int32  `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *HttpResponse) Reset() {
	*x = HttpResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protobuf_net_api_net_api_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HttpResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HttpResponse) ProtoMessage() {}

func (x *HttpResponse) ProtoReflect() protoreflect.Message {
	mi := &file_protobuf_net_api_net_api_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HttpResponse.ProtoReflect.Descriptor instead.
func (*HttpResponse) Descriptor() ([]byte, []int) {
	return file_protobuf_net_api_net_api_proto_rawDescGZIP(), []int{4}
}

func (x *HttpResponse) GetCode() int32 {
	if x != nil {
		return x.Code
	}
	return 0
}

func (x *HttpResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

var File_protobuf_net_api_net_api_proto protoreflect.FileDescriptor

var file_protobuf_net_api_net_api_proto_rawDesc = []byte{
	0x0a, 0x1e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x6e, 0x65, 0x74, 0x5f, 0x61,
	0x70, 0x69, 0x2f, 0x6e, 0x65, 0x74, 0x5f, 0x61, 0x70, 0x69, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x06, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x22, 0x5b, 0x0a, 0x07, 0x43, 0x6f, 0x6d, 0x6d,
	0x61, 0x6e, 0x64, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x02, 0x69, 0x64, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x6d, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x03, 0x63, 0x6d, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x16, 0x0a,
	0x06, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72,
	0x65, 0x73, 0x75, 0x6c, 0x74, 0x22, 0xe6, 0x01, 0x0a, 0x0d, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x64, 0x65, 0x76, 0x69, 0x63,
	0x65, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x64, 0x65,
	0x76, 0x69, 0x63, 0x65, 0x54, 0x79, 0x70, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x6f, 0x73, 0x74,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x6f, 0x73, 0x74, 0x12, 0x1a, 0x0a, 0x08,
	0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08,
	0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x61, 0x73, 0x73,
	0x77, 0x6f, 0x72, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x70, 0x61, 0x73, 0x73,
	0x77, 0x6f, 0x72, 0x64, 0x12, 0x27, 0x0a, 0x0f, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x70,
	0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x65,
	0x6e, 0x61, 0x62, 0x6c, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x12, 0x12, 0x0a,
	0x04, 0x70, 0x6f, 0x72, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x05, 0x52, 0x04, 0x70, 0x6f, 0x72,
	0x74, 0x12, 0x2b, 0x0a, 0x08, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x73, 0x18, 0x07, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x43, 0x6f, 0x6d,
	0x6d, 0x61, 0x6e, 0x64, 0x52, 0x08, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x73, 0x22, 0x69,
	0x0a, 0x0e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x12, 0x0a, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x04,
	0x63, 0x6f, 0x64, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x29,
	0x0a, 0x07, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x0f, 0x2e, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64,
	0x52, 0x07, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73, 0x22, 0x87, 0x01, 0x0a, 0x0b, 0x48, 0x74,
	0x74, 0x70, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x72, 0x6c,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x75, 0x72, 0x6c, 0x12, 0x16, 0x0a, 0x06, 0x6d,
	0x65, 0x74, 0x68, 0x6f, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x6d, 0x65, 0x74,
	0x68, 0x6f, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x12,
	0x1a, 0x0a, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x70,
	0x61, 0x72, 0x61, 0x6d, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x70, 0x61, 0x72,
	0x61, 0x6d, 0x73, 0x22, 0x3c, 0x0a, 0x0c, 0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x32, 0xb1, 0x01, 0x0a, 0x06, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x12, 0x39, 0x0a, 0x06,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x15, 0x2e, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e,
	0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x37, 0x0a, 0x04, 0x53, 0x68, 0x6f, 0x77, 0x12,
	0x15, 0x2e, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00,
	0x12, 0x33, 0x0a, 0x04, 0x48, 0x74, 0x74, 0x70, 0x12, 0x13, 0x2e, 0x64, 0x65, 0x76, 0x69, 0x63,
	0x65, 0x2e, 0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x14, 0x2e,
	0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x0c, 0x5a, 0x0a, 0x2e, 0x2f, 0x3b, 0x6e, 0x65, 0x74, 0x5f,
	0x61, 0x70, 0x69, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_protobuf_net_api_net_api_proto_rawDescOnce sync.Once
	file_protobuf_net_api_net_api_proto_rawDescData = file_protobuf_net_api_net_api_proto_rawDesc
)

func file_protobuf_net_api_net_api_proto_rawDescGZIP() []byte {
	file_protobuf_net_api_net_api_proto_rawDescOnce.Do(func() {
		file_protobuf_net_api_net_api_proto_rawDescData = protoimpl.X.CompressGZIP(file_protobuf_net_api_net_api_proto_rawDescData)
	})
	return file_protobuf_net_api_net_api_proto_rawDescData
}

var file_protobuf_net_api_net_api_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_protobuf_net_api_net_api_proto_goTypes = []interface{}{
	(*Command)(nil),        // 0: device.Command
	(*ConfigRequest)(nil),  // 1: device.ConfigRequest
	(*ConfigResponse)(nil), // 2: device.ConfigResponse
	(*HttpRequest)(nil),    // 3: device.HttpRequest
	(*HttpResponse)(nil),   // 4: device.HttpResponse
}
var file_protobuf_net_api_net_api_proto_depIdxs = []int32{
	0, // 0: device.ConfigRequest.commands:type_name -> device.Command
	0, // 1: device.ConfigResponse.results:type_name -> device.Command
	1, // 2: device.Device.Config:input_type -> device.ConfigRequest
	1, // 3: device.Device.Show:input_type -> device.ConfigRequest
	3, // 4: device.Device.Http:input_type -> device.HttpRequest
	2, // 5: device.Device.Config:output_type -> device.ConfigResponse
	2, // 6: device.Device.Show:output_type -> device.ConfigResponse
	4, // 7: device.Device.Http:output_type -> device.HttpResponse
	5, // [5:8] is the sub-list for method output_type
	2, // [2:5] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_protobuf_net_api_net_api_proto_init() }
func file_protobuf_net_api_net_api_proto_init() {
	if File_protobuf_net_api_net_api_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_protobuf_net_api_net_api_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Command); i {
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
		file_protobuf_net_api_net_api_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ConfigRequest); i {
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
		file_protobuf_net_api_net_api_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ConfigResponse); i {
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
		file_protobuf_net_api_net_api_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HttpRequest); i {
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
		file_protobuf_net_api_net_api_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HttpResponse); i {
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
			RawDescriptor: file_protobuf_net_api_net_api_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_protobuf_net_api_net_api_proto_goTypes,
		DependencyIndexes: file_protobuf_net_api_net_api_proto_depIdxs,
		MessageInfos:      file_protobuf_net_api_net_api_proto_msgTypes,
	}.Build()
	File_protobuf_net_api_net_api_proto = out.File
	file_protobuf_net_api_net_api_proto_rawDesc = nil
	file_protobuf_net_api_net_api_proto_goTypes = nil
	file_protobuf_net_api_net_api_proto_depIdxs = nil
}
