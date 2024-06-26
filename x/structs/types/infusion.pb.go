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
	Defusing        cosmossdk_io_math.Int       `protobuf:"bytes,9,opt,name=defusing,proto3,customtype=cosmossdk.io/math.Int" json:"defusing"`
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
	// 387 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x64, 0x91, 0xcd, 0x4a, 0xeb, 0x40,
	0x14, 0xc7, 0x33, 0xb7, 0x5f, 0xe9, 0xc0, 0xbd, 0xe5, 0x0e, 0x15, 0xc7, 0x14, 0xd2, 0x22, 0x2e,
	0x8a, 0x62, 0x42, 0xf5, 0x0d, 0x4a, 0x41, 0x02, 0xdd, 0x18, 0x5c, 0xb9, 0x91, 0x34, 0x99, 0xc6,
	0xd8, 0x66, 0x26, 0x64, 0xa6, 0x68, 0xde, 0xa2, 0x8f, 0xe1, 0xd2, 0x85, 0x0f, 0xd1, 0x65, 0x71,
	0x25, 0x2e, 0x8a, 0xb4, 0x0b, 0x5f, 0x43, 0x32, 0x49, 0x4a, 0xad, 0x9b, 0xcc, 0xf9, 0xff, 0xcf,
	0x47, 0x7e, 0x9c, 0x03, 0x75, 0x2e, 0xe2, 0x99, 0x2b, 0xb8, 0x59, 0xbc, 0x01, 0x1d, 0xcf, 0x78,
	0xc0, 0xa8, 0x11, 0xc5, 0x4c, 0x30, 0xd4, 0xc8, 0x7d, 0x23, 0x7f, 0xb5, 0x23, 0x97, 0xf1, 0x90,
	0xf1, 0x3b, 0x99, 0x36, 0x33, 0x91, 0xd5, 0x6a, 0x4d, 0x9f, 0xf9, 0x2c, 0xf3, 0xd3, 0x28, 0x77,
	0xb5, 0xfd, 0x3f, 0x4c, 0x48, 0x52, 0x74, 0xfc, 0x77, 0xc2, 0x80, 0x32, 0x53, 0x7e, 0x33, 0xeb,
	0x78, 0x5e, 0x82, 0xaa, 0x95, 0x33, 0xa0, 0x21, 0x6c, 0x78, 0x84, 0x8b, 0x80, 0x3a, 0x22, 0x60,
	0xf4, 0x26, 0x89, 0x08, 0x06, 0x1d, 0xd0, 0xfd, 0x77, 0xd1, 0x32, 0xf6, 0xb8, 0x0c, 0x36, 0x7a,
	0x20, 0xae, 0x48, 0x4b, 0xfa, 0x95, 0xe7, 0xaf, 0x97, 0x53, 0x60, 0xef, 0xb7, 0xa2, 0x13, 0xf8,
	0x77, 0xc7, 0xb2, 0x3c, 0xfc, 0xa7, 0x03, 0xba, 0x75, 0xfb, 0xa7, 0x89, 0x10, 0x2c, 0x8f, 0x67,
	0x64, 0x8a, 0x4b, 0x1d, 0xd0, 0x2d, 0xdb, 0x32, 0x46, 0x4d, 0x58, 0x89, 0xd8, 0x23, 0x89, 0x71,
	0x59, 0x9a, 0x99, 0x40, 0xd7, 0x10, 0xba, 0x2c, 0x0c, 0x03, 0x9e, 0xb2, 0xe2, 0x4a, 0x3a, 0xac,
	0xdf, 0x5b, 0xac, 0xda, 0xca, 0xc7, 0xaa, 0xdd, 0xca, 0x36, 0xc3, 0xbd, 0x89, 0x11, 0x30, 0x33,
	0x74, 0xc4, 0xbd, 0x31, 0x24, 0xbe, 0xe3, 0x26, 0x03, 0xe2, 0xbe, 0xbd, 0x9e, 0xc3, 0x7c, 0x71,
	0x03, 0xe2, 0xda, 0x3b, 0x43, 0x90, 0x06, 0xd5, 0x68, 0xea, 0x24, 0x24, 0xb6, 0x3c, 0x5c, 0x95,
	0x74, 0x5b, 0x8d, 0x30, 0xac, 0x39, 0x9e, 0x17, 0x13, 0xce, 0x71, 0x4d, 0xa6, 0x0a, 0x99, 0xe2,
	0xc5, 0x29, 0x3e, 0x56, 0x33, 0x3c, 0x29, 0xd0, 0x15, 0x54, 0x3d, 0x92, 0x2e, 0x92, 0xfa, 0xb8,
	0x2e, 0xe1, 0xce, 0x72, 0xb8, 0x83, 0xdf, 0x70, 0x16, 0x15, 0x3b, 0x58, 0x16, 0x15, 0xf6, 0xb6,
	0xb9, 0xdf, 0x5b, 0xac, 0x75, 0xb0, 0x5c, 0xeb, 0xe0, 0x73, 0xad, 0x83, 0xf9, 0x46, 0x57, 0x96,
	0x1b, 0x5d, 0x79, 0xdf, 0xe8, 0xca, 0xed, 0x61, 0x71, 0xd3, 0xa7, 0xed, 0x75, 0x45, 0x12, 0x11,
	0x3e, 0xaa, 0xca, 0x63, 0x5e, 0x7e, 0x07, 0x00, 0x00, 0xff, 0xff, 0x02, 0x4c, 0x73, 0xe0, 0x5f,
	0x02, 0x00, 0x00,
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
	{
		size := m.Defusing.Size()
		i -= size
		if _, err := m.Defusing.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintInfusion(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x4a
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
	l = m.Defusing.Size()
	n += 1 + l + sovInfusion(uint64(l))
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
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Defusing", wireType)
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
			if err := m.Defusing.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
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
