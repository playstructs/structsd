// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: structs/structs/substation.proto

package types

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
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

type Substation struct {
	Id                         uint64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	PlayerConnectionAllocation uint64 `protobuf:"varint,2,opt,name=playerConnectionAllocation,proto3" json:"playerConnectionAllocation,omitempty"`
	Owner                      uint64 `protobuf:"varint,3,opt,name=owner,proto3" json:"owner,omitempty"`
	Creator                    string `protobuf:"bytes,4,opt,name=creator,proto3" json:"creator,omitempty"`
	Load                       uint64 `protobuf:"varint,5,opt,name=load,proto3" json:"load,omitempty"`
	Energy                     uint64 `protobuf:"varint,6,opt,name=energy,proto3" json:"energy,omitempty"`
	ConnectedPlayerCount       uint64 `protobuf:"varint,7,opt,name=connectedPlayerCount,proto3" json:"connectedPlayerCount,omitempty"`
}

func (m *Substation) Reset()         { *m = Substation{} }
func (m *Substation) String() string { return proto.CompactTextString(m) }
func (*Substation) ProtoMessage()    {}
func (*Substation) Descriptor() ([]byte, []int) {
	return fileDescriptor_1dfac9318fba59fb, []int{0}
}
func (m *Substation) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Substation) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Substation.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Substation) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Substation.Merge(m, src)
}
func (m *Substation) XXX_Size() int {
	return m.Size()
}
func (m *Substation) XXX_DiscardUnknown() {
	xxx_messageInfo_Substation.DiscardUnknown(m)
}

var xxx_messageInfo_Substation proto.InternalMessageInfo

func (m *Substation) GetId() uint64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *Substation) GetPlayerConnectionAllocation() uint64 {
	if m != nil {
		return m.PlayerConnectionAllocation
	}
	return 0
}

func (m *Substation) GetOwner() uint64 {
	if m != nil {
		return m.Owner
	}
	return 0
}

func (m *Substation) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

func (m *Substation) GetLoad() uint64 {
	if m != nil {
		return m.Load
	}
	return 0
}

func (m *Substation) GetEnergy() uint64 {
	if m != nil {
		return m.Energy
	}
	return 0
}

func (m *Substation) GetConnectedPlayerCount() uint64 {
	if m != nil {
		return m.ConnectedPlayerCount
	}
	return 0
}

func init() {
	proto.RegisterType((*Substation)(nil), "structs.structs.Substation")
}

func init() { proto.RegisterFile("structs/structs/substation.proto", fileDescriptor_1dfac9318fba59fb) }

var fileDescriptor_1dfac9318fba59fb = []byte{
	// 263 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0x28, 0x2e, 0x29, 0x2a,
	0x4d, 0x2e, 0x29, 0xd6, 0x87, 0xd3, 0xa5, 0x49, 0xc5, 0x25, 0x89, 0x25, 0x99, 0xf9, 0x79, 0x7a,
	0x05, 0x45, 0xf9, 0x25, 0xf9, 0x42, 0xfc, 0x50, 0x19, 0x3d, 0x28, 0x2d, 0x25, 0x92, 0x9e, 0x9f,
	0x9e, 0x0f, 0x96, 0xd3, 0x07, 0xb1, 0x20, 0xca, 0xa4, 0x30, 0x0c, 0x4a, 0xcc, 0xc9, 0xc9, 0x4f,
	0x46, 0x32, 0x48, 0xe9, 0x0d, 0x23, 0x17, 0x57, 0x30, 0xdc, 0x74, 0x21, 0x3e, 0x2e, 0xa6, 0xcc,
	0x14, 0x09, 0x46, 0x05, 0x46, 0x0d, 0x96, 0x20, 0xa6, 0xcc, 0x14, 0x21, 0x3b, 0x2e, 0xa9, 0x82,
	0x9c, 0xc4, 0xca, 0xd4, 0x22, 0xe7, 0xfc, 0xbc, 0xbc, 0xd4, 0x64, 0x90, 0x1a, 0x47, 0xb8, 0x11,
	0x12, 0x4c, 0x60, 0x75, 0x78, 0x54, 0x08, 0x89, 0x70, 0xb1, 0xe6, 0x97, 0xe7, 0xa5, 0x16, 0x49,
	0x30, 0x83, 0x95, 0x42, 0x38, 0x42, 0x12, 0x5c, 0xec, 0xc9, 0x45, 0xa9, 0x89, 0x25, 0xf9, 0x45,
	0x12, 0x2c, 0x0a, 0x8c, 0x1a, 0x9c, 0x41, 0x30, 0xae, 0x90, 0x10, 0x17, 0x4b, 0x4e, 0x7e, 0x62,
	0x8a, 0x04, 0x2b, 0x58, 0x39, 0x98, 0x2d, 0x24, 0xc6, 0xc5, 0x96, 0x9a, 0x97, 0x5a, 0x94, 0x5e,
	0x29, 0xc1, 0x06, 0x16, 0x85, 0xf2, 0x84, 0x8c, 0xb8, 0x44, 0x92, 0x21, 0x76, 0xa6, 0xa6, 0x04,
	0x40, 0x9d, 0x50, 0x9a, 0x57, 0x22, 0xc1, 0x0e, 0x56, 0x85, 0x55, 0xce, 0xc9, 0xf0, 0xc4, 0x23,
	0x39, 0xc6, 0x0b, 0x8f, 0xe4, 0x18, 0x1f, 0x3c, 0x92, 0x63, 0x9c, 0xf0, 0x58, 0x8e, 0xe1, 0xc2,
	0x63, 0x39, 0x86, 0x1b, 0x8f, 0xe5, 0x18, 0xa2, 0xc4, 0x61, 0x41, 0x54, 0x01, 0x0f, 0xac, 0x92,
	0xca, 0x82, 0xd4, 0xe2, 0x24, 0x36, 0x70, 0x40, 0x19, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0xff,
	0x1c, 0x20, 0x70, 0x95, 0x01, 0x00, 0x00,
}

