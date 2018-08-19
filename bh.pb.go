// Code generated by protoc-gen-go. DO NOT EDIT.
// source: bh.proto

/*
Package main is a generated protocol buffer package.

It is generated from these files:
	bh.proto

It has these top-level messages:
	BhBlock
	BhFile
	BhMessage
*/
package main

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type BhMessage_BhMessageType int32

const (
	BhMessage_BH_PING  BhMessage_BhMessageType = 1
	BhMessage_BH_PEERS BhMessage_BhMessageType = 2
)

var BhMessage_BhMessageType_name = map[int32]string{
	1: "BH_PING",
	2: "BH_PEERS",
}
var BhMessage_BhMessageType_value = map[string]int32{
	"BH_PING":  1,
	"BH_PEERS": 2,
}

func (x BhMessage_BhMessageType) Enum() *BhMessage_BhMessageType {
	p := new(BhMessage_BhMessageType)
	*p = x
	return p
}
func (x BhMessage_BhMessageType) String() string {
	return proto.EnumName(BhMessage_BhMessageType_name, int32(x))
}
func (x *BhMessage_BhMessageType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(BhMessage_BhMessageType_value, data, "BhMessage_BhMessageType")
	if err != nil {
		return err
	}
	*x = BhMessage_BhMessageType(value)
	return nil
}
func (BhMessage_BhMessageType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{2, 0} }

type BhBlock struct {
	Offset           *int64 `protobuf:"varint,1,req,name=offset" json:"offset,omitempty"`
	Length           *int64 `protobuf:"varint,2,req,name=length" json:"length,omitempty"`
	Hash             []byte `protobuf:"bytes,3,req,name=hash" json:"hash,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *BhBlock) Reset()                    { *m = BhBlock{} }
func (m *BhBlock) String() string            { return proto.CompactTextString(m) }
func (*BhBlock) ProtoMessage()               {}
func (*BhBlock) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *BhBlock) GetOffset() int64 {
	if m != nil && m.Offset != nil {
		return *m.Offset
	}
	return 0
}

func (m *BhBlock) GetLength() int64 {
	if m != nil && m.Length != nil {
		return *m.Length
	}
	return 0
}

func (m *BhBlock) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

type BhFile struct {
	Name             []byte     `protobuf:"bytes,1,req,name=name" json:"name,omitempty"`
	Modified         *int64     `protobuf:"varint,2,req,name=modified" json:"modified,omitempty"`
	Blocks           []*BhBlock `protobuf:"bytes,3,rep,name=blocks" json:"blocks,omitempty"`
	XXX_unrecognized []byte     `json:"-"`
}

func (m *BhFile) Reset()                    { *m = BhFile{} }
func (m *BhFile) String() string            { return proto.CompactTextString(m) }
func (*BhFile) ProtoMessage()               {}
func (*BhFile) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *BhFile) GetName() []byte {
	if m != nil {
		return m.Name
	}
	return nil
}

func (m *BhFile) GetModified() int64 {
	if m != nil && m.Modified != nil {
		return *m.Modified
	}
	return 0
}

func (m *BhFile) GetBlocks() []*BhBlock {
	if m != nil {
		return m.Blocks
	}
	return nil
}

type BhMessage struct {
	Type             *BhMessage_BhMessageType `protobuf:"varint,1,req,name=type,enum=main.BhMessage_BhMessageType" json:"type,omitempty"`
	Peers            []*BhMessage_Peer        `protobuf:"bytes,2,rep,name=peers" json:"peers,omitempty"`
	Files            []*BhFile                `protobuf:"bytes,3,rep,name=files" json:"files,omitempty"`
	XXX_unrecognized []byte                   `json:"-"`
}

func (m *BhMessage) Reset()                    { *m = BhMessage{} }
func (m *BhMessage) String() string            { return proto.CompactTextString(m) }
func (*BhMessage) ProtoMessage()               {}
func (*BhMessage) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *BhMessage) GetType() BhMessage_BhMessageType {
	if m != nil && m.Type != nil {
		return *m.Type
	}
	return BhMessage_BH_PING
}

func (m *BhMessage) GetPeers() []*BhMessage_Peer {
	if m != nil {
		return m.Peers
	}
	return nil
}

func (m *BhMessage) GetFiles() []*BhFile {
	if m != nil {
		return m.Files
	}
	return nil
}

type BhMessage_Peer struct {
	Id               []byte   `protobuf:"bytes,1,req,name=id" json:"id,omitempty"`
	Addrs            [][]byte `protobuf:"bytes,2,rep,name=addrs" json:"addrs,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *BhMessage_Peer) Reset()                    { *m = BhMessage_Peer{} }
func (m *BhMessage_Peer) String() string            { return proto.CompactTextString(m) }
func (*BhMessage_Peer) ProtoMessage()               {}
func (*BhMessage_Peer) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2, 0} }

