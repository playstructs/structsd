// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: structs/structs/infusion.proto

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

type Infusion struct {
	DestinationType          ObjectType `protobuf:"varint,1,opt,name=destinationType,proto3,enum=structs.ObjectType" json:"destinationType,omitempty"`
	DestinationId            uint64     `protobuf:"varint,2,opt,name=destinationId,proto3" json:"destinationId,omitempty"`
	Fuel                     uint64     `protobuf:"varint,3,opt,name=fuel,proto3" json:"fuel,omitempty"`
	Energy                   uint64     `protobuf:"varint,4,opt,name=energy,proto3" json:"energy,omitempty"`
	LinkedSourceAllocationId uint64     `protobuf:"varint,5,opt,name=linkedSourceAllocationId,proto3" json:"linkedSourceAllocationId,omitempty"`
	LinkedPlayerAllocationId uint64     `protobuf:"varint,6,opt,name=linkedPlayerAllocationId,proto3" json:"linkedPlayerAllocationId,omitempty"`
	Address                  string     `protobuf:"bytes,7,opt,name=address,proto3" json:"address,omitempty"`
}

func (m *Infusion) Reset()         { *m = Infusion{} }
func (m *Infusion) String() string { return proto.CompactTextString(m) }
func (*Infusion) ProtoMessage()    {}
func (*Infusion) Descriptor() ([]byte, []int) {
	return fileDescriptor_95cd26029d28df40, []int{0}
}
func (m *Infusion) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Infusion) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Infusion.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Infusion) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Infusion.Merge(m, src)
}
func (m *Infusion) XXX_Size() int {
	return m.Size()
}
func (m *Infusion) XXX_DiscardUnknown() {
	xxx_messageInfo_Infusion.DiscardUnknown(m)
}

var xxx_messageInfo_Infusion proto.InternalMessageInfo

func (m *Infusion) GetDestinationType() ObjectType {
	if m != nil {
		return m.DestinationType
	}
	return ObjectType_guild
}

func (m *Infusion) GetDestinationId() uint64 {
	if m != nil {
		return m.DestinationId
	}
	return 0
}

func (m *Infusion) GetFuel() uint64 {
	if m != nil {
		return m.Fuel
	}
	return 0
}

func (m *Infusion) GetEnergy() uint64 {
	if m != nil {
		return m.Energy
	}
	return 0
}

func (m *Infusion) GetLinkedSourceAllocationId() uint64 {
	if m != nil {
		return m.LinkedSourceAllocationId
	}
	return 0
}

func (m *Infusion) GetLinkedPlayerAllocationId() uint64 {
	if m != nil {
		return m.LinkedPlayerAllocationId
	}
	return 0
}

func (m *Infusion) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

func init() {
	proto.RegisterType((*Infusion)(nil), "structs.Infusion")
}

func init() { proto.RegisterFile("structs/structs/infusion.proto", fileDescriptor_95cd26029d28df40) }

var fileDescriptor_95cd26029d28df40 = []byte{
	// 281 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x2b, 0x2e, 0x29, 0x2a,
	0x4d, 0x2e, 0x29, 0xd6, 0x87, 0xd1, 0x99, 0x79, 0x69, 0xa5, 0xc5, 0x99, 0xf9, 0x79, 0x7a, 0x05,
	0x45, 0xf9, 0x25, 0xf9, 0x42, 0xec, 0x50, 0x71, 0x29, 0x91, 0xf4, 0xfc, 0xf4, 0x7c, 0xb0, 0x98,
	0x3e, 0x88, 0x05, 0x91, 0x96, 0x92, 0x42, 0xd7, 0x9e, 0x9d, 0x5a, 0x59, 0x0c, 0x91, 0x53, 0x5a,
	0xcb, 0xc4, 0xc5, 0xe1, 0x09, 0x35, 0x4d, 0xc8, 0x96, 0x8b, 0x3f, 0x25, 0xb5, 0xb8, 0x24, 0x33,
	0x2f, 0xb1, 0x24, 0x33, 0x3f, 0x2f, 0xa4, 0xb2, 0x20, 0x55, 0x82, 0x51, 0x81, 0x51, 0x83, 0xcf,
	0x48, 0x58, 0x0f, 0xaa, 0x55, 0x2f, 0x3f, 0x29, 0x2b, 0x35, 0xb9, 0x04, 0x24, 0x15, 0x84, 0xae,
	0x56, 0x48, 0x85, 0x8b, 0x17, 0x49, 0xc8, 0x33, 0x45, 0x82, 0x49, 0x81, 0x51, 0x83, 0x25, 0x08,
	0x55, 0x50, 0x48, 0x88, 0x8b, 0x25, 0xad, 0x34, 0x35, 0x47, 0x82, 0x19, 0x2c, 0x09, 0x66, 0x0b,
	0x89, 0x71, 0xb1, 0xa5, 0xe6, 0xa5, 0x16, 0xa5, 0x57, 0x4a, 0xb0, 0x80, 0x45, 0xa1, 0x3c, 0x21,
	0x2b, 0x2e, 0x89, 0x9c, 0xcc, 0xbc, 0xec, 0xd4, 0x94, 0xe0, 0xfc, 0xd2, 0xa2, 0xe4, 0x54, 0xc7,
	0x9c, 0x9c, 0xfc, 0x64, 0x98, 0xe1, 0xac, 0x60, 0x95, 0x38, 0xe5, 0x11, 0x7a, 0x03, 0x72, 0x12,
	0x2b, 0x53, 0x8b, 0x50, 0xf4, 0xb2, 0x21, 0xeb, 0xc5, 0x94, 0x17, 0x92, 0xe0, 0x62, 0x4f, 0x4c,
	0x49, 0x29, 0x4a, 0x2d, 0x2e, 0x96, 0x60, 0x57, 0x60, 0xd4, 0xe0, 0x0c, 0x82, 0x71, 0x9d, 0x0c,
	0x4f, 0x3c, 0x92, 0x63, 0xbc, 0xf0, 0x48, 0x8e, 0xf1, 0xc1, 0x23, 0x39, 0xc6, 0x09, 0x8f, 0xe5,
	0x18, 0x2e, 0x3c, 0x96, 0x63, 0xb8, 0xf1, 0x58, 0x8e, 0x21, 0x4a, 0x1c, 0x16, 0xba, 0x15, 0xf0,
	0x70, 0x2e, 0xa9, 0x2c, 0x48, 0x2d, 0x4e, 0x62, 0x03, 0x87, 0xb4, 0x31, 0x20, 0x00, 0x00, 0xff,
	0xff, 0x21, 0xd9, 0x66, 0xd6, 0xc6, 0x01, 0x00, 0x00,
}

