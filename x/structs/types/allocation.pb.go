// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: structs/structs/allocation.proto

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

type Allocation struct {
	Id            uint64     `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	SourceType    ObjectType `protobuf:"varint,2,opt,name=sourceType,proto3,enum=structs.structs.ObjectType" json:"sourceType,omitempty"`
	SourceId      uint64     `protobuf:"varint,3,opt,name=sourceId,proto3" json:"sourceId,omitempty"`
	DestinationId uint64     `protobuf:"varint,4,opt,name=destinationId,proto3" json:"destinationId,omitempty"`
	Power         uint64     `protobuf:"varint,5,opt,name=power,proto3" json:"power,omitempty"`
	Creator       string     `protobuf:"bytes,6,opt,name=creator,proto3" json:"creator,omitempty"`
	Owner         string     `protobuf:"bytes,7,opt,name=owner,proto3" json:"owner,omitempty"`
	Locked        bool       `protobuf:"varint,8,opt,name=locked,proto3" json:"locked,omitempty"`
}

func (m *Allocation) Reset()         { *m = Allocation{} }
func (m *Allocation) String() string { return proto.CompactTextString(m) }
func (*Allocation) ProtoMessage()    {}
func (*Allocation) Descriptor() ([]byte, []int) {
	return fileDescriptor_5b374468bf8d3c09, []int{0}
}
func (m *Allocation) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Allocation) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Allocation.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Allocation) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Allocation.Merge(m, src)
}
func (m *Allocation) XXX_Size() int {
	return m.Size()
}
func (m *Allocation) XXX_DiscardUnknown() {
	xxx_messageInfo_Allocation.DiscardUnknown(m)
}

var xxx_messageInfo_Allocation proto.InternalMessageInfo

func (m *Allocation) GetId() uint64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *Allocation) GetSourceType() ObjectType {
	if m != nil {
		return m.SourceType
	}
	return ObjectType_faction
}

func (m *Allocation) GetSourceId() uint64 {
	if m != nil {
		return m.SourceId
	}
	return 0
}

func (m *Allocation) GetDestinationId() uint64 {
	if m != nil {
		return m.DestinationId
	}
	return 0
}

func (m *Allocation) GetPower() uint64 {
	if m != nil {
		return m.Power
	}
	return 0
}

func (m *Allocation) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

func (m *Allocation) GetOwner() string {
	if m != nil {
		return m.Owner
	}
	return ""
}

func (m *Allocation) GetLocked() bool {
	if m != nil {
		return m.Locked
	}
	return false
}

type AllocationPackage struct {
	Allocation *Allocation `protobuf:"bytes,1,opt,name=allocation,proto3" json:"allocation,omitempty"`
	Status     uint64      `protobuf:"varint,2,opt,name=status,proto3" json:"status,omitempty"`
}

func (m *AllocationPackage) Reset()         { *m = AllocationPackage{} }
func (m *AllocationPackage) String() string { return proto.CompactTextString(m) }
func (*AllocationPackage) ProtoMessage()    {}
func (*AllocationPackage) Descriptor() ([]byte, []int) {
	return fileDescriptor_5b374468bf8d3c09, []int{1}
}
func (m *AllocationPackage) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *AllocationPackage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_AllocationPackage.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *AllocationPackage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AllocationPackage.Merge(m, src)
}
func (m *AllocationPackage) XXX_Size() int {
	return m.Size()
}
func (m *AllocationPackage) XXX_DiscardUnknown() {
	xxx_messageInfo_AllocationPackage.DiscardUnknown(m)
}

var xxx_messageInfo_AllocationPackage proto.InternalMessageInfo

func (m *AllocationPackage) GetAllocation() *Allocation {
	if m != nil {
		return m.Allocation
	}
	return nil
}

func (m *AllocationPackage) GetStatus() uint64 {
	if m != nil {
		return m.Status
	}
	return 0
}

func init() {
	proto.RegisterType((*Allocation)(nil), "structs.structs.Allocation")
	proto.RegisterType((*AllocationPackage)(nil), "structs.structs.AllocationPackage")
}

func init() { proto.RegisterFile("structs/structs/allocation.proto", fileDescriptor_5b374468bf8d3c09) }

var fileDescriptor_5b374468bf8d3c09 = []byte{
	// 321 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x91, 0xbf, 0x4e, 0xc3, 0x30,
	0x10, 0xc6, 0xeb, 0xd2, 0x7f, 0x1c, 0xa2, 0x08, 0xab, 0x02, 0x2b, 0x48, 0x51, 0x54, 0x31, 0x64,
	0x4a, 0x05, 0x8c, 0x4c, 0xb0, 0x75, 0x43, 0x16, 0x13, 0x9b, 0x6b, 0x5b, 0x25, 0xb4, 0xea, 0x45,
	0xb6, 0xab, 0xd2, 0xb7, 0xe0, 0xb1, 0x18, 0x3b, 0x32, 0xa2, 0xf6, 0x29, 0xd8, 0x50, 0x9c, 0xb4,
	0x29, 0x15, 0xd3, 0xf9, 0x77, 0xdf, 0xe7, 0x3b, 0xdd, 0x1d, 0x44, 0xd6, 0x99, 0xb9, 0x74, 0x76,
	0xb0, 0x8d, 0x62, 0x3a, 0x45, 0x29, 0x5c, 0x8a, 0xb3, 0x24, 0x33, 0xe8, 0x90, 0x9e, 0x95, 0x4a,
	0x52, 0xc6, 0xa0, 0x37, 0xc6, 0x31, 0x7a, 0x6d, 0x90, 0xbf, 0x0a, 0x5b, 0x10, 0x1c, 0x16, 0x9a,
	0xe8, 0xa5, 0x2d, 0xb4, 0xfe, 0x0f, 0x01, 0x78, 0xd8, 0xd5, 0xa5, 0x5d, 0xa8, 0xa7, 0x8a, 0x91,
	0x88, 0xc4, 0x0d, 0x5e, 0x4f, 0x15, 0xbd, 0x07, 0xb0, 0x38, 0x37, 0x52, 0x3f, 0x2f, 0x33, 0xcd,
	0xea, 0x11, 0x89, 0xbb, 0xb7, 0x57, 0xc9, 0x41, 0xdb, 0x04, 0x47, 0x6f, 0x5a, 0xba, 0xdc, 0xc2,
	0xf7, 0xec, 0x34, 0x80, 0x4e, 0x41, 0x43, 0xc5, 0x8e, 0x7c, 0xc9, 0x1d, 0xd3, 0x6b, 0x38, 0x55,
	0xda, 0xba, 0x74, 0xe6, 0xfb, 0x0e, 0x15, 0x6b, 0x78, 0xc3, 0xdf, 0x24, 0xed, 0x41, 0x33, 0xc3,
	0x85, 0x36, 0xac, 0xe9, 0xd5, 0x02, 0x28, 0x83, 0xb6, 0x34, 0x5a, 0x38, 0x34, 0xac, 0x15, 0x91,
	0xf8, 0x98, 0x6f, 0x31, 0xf7, 0xe3, 0x62, 0xa6, 0x0d, 0x6b, 0xfb, 0x7c, 0x01, 0xf4, 0x02, 0x5a,
	0x53, 0x94, 0x13, 0xad, 0x58, 0x27, 0x22, 0x71, 0x87, 0x97, 0xd4, 0x7f, 0x85, 0xf3, 0x6a, 0xf4,
	0x27, 0x21, 0x27, 0x62, 0xac, 0xf3, 0x89, 0xab, 0x3d, 0xfb, 0x4d, 0x9c, 0xfc, 0x33, 0x71, 0xf5,
	0x8f, 0xef, 0xd9, 0xf3, 0x4e, 0xd6, 0x09, 0x37, 0xb7, 0x7e, 0x55, 0x0d, 0x5e, 0xd2, 0xe3, 0xcd,
	0xe7, 0x3a, 0x24, 0xab, 0x75, 0x48, 0xbe, 0xd7, 0x21, 0xf9, 0xd8, 0x84, 0xb5, 0xd5, 0x26, 0xac,
	0x7d, 0x6d, 0xc2, 0xda, 0xcb, 0xe5, 0xf6, 0x26, 0xef, 0xbb, 0xeb, 0xb8, 0x65, 0xa6, 0xed, 0xa8,
	0xe5, 0xef, 0x73, 0xf7, 0x1b, 0x00, 0x00, 0xff, 0xff, 0x92, 0xac, 0x45, 0xae, 0x06, 0x02, 0x00,
	0x00,
}

func (m *Allocation) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Allocation) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Allocation) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Locked {
		i--
		if m.Locked {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x40
	}
	if len(m.Owner) > 0 {
		i -= len(m.Owner)
		copy(dAtA[i:], m.Owner)
		i = encodeVarintAllocation(dAtA, i, uint64(len(m.Owner)))
		i--
		dAtA[i] = 0x3a
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintAllocation(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x32
	}
	if m.Power != 0 {
		i = encodeVarintAllocation(dAtA, i, uint64(m.Power))
		i--
		dAtA[i] = 0x28
	}
	if m.DestinationId != 0 {
		i = encodeVarintAllocation(dAtA, i, uint64(m.DestinationId))
		i--
		dAtA[i] = 0x20
	}
	if m.SourceId != 0 {
		i = encodeVarintAllocation(dAtA, i, uint64(m.SourceId))
		i--
		dAtA[i] = 0x18
	}
	if m.SourceType != 0 {
		i = encodeVarintAllocation(dAtA, i, uint64(m.SourceType))
		i--
		dAtA[i] = 0x10
	}
	if m.Id != 0 {
		i = encodeVarintAllocation(dAtA, i, uint64(m.Id))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *AllocationPackage) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *AllocationPackage) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *AllocationPackage) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Status != 0 {
		i = encodeVarintAllocation(dAtA, i, uint64(m.Status))
		i--
		dAtA[i] = 0x10
	}
	if m.Allocation != nil {
		{
			size, err := m.Allocation.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintAllocation(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintAllocation(dAtA []byte, offset int, v uint64) int {
	offset -= sovAllocation(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Allocation) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Id != 0 {
		n += 1 + sovAllocation(uint64(m.Id))
	}
	if m.SourceType != 0 {
		n += 1 + sovAllocation(uint64(m.SourceType))
	}
	if m.SourceId != 0 {
		n += 1 + sovAllocation(uint64(m.SourceId))
	}
	if m.DestinationId != 0 {
		n += 1 + sovAllocation(uint64(m.DestinationId))
	}
	if m.Power != 0 {
		n += 1 + sovAllocation(uint64(m.Power))
	}
	l = len(m.Creator)
	if l > 0 {
		n += 1 + l + sovAllocation(uint64(l))
	}
	l = len(m.Owner)
	if l > 0 {
		n += 1 + l + sovAllocation(uint64(l))
	}
	if m.Locked {
		n += 2
	}
	return n
}

func (m *AllocationPackage) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Allocation != nil {
		l = m.Allocation.Size()
		n += 1 + l + sovAllocation(uint64(l))
	}
	if m.Status != 0 {
		n += 1 + sovAllocation(uint64(m.Status))
	}
	return n
}

func sovAllocation(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozAllocation(x uint64) (n int) {
	return sovAllocation(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Allocation) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAllocation
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
			return fmt.Errorf("proto: Allocation: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Allocation: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			m.Id = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAllocation
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
				return fmt.Errorf("proto: wrong wireType = %d for field SourceType", wireType)
			}
			m.SourceType = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAllocation
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.SourceType |= ObjectType(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SourceId", wireType)
			}
			m.SourceId = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAllocation
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.SourceId |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field DestinationId", wireType)
			}
			m.DestinationId = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAllocation
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
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Power", wireType)
			}
			m.Power = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAllocation
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
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Creator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAllocation
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
				return ErrInvalidLengthAllocation
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthAllocation
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Creator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Owner", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAllocation
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
				return ErrInvalidLengthAllocation
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthAllocation
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Owner = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 8:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Locked", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAllocation
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Locked = bool(v != 0)
		default:
			iNdEx = preIndex
			skippy, err := skipAllocation(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthAllocation
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
func (m *AllocationPackage) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAllocation
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
			return fmt.Errorf("proto: AllocationPackage: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: AllocationPackage: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Allocation", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAllocation
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthAllocation
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthAllocation
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Allocation == nil {
				m.Allocation = &Allocation{}
			}
			if err := m.Allocation.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Status", wireType)
			}
			m.Status = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAllocation
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Status |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipAllocation(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthAllocation
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
func skipAllocation(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowAllocation
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
					return 0, ErrIntOverflowAllocation
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
					return 0, ErrIntOverflowAllocation
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
				return 0, ErrInvalidLengthAllocation
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupAllocation
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthAllocation
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthAllocation        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowAllocation          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupAllocation = fmt.Errorf("proto: unexpected end of group")
)
