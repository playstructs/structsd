// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: structs/structs/fleet.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
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

type Fleet struct {
	Id                   string     `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Owner                string     `protobuf:"bytes,2,opt,name=owner,proto3" json:"owner,omitempty"`
	LocationType         ObjectType `protobuf:"varint,3,opt,name=locationType,proto3,enum=structs.structs.ObjectType" json:"locationType,omitempty"`
	LocationId           string     `protobuf:"bytes,4,opt,name=locationId,proto3" json:"locationId,omitempty"`
	LocationListForward  string     `protobuf:"bytes,5,opt,name=locationListForward,proto3" json:"locationListForward,omitempty"`
	LocationListBackward string     `protobuf:"bytes,6,opt,name=locationListBackward,proto3" json:"locationListBackward,omitempty"`
	Space                []string   `protobuf:"bytes,7,rep,name=space,proto3" json:"space,omitempty"`
	Air                  []string   `protobuf:"bytes,8,rep,name=air,proto3" json:"air,omitempty"`
	Land                 []string   `protobuf:"bytes,9,rep,name=land,proto3" json:"land,omitempty"`
	Water                []string   `protobuf:"bytes,10,rep,name=water,proto3" json:"water,omitempty"`
	SpaceSlots           uint64     `protobuf:"varint,11,opt,name=spaceSlots,proto3" json:"spaceSlots,omitempty"`
	AirSlots             uint64     `protobuf:"varint,12,opt,name=airSlots,proto3" json:"airSlots,omitempty"`
	LandSlots            uint64     `protobuf:"varint,13,opt,name=landSlots,proto3" json:"landSlots,omitempty"`
	WaterSlots           uint64     `protobuf:"varint,14,opt,name=waterSlots,proto3" json:"waterSlots,omitempty"`
}

func (m *Fleet) Reset()         { *m = Fleet{} }
func (m *Fleet) String() string { return proto.CompactTextString(m) }
func (*Fleet) ProtoMessage()    {}
func (*Fleet) Descriptor() ([]byte, []int) {
	return fileDescriptor_61dd153853b86e53, []int{0}
}
func (m *Fleet) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Fleet) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Fleet.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Fleet) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Fleet.Merge(m, src)
}
func (m *Fleet) XXX_Size() int {
	return m.Size()
}
func (m *Fleet) XXX_DiscardUnknown() {
	xxx_messageInfo_Fleet.DiscardUnknown(m)
}

var xxx_messageInfo_Fleet proto.InternalMessageInfo

func (m *Fleet) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Fleet) GetOwner() string {
	if m != nil {
		return m.Owner
	}
	return ""
}

func (m *Fleet) GetLocationType() ObjectType {
	if m != nil {
		return m.LocationType
	}
	return ObjectType_guild
}

func (m *Fleet) GetLocationId() string {
	if m != nil {
		return m.LocationId
	}
	return ""
}

func (m *Fleet) GetLocationListForward() string {
	if m != nil {
		return m.LocationListForward
	}
	return ""
}

func (m *Fleet) GetLocationListBackward() string {
	if m != nil {
		return m.LocationListBackward
	}
	return ""
}

func (m *Fleet) GetSpace() []string {
	if m != nil {
		return m.Space
	}
	return nil
}

func (m *Fleet) GetAir() []string {
	if m != nil {
		return m.Air
	}
	return nil
}

func (m *Fleet) GetLand() []string {
	if m != nil {
		return m.Land
	}
	return nil
}

func (m *Fleet) GetWater() []string {
	if m != nil {
		return m.Water
	}
	return nil
}

func (m *Fleet) GetSpaceSlots() uint64 {
	if m != nil {
		return m.SpaceSlots
	}
	return 0
}

func (m *Fleet) GetAirSlots() uint64 {
	if m != nil {
		return m.AirSlots
	}
	return 0
}

func (m *Fleet) GetLandSlots() uint64 {
	if m != nil {
		return m.LandSlots
	}
	return 0
}

func (m *Fleet) GetWaterSlots() uint64 {
	if m != nil {
		return m.WaterSlots
	}
	return 0
}

type FleetAttributeRecord struct {
	AttributeId string `protobuf:"bytes,1,opt,name=attributeId,proto3" json:"attributeId,omitempty"`
	Value       uint64 `protobuf:"varint,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (m *FleetAttributeRecord) Reset()         { *m = FleetAttributeRecord{} }
func (m *FleetAttributeRecord) String() string { return proto.CompactTextString(m) }
func (*FleetAttributeRecord) ProtoMessage()    {}
func (*FleetAttributeRecord) Descriptor() ([]byte, []int) {
	return fileDescriptor_61dd153853b86e53, []int{1}
}
func (m *FleetAttributeRecord) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *FleetAttributeRecord) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_FleetAttributeRecord.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *FleetAttributeRecord) XXX_Merge(src proto.Message) {
	xxx_messageInfo_FleetAttributeRecord.Merge(m, src)
}
func (m *FleetAttributeRecord) XXX_Size() int {
	return m.Size()
}
func (m *FleetAttributeRecord) XXX_DiscardUnknown() {
	xxx_messageInfo_FleetAttributeRecord.DiscardUnknown(m)
}

