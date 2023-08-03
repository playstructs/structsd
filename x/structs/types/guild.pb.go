// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: structs/structs/guild.proto

package types

import (
	fmt "fmt"
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

type Guild struct {
	Id       uint64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Endpoint string `protobuf:"bytes,2,opt,name=endpoint,proto3" json:"endpoint,omitempty"`
	Creator  string `protobuf:"bytes,3,opt,name=creator,proto3" json:"creator,omitempty"`
	Owner    uint64 `protobuf:"varint,4,opt,name=owner,proto3" json:"owner,omitempty"`
}

func (m *Guild) Reset()         { *m = Guild{} }
func (m *Guild) String() string { return proto.CompactTextString(m) }
func (*Guild) ProtoMessage()    {}
func (*Guild) Descriptor() ([]byte, []int) {
	return fileDescriptor_d57f2d07301e2fa1, []int{0}
}
func (m *Guild) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Guild) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Guild.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Guild) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Guild.Merge(m, src)
}
func (m *Guild) XXX_Size() int {
	return m.Size()
}
func (m *Guild) XXX_DiscardUnknown() {
	xxx_messageInfo_Guild.DiscardUnknown(m)
}

var xxx_messageInfo_Guild proto.InternalMessageInfo

func (m *Guild) GetId() uint64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *Guild) GetEndpoint() string {
	if m != nil {
		return m.Endpoint
	}
	return ""
}

func (m *Guild) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

func (m *Guild) GetOwner() uint64 {
	if m != nil {
		return m.Owner
	}
	return 0
}

func init() {
	proto.RegisterType((*Guild)(nil), "structs.structs.Guild")
}

func init() { proto.RegisterFile("structs/structs/guild.proto", fileDescriptor_d57f2d07301e2fa1) }

var fileDescriptor_d57f2d07301e2fa1 = []byte{
	// 174 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x2e, 0x2e, 0x29, 0x2a,
	0x4d, 0x2e, 0x29, 0xd6, 0x87, 0xd1, 0xe9, 0xa5, 0x99, 0x39, 0x29, 0x7a, 0x05, 0x45, 0xf9, 0x25,
	0xf9, 0x42, 0xfc, 0x50, 0x41, 0x3d, 0x28, 0xad, 0x94, 0xcc, 0xc5, 0xea, 0x0e, 0x92, 0x17, 0xe2,
	0xe3, 0x62, 0xca, 0x4c, 0x91, 0x60, 0x54, 0x60, 0xd4, 0x60, 0x09, 0x62, 0xca, 0x4c, 0x11, 0x92,
	0xe2, 0xe2, 0x48, 0xcd, 0x4b, 0x29, 0xc8, 0xcf, 0xcc, 0x2b, 0x91, 0x60, 0x52, 0x60, 0xd4, 0xe0,
	0x0c, 0x82, 0xf3, 0x85, 0x24, 0xb8, 0xd8, 0x93, 0x8b, 0x52, 0x13, 0x4b, 0xf2, 0x8b, 0x24, 0x98,
	0xc1, 0x52, 0x30, 0xae, 0x90, 0x08, 0x17, 0x6b, 0x7e, 0x79, 0x5e, 0x6a, 0x91, 0x04, 0x0b, 0xd8,
	0x20, 0x08, 0xc7, 0xc9, 0xf0, 0xc4, 0x23, 0x39, 0xc6, 0x0b, 0x8f, 0xe4, 0x18, 0x1f, 0x3c, 0x92,
	0x63, 0x9c, 0xf0, 0x58, 0x8e, 0xe1, 0xc2, 0x63, 0x39, 0x86, 0x1b, 0x8f, 0xe5, 0x18, 0xa2, 0xc4,
	0x61, 0x8e, 0xac, 0x80, 0x3b, 0xb7, 0xa4, 0xb2, 0x20, 0xb5, 0x38, 0x89, 0x0d, 0xec, 0x5e, 0x63,
	0x40, 0x00, 0x00, 0x00, 0xff, 0xff, 0xda, 0xa0, 0x86, 0x09, 0xce, 0x00, 0x00, 0x00,
}

func (m *Guild) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Guild) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Guild) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Owner != 0 {
		i = encodeVarintGuild(dAtA, i, uint64(m.Owner))
		i--
		dAtA[i] = 0x20
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintGuild(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Endpoint) > 0 {
		i -= len(m.Endpoint)
		copy(dAtA[i:], m.Endpoint)
		i = encodeVarintGuild(dAtA, i, uint64(len(m.Endpoint)))
		i--
		dAtA[i] = 0x12
	}
	if m.Id != 0 {
		i = encodeVarintGuild(dAtA, i, uint64(m.Id))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintGuild(dAtA []byte, offset int, v uint64) int {
	offset -= sovGuild(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Guild) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Id != 0 {
		n += 1 + sovGuild(uint64(m.Id))
	}
	l = len(m.Endpoint)
	if l > 0 {
		n += 1 + l + sovGuild(uint64(l))
	}
	l = len(m.Creator)
	if l > 0 {
		n += 1 + l + sovGuild(uint64(l))
	}
	if m.Owner != 0 {
		n += 1 + sovGuild(uint64(m.Owner))
	}
	return n
}

func sovGuild(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozGuild(x uint64) (n int) {
	return sovGuild(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Guild) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGuild
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
			return fmt.Errorf("proto: Guild: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Guild: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			m.Id = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGuild
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Id |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Endpoint", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGuild
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
				return ErrInvalidLengthGuild
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGuild
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Endpoint = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Creator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGuild
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
				return ErrInvalidLengthGuild
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGuild
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Creator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Owner", wireType)
			}
			m.Owner = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGuild
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Owner |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipGuild(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGuild
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
func skipGuild(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowGuild
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
					return 0, ErrIntOverflowGuild
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
					return 0, ErrIntOverflowGuild
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
				return 0, ErrInvalidLengthGuild
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupGuild
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthGuild
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthGuild        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowGuild          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupGuild = fmt.Errorf("proto: unexpected end of group")
)