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

func init() {
	proto.RegisterType((*GridRecord)(nil), "structs.GridRecord")
}

func init() { proto.RegisterFile("structs/structs/grid.proto", fileDescriptor_e87ac0223f374538) }

var fileDescriptor_e87ac0223f374538 = []byte{
	// 163 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x2a, 0x2e, 0x29, 0x2a,
	0x4d, 0x2e, 0x29, 0xd6, 0x87, 0xd1, 0xe9, 0x45, 0x99, 0x29, 0x7a, 0x05, 0x45, 0xf9, 0x25, 0xf9,
	0x42, 0xec, 0x50, 0x31, 0x29, 0x91, 0xf4, 0xfc, 0xf4, 0x7c, 0xb0, 0x98, 0x3e, 0x88, 0x05, 0x91,
	0x56, 0x72, 0xe1, 0xe2, 0x72, 0x2f, 0xca, 0x4c, 0x09, 0x4a, 0x4d, 0xce, 0x2f, 0x4a, 0x11, 0x52,
	0xe0, 0xe2, 0x4e, 0x2c, 0x29, 0x29, 0xca, 0x4c, 0x2a, 0x2d, 0x49, 0xf5, 0x4c, 0x91, 0x60, 0x54,
	0x60, 0xd4, 0xe0, 0x0c, 0x42, 0x16, 0x12, 0x12, 0xe1, 0x62, 0x2d, 0x4b, 0xcc, 0x29, 0x4d, 0x95,
	0x60, 0x52, 0x60, 0xd4, 0x60, 0x09, 0x82, 0x70, 0x9c, 0x0c, 0x4f, 0x3c, 0x92, 0x63, 0xbc, 0xf0,
	0x48, 0x8e, 0xf1, 0xc1, 0x23, 0x39, 0xc6, 0x09, 0x8f, 0xe5, 0x18, 0x2e, 0x3c, 0x96, 0x63, 0xb8,
	0xf1, 0x58, 0x8e, 0x21, 0x4a, 0x1c, 0xe6, 0xa4, 0x0a, 0xb8, 0xe3, 0x4a, 0x2a, 0x0b, 0x52, 0x8b,
	0x93, 0xd8, 0xc0, 0xf6, 0x1b, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0x21, 0xcb, 0x0e, 0x92, 0xbc,
	0x00, 0x00, 0x00,
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
