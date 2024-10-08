// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: structs/structs/grid.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type GridRecord struct {
	AttributeId string `protobuf:"bytes,1,opt,name=attributeId,proto3" json:"attributeId,omitempty"`
	Value       uint64 `protobuf:"varint,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (m *GridRecord) Reset()         { *m = GridRecord{} }
func (m *GridRecord) String() string { return proto.CompactTextString(m) }
func (*GridRecord) ProtoMessage()    {}
func (*GridRecord) Descriptor() ([]byte, []int) {
	return fileDescriptor_e87ac0223f374538, []int{0}
}
func (m *GridRecord) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GridRecord) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GridRecord.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GridRecord) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GridRecord.Merge(m, src)
}
func (m *GridRecord) XXX_Size() int {
	return m.Size()
}
func (m *GridRecord) XXX_DiscardUnknown() {
	xxx_messageInfo_GridRecord.DiscardUnknown(m)
}

var xxx_messageInfo_GridRecord proto.InternalMessageInfo

func (m *GridRecord) GetAttributeId() string {
	if m != nil {
		return m.AttributeId
	}
	return ""
}

func (m *GridRecord) GetValue() uint64 {
	if m != nil {
		return m.Value
	}
	return 0
}

type GridAttributes struct {
	Ore                    uint64 `protobuf:"varint,1,opt,name=ore,proto3" json:"ore,omitempty"`
	Fuel                   uint64 `protobuf:"varint,2,opt,name=fuel,proto3" json:"fuel,omitempty"`
	Capacity               uint64 `protobuf:"varint,3,opt,name=capacity,proto3" json:"capacity,omitempty"`
	Load                   uint64 `protobuf:"varint,4,opt,name=load,proto3" json:"load,omitempty"`
	StructsLoad            uint64 `protobuf:"varint,5,opt,name=structsLoad,proto3" json:"structsLoad,omitempty"`
	Power                  uint64 `protobuf:"varint,6,opt,name=power,proto3" json:"power,omitempty"`
	ConnectionCapacity     uint64 `protobuf:"varint,7,opt,name=connectionCapacity,proto3" json:"connectionCapacity,omitempty"`
	ConnectionCount        uint64 `protobuf:"varint,8,opt,name=connectionCount,proto3" json:"connectionCount,omitempty"`
	AllocationPointerStart uint64 `protobuf:"varint,9,opt,name=allocationPointerStart,proto3" json:"allocationPointerStart,omitempty"`
	AllocationPointerEnd   uint64 `protobuf:"varint,10,opt,name=allocationPointerEnd,proto3" json:"allocationPointerEnd,omitempty"`
	ProxyNonce             uint64 `protobuf:"varint,11,opt,name=proxyNonce,proto3" json:"proxyNonce,omitempty"`
	LastAction             uint64 `protobuf:"varint,12,opt,name=lastAction,proto3" json:"lastAction,omitempty"`
	Nonce                  uint64 `protobuf:"varint,13,opt,name=nonce,proto3" json:"nonce,omitempty"`
	Ready                  uint64 `protobuf:"varint,14,opt,name=ready,proto3" json:"ready,omitempty"`
}

func (m *GridAttributes) Reset()         { *m = GridAttributes{} }
func (m *GridAttributes) String() string { return proto.CompactTextString(m) }
func (*GridAttributes) ProtoMessage()    {}
func (*GridAttributes) Descriptor() ([]byte, []int) {
	return fileDescriptor_e87ac0223f374538, []int{1}
}
func (m *GridAttributes) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GridAttributes) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GridAttributes.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GridAttributes) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GridAttributes.Merge(m, src)
}
func (m *GridAttributes) XXX_Size() int {
	return m.Size()
}
func (m *GridAttributes) XXX_DiscardUnknown() {
	xxx_messageInfo_GridAttributes.DiscardUnknown(m)
}

var xxx_messageInfo_GridAttributes proto.InternalMessageInfo

func (m *GridAttributes) GetOre() uint64 {
	if m != nil {
		return m.Ore
	}
	return 0
}

func (m *GridAttributes) GetFuel() uint64 {
	if m != nil {
		return m.Fuel
	}
	return 0
}

func (m *GridAttributes) GetCapacity() uint64 {
	if m != nil {
		return m.Capacity
	}
	return 0
}

func (m *GridAttributes) GetLoad() uint64 {
	if m != nil {
		return m.Load
	}
	return 0
}

func (m *GridAttributes) GetStructsLoad() uint64 {
	if m != nil {
		return m.StructsLoad
	}
	return 0
}

func (m *GridAttributes) GetPower() uint64 {
	if m != nil {
		return m.Power
	}
	return 0
}

func (m *GridAttributes) GetConnectionCapacity() uint64 {
	if m != nil {
		return m.ConnectionCapacity
	}
	return 0
}

func (m *GridAttributes) GetConnectionCount() uint64 {
	if m != nil {
		return m.ConnectionCount
	}
	return 0
}

func (m *GridAttributes) GetAllocationPointerStart() uint64 {
	if m != nil {
		return m.AllocationPointerStart
	}
	return 0
}

func (m *GridAttributes) GetAllocationPointerEnd() uint64 {
	if m != nil {
		return m.AllocationPointerEnd
	}
	return 0
}

func (m *GridAttributes) GetProxyNonce() uint64 {
	if m != nil {
		return m.ProxyNonce
	}
	return 0
}

func (m *GridAttributes) GetLastAction() uint64 {
	if m != nil {
		return m.LastAction
	}
	return 0
}

func (m *GridAttributes) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *GridAttributes) GetReady() uint64 {
	if m != nil {
		return m.Ready
	}
	return 0
}

func init() {
	proto.RegisterType((*GridRecord)(nil), "structs.structs.GridRecord")
	proto.RegisterType((*GridAttributes)(nil), "structs.structs.GridAttributes")
}

func init() { proto.RegisterFile("structs/structs/grid.proto", fileDescriptor_e87ac0223f374538) }

var fileDescriptor_e87ac0223f374538 = []byte{
	// 372 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x92, 0xcf, 0x4e, 0xc2, 0x40,
	0x10, 0xc6, 0xa9, 0x14, 0x84, 0x41, 0xc1, 0x6c, 0x88, 0x6e, 0x38, 0x34, 0x84, 0x13, 0x27, 0x88,
	0x9a, 0x78, 0xc7, 0x3f, 0x31, 0x26, 0xc6, 0x98, 0x7a, 0xf3, 0xb6, 0x6c, 0x57, 0xd2, 0xa4, 0xd9,
	0x6d, 0xb6, 0x5b, 0xa5, 0x6f, 0xe1, 0x63, 0x79, 0xe4, 0xe8, 0xd1, 0xc0, 0x8b, 0x98, 0x9d, 0xb6,
	0xd8, 0x28, 0x9e, 0x76, 0xbe, 0xdf, 0xf7, 0xcd, 0x76, 0xba, 0x19, 0x18, 0x24, 0x46, 0xa7, 0xdc,
	0x24, 0xd3, 0xf2, 0x5c, 0xe8, 0x30, 0x98, 0xc4, 0x5a, 0x19, 0x45, 0x7a, 0x05, 0x9b, 0x14, 0xe7,
	0xa0, 0xbf, 0x50, 0x0b, 0x85, 0xde, 0xd4, 0x56, 0x79, 0x6c, 0x74, 0x0d, 0x70, 0xab, 0xc3, 0xc0,
	0x17, 0x5c, 0xe9, 0x80, 0x0c, 0xa1, 0xc3, 0x8c, 0xd1, 0xe1, 0x3c, 0x35, 0xe2, 0x2e, 0xa0, 0xce,
	0xd0, 0x19, 0xb7, 0xfd, 0x2a, 0x22, 0x7d, 0x68, 0xbc, 0xb2, 0x28, 0x15, 0x74, 0x6f, 0xe8, 0x8c,
	0x5d, 0x3f, 0x17, 0xa3, 0x55, 0x1d, 0xba, 0xf6, 0x9a, 0x59, 0x99, 0x4c, 0xc8, 0x11, 0xd4, 0x95,
	0x16, 0x78, 0x85, 0xeb, 0xdb, 0x92, 0x10, 0x70, 0x5f, 0x52, 0x11, 0x15, 0x9d, 0x58, 0x93, 0x01,
	0xb4, 0x38, 0x8b, 0x19, 0x0f, 0x4d, 0x46, 0xeb, 0xc8, 0xb7, 0xda, 0xe6, 0x23, 0xc5, 0x02, 0xea,
	0xe6, 0x79, 0x5b, 0xdb, 0x01, 0x8b, 0xff, 0xb9, 0xb7, 0x56, 0x03, 0xad, 0x2a, 0xb2, 0x03, 0xc6,
	0xea, 0x4d, 0x68, 0xda, 0xcc, 0x07, 0x44, 0x41, 0x26, 0x40, 0xb8, 0x92, 0x52, 0x70, 0x13, 0x2a,
	0x79, 0x55, 0x7e, 0x71, 0x1f, 0x23, 0x3b, 0x1c, 0x32, 0x86, 0x5e, 0x85, 0xaa, 0x54, 0x1a, 0xda,
	0xc2, 0xf0, 0x6f, 0x4c, 0x2e, 0xe0, 0x98, 0x45, 0x91, 0xe2, 0xcc, 0xa2, 0x47, 0x15, 0x4a, 0x23,
	0xf4, 0x93, 0x61, 0xda, 0xd0, 0x36, 0x36, 0xfc, 0xe3, 0x92, 0x33, 0xe8, 0xff, 0x71, 0x6e, 0x64,
	0x40, 0x01, 0xbb, 0x76, 0x7a, 0xc4, 0x03, 0x88, 0xb5, 0x5a, 0x66, 0x0f, 0x4a, 0x72, 0x41, 0x3b,
	0x98, 0xac, 0x10, 0xeb, 0x47, 0x2c, 0x31, 0x33, 0x1c, 0x8f, 0x1e, 0xe4, 0xfe, 0x0f, 0xb1, 0x6f,
	0x23, 0xb1, 0xf5, 0x30, 0x7f, 0x1b, 0x14, 0x96, 0x6a, 0xc1, 0x82, 0x8c, 0x76, 0x73, 0x8a, 0xe2,
	0xf2, 0xf4, 0x63, 0xed, 0x39, 0xab, 0xb5, 0xe7, 0x7c, 0xad, 0x3d, 0xe7, 0x7d, 0xe3, 0xd5, 0x56,
	0x1b, 0xaf, 0xf6, 0xb9, 0xf1, 0x6a, 0xcf, 0x27, 0xe5, 0xb6, 0x2d, 0xb7, 0x7b, 0x67, 0xb2, 0x58,
	0x24, 0xf3, 0x26, 0xae, 0xd4, 0xf9, 0x77, 0x00, 0x00, 0x00, 0xff, 0xff, 0xb0, 0xa5, 0x57, 0x08,
	0x97, 0x02, 0x00, 0x00,
}

func (m *GridRecord) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GridRecord) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GridRecord) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Value != 0 {
		i = encodeVarintGrid(dAtA, i, uint64(m.Value))
		i--
		dAtA[i] = 0x10
	}
	if len(m.AttributeId) > 0 {
		i -= len(m.AttributeId)
		copy(dAtA[i:], m.AttributeId)
		i = encodeVarintGrid(dAtA, i, uint64(len(m.AttributeId)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *GridAttributes) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GridAttributes) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GridAttributes) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Ready != 0 {
		i = encodeVarintGrid(dAtA, i, uint64(m.Ready))
		i--
		dAtA[i] = 0x70
	}
	if m.Nonce != 0 {
		i = encodeVarintGrid(dAtA, i, uint64(m.Nonce))
		i--
		dAtA[i] = 0x68
	}
	if m.LastAction != 0 {
		i = encodeVarintGrid(dAtA, i, uint64(m.LastAction))
		i--
		dAtA[i] = 0x60
	}
	if m.ProxyNonce != 0 {
		i = encodeVarintGrid(dAtA, i, uint64(m.ProxyNonce))
		i--
		dAtA[i] = 0x58
	}
	if m.AllocationPointerEnd != 0 {
		i = encodeVarintGrid(dAtA, i, uint64(m.AllocationPointerEnd))
		i--
		dAtA[i] = 0x50
	}
	if m.AllocationPointerStart != 0 {
		i = encodeVarintGrid(dAtA, i, uint64(m.AllocationPointerStart))
		i--
		dAtA[i] = 0x48
	}
	if m.ConnectionCount != 0 {
		i = encodeVarintGrid(dAtA, i, uint64(m.ConnectionCount))
		i--
		dAtA[i] = 0x40
	}
	if m.ConnectionCapacity != 0 {
		i = encodeVarintGrid(dAtA, i, uint64(m.ConnectionCapacity))
		i--
		dAtA[i] = 0x38
	}
	if m.Power != 0 {
		i = encodeVarintGrid(dAtA, i, uint64(m.Power))
		i--
		dAtA[i] = 0x30
	}
	if m.StructsLoad != 0 {
		i = encodeVarintGrid(dAtA, i, uint64(m.StructsLoad))
		i--
		dAtA[i] = 0x28
	}
	if m.Load != 0 {
		i = encodeVarintGrid(dAtA, i, uint64(m.Load))
		i--
		dAtA[i] = 0x20
	}
	if m.Capacity != 0 {
		i = encodeVarintGrid(dAtA, i, uint64(m.Capacity))
		i--
		dAtA[i] = 0x18
	}
	if m.Fuel != 0 {
		i = encodeVarintGrid(dAtA, i, uint64(m.Fuel))
		i--
		dAtA[i] = 0x10
	}
	if m.Ore != 0 {
		i = encodeVarintGrid(dAtA, i, uint64(m.Ore))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintGrid(dAtA []byte, offset int, v uint64) int {
	offset -= sovGrid(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *GridRecord) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.AttributeId)
	if l > 0 {
		n += 1 + l + sovGrid(uint64(l))
	}
	if m.Value != 0 {
		n += 1 + sovGrid(uint64(m.Value))
	}
	return n
}

func (m *GridAttributes) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Ore != 0 {
		n += 1 + sovGrid(uint64(m.Ore))
	}
	if m.Fuel != 0 {
		n += 1 + sovGrid(uint64(m.Fuel))
	}
	if m.Capacity != 0 {
		n += 1 + sovGrid(uint64(m.Capacity))
	}
	if m.Load != 0 {
		n += 1 + sovGrid(uint64(m.Load))
	}
	if m.StructsLoad != 0 {
		n += 1 + sovGrid(uint64(m.StructsLoad))
	}
	if m.Power != 0 {
		n += 1 + sovGrid(uint64(m.Power))
	}
	if m.ConnectionCapacity != 0 {
		n += 1 + sovGrid(uint64(m.ConnectionCapacity))
	}
	if m.ConnectionCount != 0 {
		n += 1 + sovGrid(uint64(m.ConnectionCount))
	}
	if m.AllocationPointerStart != 0 {
		n += 1 + sovGrid(uint64(m.AllocationPointerStart))
	}
	if m.AllocationPointerEnd != 0 {
		n += 1 + sovGrid(uint64(m.AllocationPointerEnd))
	}
	if m.ProxyNonce != 0 {
		n += 1 + sovGrid(uint64(m.ProxyNonce))
	}
	if m.LastAction != 0 {
		n += 1 + sovGrid(uint64(m.LastAction))
	}
	if m.Nonce != 0 {
		n += 1 + sovGrid(uint64(m.Nonce))
	}
	if m.Ready != 0 {
		n += 1 + sovGrid(uint64(m.Ready))
	}
	return n
}

func sovGrid(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozGrid(x uint64) (n int) {
	return sovGrid(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *GridRecord) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGrid
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: GridRecord: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GridRecord: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AttributeId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGrid
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGrid
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.AttributeId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			m.Value = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Value |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipGrid(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGrid
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *GridAttributes) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGrid
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: GridAttributes: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GridAttributes: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Ore", wireType)
			}
			m.Ore = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Ore |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Fuel", wireType)
			}
			m.Fuel = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Fuel |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Capacity", wireType)
			}
			m.Capacity = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Capacity |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Load", wireType)
			}
			m.Load = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Load |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field StructsLoad", wireType)
			}
			m.StructsLoad = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.StructsLoad |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Power", wireType)
			}
			m.Power = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Power |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ConnectionCapacity", wireType)
			}
			m.ConnectionCapacity = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ConnectionCapacity |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 8:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ConnectionCount", wireType)
			}
			m.ConnectionCount = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ConnectionCount |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 9:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field AllocationPointerStart", wireType)
			}
			m.AllocationPointerStart = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.AllocationPointerStart |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 10:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field AllocationPointerEnd", wireType)
			}
			m.AllocationPointerEnd = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.AllocationPointerEnd |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 11:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProxyNonce", wireType)
			}
			m.ProxyNonce = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ProxyNonce |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 12:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastAction", wireType)
			}
			m.LastAction = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LastAction |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 13:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Nonce", wireType)
			}
			m.Nonce = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Nonce |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 14:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Ready", wireType)
			}
			m.Ready = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Ready |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipGrid(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGrid
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipGrid(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowGrid
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowGrid
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthGrid
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupGrid
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthGrid
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthGrid        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowGrid          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupGrid = fmt.Errorf("proto: unexpected end of group")
)
