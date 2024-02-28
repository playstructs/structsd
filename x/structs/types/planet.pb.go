// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: structs/structs/planet.proto

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

type Planet struct {
	Id           string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	MaxOre       uint64   `protobuf:"varint,2,opt,name=maxOre,proto3" json:"maxOre,omitempty"`
	OreRemaining uint64   `protobuf:"varint,3,opt,name=OreRemaining,proto3" json:"OreRemaining,omitempty"`
	OreStored    uint64   `protobuf:"varint,4,opt,name=OreStored,proto3" json:"OreStored,omitempty"`
	Creator      string   `protobuf:"bytes,5,opt,name=creator,proto3" json:"creator,omitempty"`
	Owner        string   `protobuf:"bytes,6,opt,name=owner,proto3" json:"owner,omitempty"`
	Space        []string `protobuf:"bytes,7,rep,name=space,proto3" json:"space,omitempty"`
	Sky          []string `protobuf:"bytes,8,rep,name=sky,proto3" json:"sky,omitempty"`
	Land         []string `protobuf:"bytes,9,rep,name=land,proto3" json:"land,omitempty"`
	Water        []string `protobuf:"bytes,10,rep,name=water,proto3" json:"water,omitempty"`
	SpaceSlots   uint64   `protobuf:"varint,11,opt,name=spaceSlots,proto3" json:"spaceSlots,omitempty"`
	SkySlots     uint64   `protobuf:"varint,12,opt,name=skySlots,proto3" json:"skySlots,omitempty"`
	LandSlots    uint64   `protobuf:"varint,13,opt,name=landSlots,proto3" json:"landSlots,omitempty"`
	WaterSlots   uint64   `protobuf:"varint,14,opt,name=waterSlots,proto3" json:"waterSlots,omitempty"`
	Status       uint64   `protobuf:"varint,15,opt,name=status,proto3" json:"status,omitempty"`
}

func (m *Planet) Reset()         { *m = Planet{} }
func (m *Planet) String() string { return proto.CompactTextString(m) }
func (*Planet) ProtoMessage()    {}
func (*Planet) Descriptor() ([]byte, []int) {
	return fileDescriptor_6d6079b21199cebc, []int{0}
}
func (m *Planet) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Planet) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Planet.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Planet) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Planet.Merge(m, src)
}
func (m *Planet) XXX_Size() int {
	return m.Size()
}
func (m *Planet) XXX_DiscardUnknown() {
	xxx_messageInfo_Planet.DiscardUnknown(m)
}

var xxx_messageInfo_Planet proto.InternalMessageInfo

func (m *Planet) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Planet) GetMaxOre() uint64 {
	if m != nil {
		return m.MaxOre
	}
	return 0
}

func (m *Planet) GetOreRemaining() uint64 {
	if m != nil {
		return m.OreRemaining
	}
	return 0
}

func (m *Planet) GetOreStored() uint64 {
	if m != nil {
		return m.OreStored
	}
	return 0
}

func (m *Planet) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

func (m *Planet) GetOwner() string {
	if m != nil {
		return m.Owner
	}
	return ""
}

func (m *Planet) GetSpace() []string {
	if m != nil {
		return m.Space
	}
	return nil
}

func (m *Planet) GetSky() []string {
	if m != nil {
		return m.Sky
	}
	return nil
}

func (m *Planet) GetLand() []string {
	if m != nil {
		return m.Land
	}
	return nil
}

func (m *Planet) GetWater() []string {
	if m != nil {
		return m.Water
	}
	return nil
}

func (m *Planet) GetSpaceSlots() uint64 {
	if m != nil {
		return m.SpaceSlots
	}
	return 0
}

func (m *Planet) GetSkySlots() uint64 {
	if m != nil {
		return m.SkySlots
	}
	return 0
}

func (m *Planet) GetLandSlots() uint64 {
	if m != nil {
		return m.LandSlots
	}
	return 0
}