var xxx_messageInfo_FleetAttributeRecord proto.InternalMessageInfo

func (m *FleetAttributeRecord) GetAttributeId() string {
	if m != nil {
		return m.AttributeId
	}
	return ""
}

func (m *FleetAttributeRecord) GetValue() uint64 {
	if m != nil {
		return m.Value
	}
	return 0
}

func init() {
	proto.RegisterType((*Fleet)(nil), "structs.structs.Fleet")
	proto.RegisterType((*FleetAttributeRecord)(nil), "structs.structs.FleetAttributeRecord")
}

func init() { proto.RegisterFile("structs/structs/fleet.proto", fileDescriptor_61dd153853b86e53) }

var fileDescriptor_61dd153853b86e53 = []byte{
	// 389 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x92, 0xc1, 0x8e, 0xda, 0x30,
	0x10, 0x86, 0x09, 0x09, 0x14, 0x06, 0x4a, 0x5b, 0x17, 0xa9, 0x56, 0xa8, 0xa2, 0x88, 0x13, 0xa7,
	0xd0, 0xd2, 0x07, 0xa8, 0xca, 0x01, 0x09, 0xa9, 0xea, 0x21, 0xed, 0xa9, 0x37, 0x93, 0xb8, 0x92,
	0x4b, 0x1a, 0x47, 0x8e, 0x29, 0xe5, 0x2d, 0xfa, 0x48, 0x7b, 0xdc, 0x23, 0xc7, 0x3d, 0xae, 0xe0,
	0x45, 0x56, 0x1e, 0x13, 0xc8, 0x22, 0x2e, 0xd8, 0xff, 0xf7, 0xcf, 0xfc, 0x64, 0xe4, 0x81, 0x51,
	0xa9, 0xd5, 0x26, 0xd1, 0xe5, 0xb4, 0x3a, 0x7f, 0x65, 0x9c, 0xeb, 0xa8, 0x50, 0x52, 0x4b, 0xf2,
	0xea, 0x04, 0xa3, 0xd3, 0xe9, 0xfb, 0xd7, 0xd5, 0x6b, 0xbe, 0x2b, 0x6d, 0xb1, 0xff, 0x86, 0xfd,
	0x11, 0xb9, 0x9c, 0xe2, 0xaf, 0x45, 0xe3, 0x3b, 0x17, 0x5a, 0x0b, 0x93, 0x47, 0x06, 0xd0, 0x14,
	0x29, 0x75, 0x42, 0x67, 0xd2, 0x8d, 0x9b, 0x22, 0x25, 0x43, 0x68, 0xc9, 0x6d, 0xce, 0x15, 0x6d,
	0x22, 0xb2, 0x82, 0x7c, 0x86, 0x7e, 0x26, 0x13, 0xa6, 0x85, 0xcc, 0x7f, 0xec, 0x0a, 0x4e, 0xdd,
	0xd0, 0x99, 0x0c, 0x66, 0xa3, 0xe8, 0xea, 0x33, 0x22, 0xb9, 0xfa, 0xcd, 0x13, 0x6d, 0x4a, 0xe2,
	0x67, 0x0d, 0x24, 0x00, 0xa8, 0xf4, 0x32, 0xa5, 0x1e, 0x66, 0xd7, 0x08, 0xf9, 0x00, 0x6f, 0x2b,
	0xf5, 0x55, 0x94, 0x7a, 0x21, 0xd5, 0x96, 0xa9, 0x94, 0xb6, 0xb0, 0xf0, 0x96, 0x45, 0x66, 0x30,
	0xac, 0xe3, 0x39, 0x4b, 0xd6, 0xd8, 0xd2, 0xc6, 0x96, 0x9b, 0x9e, 0x19, 0xae, 0x2c, 0x58, 0xc2,
	0xe9, 0x8b, 0xd0, 0x35, 0xc3, 0xa1, 0x20, 0xaf, 0xc1, 0x65, 0x42, 0xd1, 0x0e, 0x32, 0x73, 0x25,
	0x04, 0xbc, 0x8c, 0xe5, 0x29, 0xed, 0x22, 0xc2, 0xbb, 0xe9, 0xdd, 0x32, 0xcd, 0x15, 0x05, 0xdb,
	0x8b, 0xc2, 0xcc, 0x85, 0x21, 0xdf, 0x33, 0xa9, 0x4b, 0xda, 0x0b, 0x9d, 0x89, 0x17, 0xd7, 0x08,
	0xf1, 0xa1, 0xc3, 0x84, 0xb2, 0x6e, 0x1f, 0xdd, 0xb3, 0x26, 0xef, 0xa1, 0x6b, 0x92, 0xad, 0xf9,
	0x12, 0xcd, 0x0b, 0x30, 0xc9, 0xf8, 0x17, 0xd6, 0x1e, 0xd8, 0xe4, 0x0b, 0x19, 0x7f, 0x83, 0x21,
	0xbe, 0xe0, 0x17, 0xad, 0x95, 0x58, 0x6d, 0x34, 0x8f, 0x79, 0x22, 0x55, 0x4a, 0x42, 0xe8, 0xb1,
	0x0a, 0x2d, 0xab, 0x97, 0xad, 0x23, 0x33, 0xc9, 0x5f, 0x96, 0x6d, 0x38, 0x3e, 0xb1, 0x17, 0x5b,
	0x31, 0xff, 0x78, 0x7f, 0x08, 0x9c, 0xfd, 0x21, 0x70, 0x1e, 0x0f, 0x81, 0xf3, 0xff, 0x18, 0x34,
	0xf6, 0xc7, 0xa0, 0xf1, 0x70, 0x0c, 0x1a, 0x3f, 0xdf, 0x55, 0x3b, 0xf5, 0xef, 0xbc, 0x5d, 0x7a,
	0x57, 0xf0, 0x72, 0xd5, 0xc6, 0x65, 0xfa, 0xf4, 0x14, 0x00, 0x00, 0xff, 0xff, 0x01, 0xd1, 0x9e,
	0x12, 0xab, 0x02, 0x00, 0x00,
}