func (m *Infusion) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Infusion) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Infusion) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintInfusion(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0x3a
	}
	if m.LinkedPlayerAllocationId != 0 {
		i = encodeVarintInfusion(dAtA, i, uint64(m.LinkedPlayerAllocationId))
		i--
		dAtA[i] = 0x30
	}
	if m.LinkedSourceAllocationId != 0 {
		i = encodeVarintInfusion(dAtA, i, uint64(m.LinkedSourceAllocationId))
		i--
		dAtA[i] = 0x28
	}
	if m.Energy != 0 {
		i = encodeVarintInfusion(dAtA, i, uint64(m.Energy))
		i--
		dAtA[i] = 0x20
	}
	if m.Fuel != 0 {
		i = encodeVarintInfusion(dAtA, i, uint64(m.Fuel))
		i--
		dAtA[i] = 0x18
	}
	if m.DestinationId != 0 {
		i = encodeVarintInfusion(dAtA, i, uint64(m.DestinationId))
		i--
		dAtA[i] = 0x10
	}
	if m.DestinationType != 0 {
		i = encodeVarintInfusion(dAtA, i, uint64(m.DestinationType))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintInfusion(dAtA []byte, offset int, v uint64) int {
	offset -= sovInfusion(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Infusion) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.DestinationType != 0 {
		n += 1 + sovInfusion(uint64(m.DestinationType))
	}
	if m.DestinationId != 0 {
		n += 1 + sovInfusion(uint64(m.DestinationId))
	}
	if m.Fuel != 0 {
		n += 1 + sovInfusion(uint64(m.Fuel))
	}
	if m.Energy != 0 {
		n += 1 + sovInfusion(uint64(m.Energy))
	}
	if m.LinkedSourceAllocationId != 0 {
		n += 1 + sovInfusion(uint64(m.LinkedSourceAllocationId))
	}
	if m.LinkedPlayerAllocationId != 0 {
		n += 1 + sovInfusion(uint64(m.LinkedPlayerAllocationId))
	}
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovInfusion(uint64(l))
	}
	return n
}

func sovInfusion(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozInfusion(x uint64) (n int) {
	return sovInfusion(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Infusion) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowInfusion
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
			return fmt.Errorf("proto: Infusion: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Infusion: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field DestinationType", wireType)
			}
			m.DestinationType = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowInfusion
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.DestinationType |= ObjectType(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field DestinationId", wireType)
			}
			m.DestinationId = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowInfusion
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.DestinationId |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Fuel", wireType)
			}
			m.Fuel = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowInfusion
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
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Energy", wireType)
			}
			m.Energy = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowInfusion
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Energy |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LinkedSourceAllocationId", wireType)
			}
			m.LinkedSourceAllocationId = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowInfusion
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LinkedSourceAllocationId |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LinkedPlayerAllocationId", wireType)
			}
			m.LinkedPlayerAllocationId = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowInfusion
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LinkedPlayerAllocationId |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowInfusion
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
				return ErrInvalidLengthInfusion
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthInfusion
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Address = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipInfusion(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthInfusion
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
func skipInfusion(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowInfusion
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
					return 0, ErrIntOverflowInfusion
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
					return 0, ErrIntOverflowInfusion
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
				return 0, ErrInvalidLengthInfusion
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupInfusion
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthInfusion
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthInfusion        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowInfusion          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupInfusion = fmt.Errorf("proto: unexpected end of group")
)
