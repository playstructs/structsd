// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: structs/structs/infusion.proto

package types

import (
	cosmossdk_io_math "cosmossdk.io/math"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
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
	DestinationType ObjectType                  `protobuf:"varint,1,opt,name=destinationType,proto3,enum=structs.structs.ObjectType" json:"destinationType,omitempty"`
	DestinationId   string                      `protobuf:"bytes,2,opt,name=destinationId,proto3" json:"destinationId,omitempty"`
	Fuel            uint64                      `protobuf:"varint,3,opt,name=fuel,proto3" json:"fuel,omitempty"`
	Power           uint64                      `protobuf:"varint,4,opt,name=power,proto3" json:"power,omitempty"`
	Commission      cosmossdk_io_math.LegacyDec `protobuf:"bytes,5,opt,name=commission,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"commission"`
	PlayerId        string                      `protobuf:"bytes,6,opt,name=playerId,proto3" json:"playerId,omitempty"`
	Address         string                      `protobuf:"bytes,7,opt,name=address,proto3" json:"address,omitempty"`
	Ratio           uint64                      `protobuf:"varint,8,opt,name=ratio,proto3" json:"ratio,omitempty"`
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

func (m *Infusion) GetDestinationId() string {
	if m != nil {
		return m.DestinationId
	}
	return ""
}

func (m *Infusion) GetFuel() uint64 {
	if m != nil {
		return m.Fuel
	}
	return 0
}

func (m *Infusion) GetPower() uint64 {
	if m != nil {
		return m.Power
	}
	return 0
}

func (m *Infusion) GetPlayerId() string {
	if m != nil {
		return m.PlayerId
	}
	return ""
}

func (m *Infusion) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

func (m *Infusion) GetRatio() uint64 {
	if m != nil {
		return m.Ratio
	}
	return 0
}

func init() {
	proto.RegisterType((*Infusion)(nil), "structs.structs.Infusion")
}

func init() { proto.RegisterFile("structs/structs/infusion.proto", fileDescriptor_95cd26029d28df40) }

var fileDescriptor_95cd26029d28df40 = []byte{
	// 361 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x5c, 0x91, 0xcd, 0x4a, 0xeb, 0x40,
	0x14, 0xc7, 0x33, 0xbd, 0xfd, 0xba, 0x03, 0xf7, 0x96, 0x3b, 0x14, 0xee, 0x98, 0xc2, 0xb4, 0x88,
	0x8b, 0x22, 0x98, 0x50, 0x7d, 0x83, 0xd2, 0x4d, 0xa1, 0x1b, 0x83, 0x2b, 0x37, 0x92, 0x4e, 0xa6,
	0x75, 0x6c, 0x93, 0x09, 0x99, 0x29, 0x9a, 0xb7, 0xf0, 0x31, 0x5c, 0xba, 0xf0, 0x1d, 0xec, 0xb2,
	0xb8, 0x12, 0x17, 0x45, 0xda, 0x85, 0xaf, 0x21, 0x99, 0x49, 0x4a, 0xed, 0x26, 0x73, 0xfe, 0xff,
	0xf3, 0x91, 0x1f, 0xe7, 0x40, 0x22, 0x55, 0xb2, 0xa0, 0x4a, 0xba, 0xc5, 0xcb, 0xa3, 0xc9, 0x42,
	0x72, 0x11, 0x39, 0x71, 0x22, 0x94, 0x40, 0x8d, 0xdc, 0x77, 0xf2, 0xd7, 0x3e, 0xa2, 0x42, 0x86,
	0x42, 0xde, 0xe8, 0xb4, 0x6b, 0x84, 0xa9, 0xb5, 0x9b, 0x53, 0x31, 0x15, 0xc6, 0xcf, 0xa2, 0xdc,
	0xb5, 0x0f, 0xff, 0x30, 0x63, 0x69, 0xd1, 0xf1, 0xcf, 0x0f, 0x79, 0x24, 0x5c, 0xfd, 0x35, 0xd6,
	0xf1, 0x6b, 0x09, 0xd6, 0x87, 0x39, 0x03, 0x1a, 0xc1, 0x46, 0xc0, 0xa4, 0xe2, 0x91, 0xaf, 0xb8,
	0x88, 0xae, 0xd2, 0x98, 0x61, 0xd0, 0x01, 0xdd, 0xbf, 0xe7, 0x2d, 0xe7, 0x80, 0xcb, 0x11, 0xe3,
	0x3b, 0x46, 0x55, 0x56, 0xd2, 0xaf, 0x3c, 0x7d, 0x3d, 0x9f, 0x02, 0xef, 0xb0, 0x15, 0x9d, 0xc0,
	0x3f, 0x7b, 0xd6, 0x30, 0xc0, 0xa5, 0x0e, 0xe8, 0xfe, 0xf6, 0x7e, 0x9a, 0x08, 0xc1, 0xf2, 0x64,
	0xc1, 0xe6, 0xf8, 0x57, 0x07, 0x74, 0xcb, 0x9e, 0x8e, 0x51, 0x13, 0x56, 0x62, 0x71, 0xcf, 0x12,
	0x5c, 0xd6, 0xa6, 0x11, 0xe8, 0x12, 0x42, 0x2a, 0xc2, 0x90, 0xcb, 0x8c, 0x15, 0x57, 0xb2, 0x61,
	0xfd, 0xde, 0x72, 0xdd, 0xb6, 0x3e, 0xd6, 0xed, 0x96, 0xd9, 0x8c, 0x0c, 0x66, 0x0e, 0x17, 0x6e,
	0xe8, 0xab, 0x5b, 0x67, 0xc4, 0xa6, 0x3e, 0x4d, 0x07, 0x8c, 0xbe, 0xbd, 0x9c, 0xc1, 0x7c, 0x71,
	0x03, 0x46, 0xbd, 0xbd, 0x21, 0xc8, 0x86, 0xf5, 0x78, 0xee, 0xa7, 0x2c, 0x19, 0x06, 0xb8, 0xaa,
	0xe9, 0x76, 0x1a, 0x61, 0x58, 0xf3, 0x83, 0x20, 0x61, 0x52, 0xe2, 0x9a, 0x4e, 0x15, 0x32, 0xc3,
	0x4b, 0x32, 0x7c, 0x5c, 0x37, 0x78, 0x5a, 0xf4, 0x7b, 0xcb, 0x0d, 0x01, 0xab, 0x0d, 0x01, 0x9f,
	0x1b, 0x02, 0x1e, 0xb7, 0xc4, 0x5a, 0x6d, 0x89, 0xf5, 0xbe, 0x25, 0xd6, 0xf5, 0xff, 0xe2, 0x14,
	0x0f, 0xbb, 0xa3, 0xa8, 0x34, 0x66, 0x72, 0x5c, 0xd5, 0x37, 0xb8, 0xf8, 0x0e, 0x00, 0x00, 0xff,
	0xff, 0x0b, 0x5b, 0x2c, 0x1c, 0x16, 0x02, 0x00, 0x00,
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
	if m.Ratio != 0 {
		i = encodeVarintInfusion(dAtA, i, uint64(m.Ratio))
		i--
		dAtA[i] = 0x40
	}
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintInfusion(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0x3a
	}
	if len(m.PlayerId) > 0 {
		i -= len(m.PlayerId)
		copy(dAtA[i:], m.PlayerId)
		i = encodeVarintInfusion(dAtA, i, uint64(len(m.PlayerId)))
		i--
		dAtA[i] = 0x32
	}
	{
		size := m.Commission.Size()
		i -= size
		if _, err := m.Commission.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintInfusion(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x2a
	if m.Power != 0 {
		i = encodeVarintInfusion(dAtA, i, uint64(m.Power))
		i--
		dAtA[i] = 0x20
	}
	if m.Fuel != 0 {
		i = encodeVarintInfusion(dAtA, i, uint64(m.Fuel))
		i--
		dAtA[i] = 0x18
	}
	if len(m.DestinationId) > 0 {
		i -= len(m.DestinationId)
		copy(dAtA[i:], m.DestinationId)
		i = encodeVarintInfusion(dAtA, i, uint64(len(m.DestinationId)))
		i--
		dAtA[i] = 0x12
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
	l = len(m.DestinationId)
	if l > 0 {
		n += 1 + l + sovInfusion(uint64(l))
	}
	if m.Fuel != 0 {
		n += 1 + sovInfusion(uint64(m.Fuel))
	}
	if m.Power != 0 {
		n += 1 + sovInfusion(uint64(m.Power))
	}
	l = m.Commission.Size()
	n += 1 + l + sovInfusion(uint64(l))
	l = len(m.PlayerId)
	if l > 0 {
		n += 1 + l + sovInfusion(uint64(l))
	}
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovInfusion(uint64(l))
	}
	if m.Ratio != 0 {
		n += 1 + sovInfusion(uint64(m.Ratio))
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
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DestinationId", wireType)
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
			m.DestinationId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
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
				return fmt.Errorf("proto: wrong wireType = %d for field Power", wireType)
			}
			m.Power = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowInfusion
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
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Commission", wireType)
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
			if err := m.Commission.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PlayerId", wireType)
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
			m.PlayerId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
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
		case 8:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Ratio", wireType)
			}
			m.Ratio = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowInfusion
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Ratio |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
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