func (m *Substation) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Substation) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Substation) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.ConnectedPlayerCount != 0 {
		i = encodeVarintSubstation(dAtA, i, uint64(m.ConnectedPlayerCount))
		i--
		dAtA[i] = 0x38
	}
	if m.Energy != 0 {
		i = encodeVarintSubstation(dAtA, i, uint64(m.Energy))
		i--
		dAtA[i] = 0x30
	}
	if m.Load != 0 {
		i = encodeVarintSubstation(dAtA, i, uint64(m.Load))
		i--
		dAtA[i] = 0x28
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintSubstation(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x22
	}
	if m.Owner != 0 {
		i = encodeVarintSubstation(dAtA, i, uint64(m.Owner))
		i--
		dAtA[i] = 0x18
	}
	if m.PlayerConnectionAllocation != 0 {
		i = encodeVarintSubstation(dAtA, i, uint64(m.PlayerConnectionAllocation))
		i--
		dAtA[i] = 0x10
	}
	if m.Id != 0 {
		i = encodeVarintSubstation(dAtA, i, uint64(m.Id))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintSubstation(dAtA []byte, offset int, v uint64) int {
	offset -= sovSubstation(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Substation) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Id != 0 {
		n += 1 + sovSubstation(uint64(m.Id))
	}
	if m.PlayerConnectionAllocation != 0 {
		n += 1 + sovSubstation(uint64(m.PlayerConnectionAllocation))
	}
	if m.Owner != 0 {
		n += 1 + sovSubstation(uint64(m.Owner))
	}
	l = len(m.Creator)
	if l > 0 {
		n += 1 + l + sovSubstation(uint64(l))
	}
	if m.Load != 0 {
		n += 1 + sovSubstation(uint64(m.Load))
	}
	if m.Energy != 0 {
		n += 1 + sovSubstation(uint64(m.Energy))
	}
	if m.ConnectedPlayerCount != 0 {
		n += 1 + sovSubstation(uint64(m.ConnectedPlayerCount))
	}
	return n
}

func sovSubstation(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozSubstation(x uint64) (n int) {
	return sovSubstation(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Substation) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSubstation
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
			return fmt.Errorf("proto: Substation: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Substation: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			m.Id = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSubstation
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
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field PlayerConnectionAllocation", wireType)
			}
			m.PlayerConnectionAllocation = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSubstation
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.PlayerConnectionAllocation |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Owner", wireType)
			}
			m.Owner = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSubstation
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
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Creator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSubstation
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
				return ErrInvalidLengthSubstation
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthSubstation
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Creator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Load", wireType)
			}
			m.Load = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSubstation
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
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Energy", wireType)
			}
			m.Energy = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSubstation
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
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ConnectedPlayerCount", wireType)
			}
			m.ConnectedPlayerCount = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSubstation
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ConnectedPlayerCount |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipSubstation(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSubstation
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
func skipSubstation(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowSubstation
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
					return 0, ErrIntOverflowSubstation
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
					return 0, ErrIntOverflowSubstation
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
				return 0, ErrInvalidLengthSubstation
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupSubstation
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthSubstation
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthSubstation        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowSubstation          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupSubstation = fmt.Errorf("proto: unexpected end of group")
)