func (m *BhMessage_Peer) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *BhMessage_Peer) GetAddrs() [][]byte {
	if m != nil {
		return m.Addrs
	}
	return nil
}

func init() {
	proto.RegisterType((*BhBlock)(nil), "main.BhBlock")
	proto.RegisterType((*BhFile)(nil), "main.BhFile")
	proto.RegisterType((*BhMessage)(nil), "main.BhMessage")
	proto.RegisterType((*BhMessage_Peer)(nil), "main.BhMessage.Peer")
	proto.RegisterEnum("main.BhMessage_BhMessageType", BhMessage_BhMessageType_name, BhMessage_BhMessageType_value)
}

func init() { proto.RegisterFile("bh.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 288 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x5c, 0x4f, 0xdb, 0x6a, 0xeb, 0x30,
	0x10, 0x24, 0xb2, 0x73, 0x39, 0x1b, 0x27, 0x84, 0x25, 0x1c, 0x4c, 0xa0, 0x10, 0x0c, 0x85, 0x10,
	0x8a, 0xa1, 0xf9, 0x04, 0x43, 0x7a, 0x79, 0x48, 0x09, 0x6a, 0xdf, 0x83, 0x53, 0xad, 0x23, 0x51,
	0xdf, 0xb0, 0xfc, 0x92, 0x3f, 0xee, 0x67, 0x14, 0x59, 0x72, 0x4b, 0xfb, 0xb6, 0xb3, 0x33, 0xb3,
	0x33, 0x0b, 0x93, 0xb3, 0x8c, 0xeb, 0xa6, 0x6a, 0x2b, 0xf4, 0x8b, 0x54, 0x95, 0xd1, 0x01, 0xc6,
	0x89, 0x4c, 0xf2, 0xea, 0xfd, 0x03, 0xff, 0xc3, 0xa8, 0xca, 0x32, 0x4d, 0x6d, 0x38, 0x58, 0xb3,
	0x8d, 0xc7, 0x1d, 0x32, 0xfb, 0x9c, 0xca, 0x4b, 0x2b, 0x43, 0x66, 0xf7, 0x16, 0x21, 0x82, 0x2f,
	0x53, 0x2d, 0x43, 0x6f, 0xcd, 0x36, 0x01, 0xef, 0xe6, 0xe8, 0x04, 0xa3, 0x44, 0x3e, 0xa8, 0x9c,
	0x0c, 0x5b, 0xa6, 0x05, 0x75, 0xb7, 0x02, 0xde, 0xcd, 0xb8, 0x82, 0x49, 0x51, 0x09, 0x95, 0x29,
	0x12, 0xee, 0xd6, 0x37, 0xc6, 0x5b, 0x18, 0x9d, 0x4d, 0x0d, 0x1d, 0x7a, 0x6b, 0x6f, 0x33, 0xdd,
	0xcd, 0x62, 0xd3, 0x2f, 0x76, 0xe5, 0xb8, 0x23, 0xa3, 0xcf, 0x01, 0xfc, 0x4b, 0xe4, 0x81, 0xb4,
	0x4e, 0x2f, 0x84, 0xf7, 0xe0, 0xb7, 0xd7, 0xda, 0x86, 0xcc, 0x77, 0x37, 0xbd, 0xc5, 0xd1, 0x3f,
	0xd3, 0xdb, 0xb5, 0x26, 0xde, 0x49, 0x71, 0x0b, 0xc3, 0x9a, 0xa8, 0xd1, 0x21, 0xeb, 0x62, 0x96,
	0x7f, 0x3d, 0x47, 0xa2, 0x86, 0x5b, 0x09, 0x46, 0x30, 0xcc, 0x54, 0x4e, 0x7d, 0xa5, 0xa0, 0xd7,
	0x9a, 0x07, 0xb9, 0xa5, 0x56, 0x77, 0xe0, 0x1b, 0x0b, 0xce, 0x81, 0x29, 0xe1, 0xbe, 0x65, 0x4a,
	0xe0, 0x12, 0x86, 0xa9, 0x10, 0x2e, 0x27, 0xe0, 0x16, 0x44, 0x5b, 0x98, 0xfd, 0x2a, 0x85, 0x53,
	0x18, 0x27, 0x4f, 0xa7, 0xe3, 0xf3, 0xcb, 0xe3, 0x62, 0x80, 0x01, 0x4c, 0x0c, 0xd8, 0xef, 0xf9,
	0xeb, 0x82, 0x7d, 0x05, 0x00, 0x00, 0xff, 0xff, 0x01, 0x91, 0x7c, 0x07, 0xab, 0x01, 0x00, 0x00,
}