func (m *Planet) GetWaterSlots() uint64 {
	if m != nil {
		return m.WaterSlots
	}
	return 0
}

func (m *Planet) GetStatus() uint64 {
	if m != nil {
		return m.Status
	}
	return 0
}

func init() {
	proto.RegisterType((*Planet)(nil), "structs.Planet")
}

func init() { proto.RegisterFile("structs/structs/planet.proto", fileDescriptor_6d6079b21199cebc) }

var fileDescriptor_6d6079b21199cebc = []byte{
	// 315 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x54, 0x91, 0xc1, 0x4e, 0x3a, 0x31,
	0x10, 0xc6, 0x29, 0x0b, 0x0b, 0xcc, 0x9f, 0x3f, 0x9a, 0x89, 0xd1, 0x89, 0x21, 0x0d, 0xe1, 0xc4,
	0x49, 0x63, 0x7c, 0x03, 0x5f, 0x00, 0x03, 0x37, 0x6f, 0x95, 0x6d, 0xcc, 0x06, 0xd8, 0x6e, 0xda,
	0x12, 0xe0, 0x05, 0x3c, 0xfb, 0x58, 0x1e, 0x39, 0x7a, 0x34, 0xf0, 0x22, 0xa6, 0x53, 0x58, 0xf4,
	0xb4, 0xfd, 0x7e, 0xdf, 0xf6, 0x9b, 0xce, 0x0c, 0xf4, 0x9d, 0xb7, 0xab, 0x99, 0x77, 0xf7, 0xa7,
	0x6f, 0xb9, 0x50, 0x85, 0xf6, 0x77, 0xa5, 0x35, 0xde, 0x60, 0xeb, 0x48, 0x87, 0xef, 0x09, 0xa4,
	0xcf, 0xec, 0x60, 0x0f, 0xea, 0x79, 0x46, 0x62, 0x20, 0x46, 0x9d, 0x49, 0x3d, 0xcf, 0xf0, 0x1a,
	0xd2, 0xa5, 0xda, 0x8c, 0xad, 0xa6, 0xfa, 0x40, 0x8c, 0x1a, 0x93, 0xa3, 0xc2, 0x21, 0x74, 0xc7,
	0x56, 0x4f, 0xf4, 0x52, 0xe5, 0x45, 0x5e, 0xbc, 0x51, 0xc2, 0xee, 0x1f, 0x86, 0x7d, 0xe8, 0x8c,
	0xad, 0x9e, 0x7a, 0x63, 0x75, 0x46, 0x0d, 0xfe, 0xe1, 0x0c, 0x90, 0xa0, 0x35, 0xb3, 0x5a, 0x79,
	0x63, 0xa9, 0xc9, 0xe5, 0x4e, 0x12, 0xaf, 0xa0, 0x69, 0xd6, 0x85, 0xb6, 0x94, 0x32, 0x8f, 0x22,
	0x50, 0x57, 0xaa, 0x99, 0xa6, 0xd6, 0x20, 0x09, 0x94, 0x05, 0x5e, 0x42, 0xe2, 0xe6, 0x5b, 0x6a,
	0x33, 0x0b, 0x47, 0x44, 0x68, 0x2c, 0x54, 0x91, 0x51, 0x87, 0x11, 0x9f, 0xc3, 0xdd, 0xb5, 0xf2,
	0xda, 0x12, 0xc4, 0xbb, 0x2c, 0x50, 0x02, 0x70, 0xc8, 0x74, 0x61, 0xbc, 0xa3, 0x7f, 0xfc, 0xc0,
	0x5f, 0x04, 0x6f, 0xa1, 0xed, 0xe6, 0xdb, 0xe8, 0x76, 0xd9, 0xad, 0x74, 0xe8, 0x2d, 0x24, 0x47,
	0xf3, 0x7f, 0xec, 0xad, 0x02, 0x21, 0x99, 0x4b, 0x44, 0xbb, 0x17, 0x93, 0xcf, 0x24, 0x4c, 0xd5,
	0x79, 0xe5, 0x57, 0x8e, 0x2e, 0xe2, 0x54, 0xa3, 0x7a, 0x7a, 0xf8, 0xdc, 0x4b, 0xb1, 0xdb, 0x4b,
	0xf1, 0xbd, 0x97, 0xe2, 0xe3, 0x20, 0x6b, 0xbb, 0x83, 0xac, 0x7d, 0x1d, 0x64, 0xed, 0xe5, 0xe6,
	0xb4, 0xc1, 0x4d, 0xb5, 0x4b, 0xbf, 0x2d, 0xb5, 0x7b, 0x4d, 0x79, 0x97, 0x8f, 0x3f, 0x01, 0x00,
	0x00, 0xff, 0xff, 0x5a, 0xbb, 0x7b, 0x46, 0xeb, 0x01, 0x00, 0x00,
}