func (m *Fleet) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Fleet) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Fleet) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.WaterSlots != 0 {
		i = encodeVarintFleet(dAtA, i, uint64(m.WaterSlots))
		i--
		dAtA[i] = 0x70
	}
	if m.LandSlots != 0 {
		i = encodeVarintFleet(dAtA, i, uint64(m.LandSlots))
		i--
		dAtA[i] = 0x68
	}
	if m.AirSlots != 0 {
		i = encodeVarintFleet(dAtA, i, uint64(m.AirSlots))
		i--
		dAtA[i] = 0x60
	}
	if m.SpaceSlots != 0 {
		i = encodeVarintFleet(dAtA, i, uint64(m.SpaceSlots))
		i--
		dAtA[i] = 0x58
	}
	if len(m.Water) > 0 {
		for iNdEx := len(m.Water) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Water[iNdEx])
			copy(dAtA[i:], m.Water[iNdEx])
			i = encodeVarintFleet(dAtA, i, uint64(len(m.Water[iNdEx])))
			i--
			dAtA[i] = 0x52
		}
	}
	if len(m.Land) > 0 {
		for iNdEx := len(m.Land) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Land[iNdEx])
			copy(dAtA[i:], m.Land[iNdEx])
			i = encodeVarintFleet(dAtA, i, uint64(len(m.Land[iNdEx])))
			i--
			dAtA[i] = 0x4a
		}
	}
	if len(m.Air) > 0 {
		for iNdEx := len(m.Air) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Air[iNdEx])
			copy(dAtA[i:], m.Air[iNdEx])
			i = encodeVarintFleet(dAtA, i, uint64(len(m.Air[iNdEx])))
			i--
			dAtA[i] = 0x42
		}
	}
	if len(m.Space) > 0 {
		for iNdEx := len(m.Space) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Space[iNdEx])
			copy(dAtA[i:], m.Space[iNdEx])
			i = encodeVarintFleet(dAtA, i, uint64(len(m.Space[iNdEx])))
			i--
			dAtA[i] = 0x3a
		}
	}
	if len(m.LocationListBackward) > 0 {
		i -= len(m.LocationListBackward)
		copy(dAtA[i:], m.LocationListBackward)
		i = encodeVarintFleet(dAtA, i, uint64(len(m.LocationListBackward)))
		i--
		dAtA[i] = 0x32
	}
	if len(m.LocationListForward) > 0 {
		i -= len(m.LocationListForward)
		copy(dAtA[i:], m.LocationListForward)
		i = encodeVarintFleet(dAtA, i, uint64(len(m.LocationListForward)))
		i--
		dAtA[i] = 0x2a
	}
	if len(m.LocationId) > 0 {
		i -= len(m.LocationId)
		copy(dAtA[i:], m.LocationId)
		i = encodeVarintFleet(dAtA, i, uint64(len(m.LocationId)))
		i--
		dAtA[i] = 0x22
	}
	if m.LocationType != 0 {
		i = encodeVarintFleet(dAtA, i, uint64(m.LocationType))
		i--
		dAtA[i] = 0x18
	}
	if len(m.Owner) > 0 {
		i -= len(m.Owner)
		copy(dAtA[i:], m.Owner)
		i = encodeVarintFleet(dAtA, i, uint64(len(m.Owner)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintFleet(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *FleetAttributeRecord) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *FleetAttributeRecord) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *FleetAttributeRecord) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Value != 0 {
		i = encodeVarintFleet(dAtA, i, uint64(m.Value))
		i--
		dAtA[i] = 0x10
	}
	if len(m.AttributeId) > 0 {
		i -= len(m.AttributeId)
		copy(dAtA[i:], m.AttributeId)
		i = encodeVarintFleet(dAtA, i, uint64(len(m.AttributeId)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintFleet(dAtA []byte, offset int, v uint64) int {
	offset -= sovFleet(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Fleet) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovFleet(uint64(l))
	}
	l = len(m.Owner)
	if l > 0 {
		n += 1 + l + sovFleet(uint64(l))
	}
	if m.LocationType != 0 {
		n += 1 + sovFleet(uint64(m.LocationType))
	}
	l = len(m.LocationId)
	if l > 0 {
		n += 1 + l + sovFleet(uint64(l))
	}
	l = len(m.LocationListForward)
	if l > 0 {
		n += 1 + l + sovFleet(uint64(l))
	}
	l = len(m.LocationListBackward)
	if l > 0 {
		n += 1 + l + sovFleet(uint64(l))
	}
	if len(m.Space) > 0 {
		for _, s := range m.Space {
			l = len(s)
			n += 1 + l + sovFleet(uint64(l))
		}
	}
	if len(m.Air) > 0 {
		for _, s := range m.Air {
			l = len(s)
			n += 1 + l + sovFleet(uint64(l))
		}
	}
	if len(m.Land) > 0 {
		for _, s := range m.Land {
			l = len(s)
			n += 1 + l + sovFleet(uint64(l))
		}
	}
	if len(m.Water) > 0 {
		for _, s := range m.Water {
			l = len(s)
			n += 1 + l + sovFleet(uint64(l))
		}
	}
	if m.SpaceSlots != 0 {
		n += 1 + sovFleet(uint64(m.SpaceSlots))
	}
	if m.AirSlots != 0 {
		n += 1 + sovFleet(uint64(m.AirSlots))
	}
	if m.LandSlots != 0 {
		n += 1 + sovFleet(uint64(m.LandSlots))
	}
	if m.WaterSlots != 0 {
		n += 1 + sovFleet(uint64(m.WaterSlots))
	}
	return n
}

func (m *FleetAttributeRecord) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.AttributeId)
	if l > 0 {
		n += 1 + l + sovFleet(uint64(l))
	}
	if m.Value != 0 {
		n += 1 + sovFleet(uint64(m.Value))
	}
	return n
}

func sovFleet(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozFleet(x uint64) (n int) {
	return sovFleet(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Fleet) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowFleet
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
			return fmt.Errorf("proto: Fleet: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Fleet: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFleet
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
				return ErrInvalidLengthFleet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFleet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Owner", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFleet
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
				return ErrInvalidLengthFleet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFleet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Owner = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LocationType", wireType)
			}
			m.LocationType = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFleet
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LocationType |= ObjectType(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LocationId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFleet
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
				return ErrInvalidLengthFleet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFleet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.LocationId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LocationListForward", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFleet
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
				return ErrInvalidLengthFleet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFleet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.LocationListForward = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LocationListBackward", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFleet
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
				return ErrInvalidLengthFleet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFleet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.LocationListBackward = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Space", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFleet
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
				return ErrInvalidLengthFleet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFleet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Space = append(m.Space, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Air", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFleet
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
				return ErrInvalidLengthFleet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFleet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Air = append(m.Air, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Land", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFleet
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
				return ErrInvalidLengthFleet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFleet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Land = append(m.Land, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Water", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFleet
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
				return ErrInvalidLengthFleet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFleet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Water = append(m.Water, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 11:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SpaceSlots", wireType)
			}
			m.SpaceSlots = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFleet
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.SpaceSlots |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 12:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field AirSlots", wireType)
			}
			m.AirSlots = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFleet
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.AirSlots |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 13:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LandSlots", wireType)
			}
			m.LandSlots = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFleet
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LandSlots |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 14:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field WaterSlots", wireType)
			}
			m.WaterSlots = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFleet
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.WaterSlots |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipFleet(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthFleet
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
func (m *FleetAttributeRecord) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowFleet
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
			return fmt.Errorf("proto: FleetAttributeRecord: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: FleetAttributeRecord: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AttributeId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFleet
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
				return ErrInvalidLengthFleet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFleet
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
					return ErrIntOverflowFleet
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
			skippy, err := skipFleet(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthFleet
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
func skipFleet(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowFleet
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
					return 0, ErrIntOverflowFleet
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
					return 0, ErrIntOverflowFleet
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
				return 0, ErrInvalidLengthFleet
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupFleet
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthFleet
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthFleet        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowFleet          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupFleet = fmt.Errorf("proto: unexpected end of group")
)