func (m *Planet) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Planet) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Planet) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Status != 0 {
		i = encodeVarintPlanet(dAtA, i, uint64(m.Status))
		i--
		dAtA[i] = 0x78
	}
	if m.WaterSlots != 0 {
		i = encodeVarintPlanet(dAtA, i, uint64(m.WaterSlots))
		i--
		dAtA[i] = 0x70
	}
	if m.LandSlots != 0 {
		i = encodeVarintPlanet(dAtA, i, uint64(m.LandSlots))
		i--
		dAtA[i] = 0x68
	}
	if m.SkySlots != 0 {
		i = encodeVarintPlanet(dAtA, i, uint64(m.SkySlots))
		i--
		dAtA[i] = 0x60
	}
	if m.SpaceSlots != 0 {
		i = encodeVarintPlanet(dAtA, i, uint64(m.SpaceSlots))
		i--
		dAtA[i] = 0x58
	}
	if len(m.Water) > 0 {
		for iNdEx := len(m.Water) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Water[iNdEx])
			copy(dAtA[i:], m.Water[iNdEx])
			i = encodeVarintPlanet(dAtA, i, uint64(len(m.Water[iNdEx])))
			i--
			dAtA[i] = 0x52
		}
	}
	if len(m.Land) > 0 {
		for iNdEx := len(m.Land) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Land[iNdEx])
			copy(dAtA[i:], m.Land[iNdEx])
			i = encodeVarintPlanet(dAtA, i, uint64(len(m.Land[iNdEx])))
			i--
			dAtA[i] = 0x4a
		}
	}
	if len(m.Sky) > 0 {
		for iNdEx := len(m.Sky) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Sky[iNdEx])
			copy(dAtA[i:], m.Sky[iNdEx])
			i = encodeVarintPlanet(dAtA, i, uint64(len(m.Sky[iNdEx])))
			i--
			dAtA[i] = 0x42
		}
	}
	if len(m.Space) > 0 {
		for iNdEx := len(m.Space) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Space[iNdEx])
			copy(dAtA[i:], m.Space[iNdEx])
			i = encodeVarintPlanet(dAtA, i, uint64(len(m.Space[iNdEx])))
			i--
			dAtA[i] = 0x3a
		}
	}
	if len(m.Owner) > 0 {
		i -= len(m.Owner)
		copy(dAtA[i:], m.Owner)
		i = encodeVarintPlanet(dAtA, i, uint64(len(m.Owner)))
		i--
		dAtA[i] = 0x32
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintPlanet(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x2a
	}
	if m.OreStored != 0 {
		i = encodeVarintPlanet(dAtA, i, uint64(m.OreStored))
		i--
		dAtA[i] = 0x20
	}
	if m.OreRemaining != 0 {
		i = encodeVarintPlanet(dAtA, i, uint64(m.OreRemaining))
		i--
		dAtA[i] = 0x18
	}
	if m.MaxOre != 0 {
		i = encodeVarintPlanet(dAtA, i, uint64(m.MaxOre))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintPlanet(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintPlanet(dAtA []byte, offset int, v uint64) int {
	offset -= sovPlanet(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Planet) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovPlanet(uint64(l))
	}
	if m.MaxOre != 0 {
		n += 1 + sovPlanet(uint64(m.MaxOre))
	}
	if m.OreRemaining != 0 {
		n += 1 + sovPlanet(uint64(m.OreRemaining))
	}
	if m.OreStored != 0 {
		n += 1 + sovPlanet(uint64(m.OreStored))
	}
	l = len(m.Creator)
	if l > 0 {
		n += 1 + l + sovPlanet(uint64(l))
	}
	l = len(m.Owner)
	if l > 0 {
		n += 1 + l + sovPlanet(uint64(l))
	}
	if len(m.Space) > 0 {
		for _, s := range m.Space {
			l = len(s)
			n += 1 + l + sovPlanet(uint64(l))
		}
	}
	if len(m.Sky) > 0 {
		for _, s := range m.Sky {
			l = len(s)
			n += 1 + l + sovPlanet(uint64(l))
		}
	}
	if len(m.Land) > 0 {
		for _, s := range m.Land {
			l = len(s)
			n += 1 + l + sovPlanet(uint64(l))
		}
	}
	if len(m.Water) > 0 {
		for _, s := range m.Water {
			l = len(s)
			n += 1 + l + sovPlanet(uint64(l))
		}
	}
	if m.SpaceSlots != 0 {
		n += 1 + sovPlanet(uint64(m.SpaceSlots))
	}
	if m.SkySlots != 0 {
		n += 1 + sovPlanet(uint64(m.SkySlots))
	}
	if m.LandSlots != 0 {
		n += 1 + sovPlanet(uint64(m.LandSlots))
	}
	if m.WaterSlots != 0 {
		n += 1 + sovPlanet(uint64(m.WaterSlots))
	}
	if m.Status != 0 {
		n += 1 + sovPlanet(uint64(m.Status))
	}
	return n
}

func sovPlanet(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozPlanet(x uint64) (n int) {
	return sovPlanet(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Planet) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPlanet
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
			return fmt.Errorf("proto: Planet: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Planet: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlanet
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
				return ErrInvalidLengthPlanet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlanet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaxOre", wireType)
			}
			m.MaxOre = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlanet
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MaxOre |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field OreRemaining", wireType)
			}
			m.OreRemaining = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlanet
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.OreRemaining |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field OreStored", wireType)
			}
			m.OreStored = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlanet
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.OreStored |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Creator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlanet
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
				return ErrInvalidLengthPlanet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlanet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Creator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Owner", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlanet
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
				return ErrInvalidLengthPlanet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlanet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Owner = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Space", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlanet
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
				return ErrInvalidLengthPlanet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlanet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Space = append(m.Space, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Sky", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlanet
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
				return ErrInvalidLengthPlanet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlanet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Sky = append(m.Sky, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Land", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlanet
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
				return ErrInvalidLengthPlanet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlanet
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
					return ErrIntOverflowPlanet
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
				return ErrInvalidLengthPlanet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlanet
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
					return ErrIntOverflowPlanet
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
				return fmt.Errorf("proto: wrong wireType = %d for field SkySlots", wireType)
			}
			m.SkySlots = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlanet
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.SkySlots |= uint64(b&0x7F) << shift
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
					return ErrIntOverflowPlanet
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
					return ErrIntOverflowPlanet
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
		case 15:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Status", wireType)
			}
			m.Status = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlanet
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
			skippy, err := skipPlanet(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthPlanet
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
func skipPlanet(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowPlanet
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
					return 0, ErrIntOverflowPlanet
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
					return 0, ErrIntOverflowPlanet
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
				return 0, ErrInvalidLengthPlanet
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupPlanet
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthPlanet
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthPlanet        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowPlanet          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupPlanet = fmt.Errorf("proto: unexpected end of group")
)
