// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: structs/structs/player.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	types "github.com/cosmos/cosmos-sdk/types"
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

type Player struct {
	Id                string     `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Index             uint64     `protobuf:"varint,2,opt,name=index,proto3" json:"index,omitempty"`
	GuildId           string     `protobuf:"bytes,3,opt,name=guildId,proto3" json:"guildId,omitempty"`
	SubstationId      string     `protobuf:"bytes,4,opt,name=substationId,proto3" json:"substationId,omitempty"`
	Creator           string     `protobuf:"bytes,5,opt,name=creator,proto3" json:"creator,omitempty"`
	PrimaryAddress    string     `protobuf:"bytes,6,opt,name=primaryAddress,proto3" json:"primaryAddress,omitempty"`
	PlanetId          string     `protobuf:"bytes,7,opt,name=planetId,proto3" json:"planetId,omitempty"`
	Load              uint64     `protobuf:"varint,8,opt,name=load,proto3" json:"load,omitempty"`
	Capacity          uint64     `protobuf:"varint,9,opt,name=capacity,proto3" json:"capacity,omitempty"`
	CapacitySecondary uint64     `protobuf:"varint,10,opt,name=capacitySecondary,proto3" json:"capacitySecondary,omitempty"`
	StructsLoad       uint64     `protobuf:"varint,11,opt,name=structsLoad,proto3" json:"structsLoad,omitempty"`
	Storage           types.Coin `protobuf:"bytes,12,opt,name=storage,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"storage"`
}

func (m *Player) Reset()         { *m = Player{} }
func (m *Player) String() string { return proto.CompactTextString(m) }
func (*Player) ProtoMessage()    {}
func (*Player) Descriptor() ([]byte, []int) {
	return fileDescriptor_ca9c11ebc41e4761, []int{0}
}
func (m *Player) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Player) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Player.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Player) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Player.Merge(m, src)
}
func (m *Player) XXX_Size() int {
	return m.Size()
}
func (m *Player) XXX_DiscardUnknown() {
	xxx_messageInfo_Player.DiscardUnknown(m)
}

var xxx_messageInfo_Player proto.InternalMessageInfo

func (m *Player) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Player) GetIndex() uint64 {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *Player) GetGuildId() string {
	if m != nil {
		return m.GuildId
	}
	return ""
}

func (m *Player) GetSubstationId() string {
	if m != nil {
		return m.SubstationId
	}
	return ""
}

func (m *Player) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

func (m *Player) GetPrimaryAddress() string {
	if m != nil {
		return m.PrimaryAddress
	}
	return ""
}

func (m *Player) GetPlanetId() string {
	if m != nil {
		return m.PlanetId
	}
	return ""
}

func (m *Player) GetLoad() uint64 {
	if m != nil {
		return m.Load
	}
	return 0
}

func (m *Player) GetCapacity() uint64 {
	if m != nil {
		return m.Capacity
	}
	return 0
}

func (m *Player) GetCapacitySecondary() uint64 {
	if m != nil {
		return m.CapacitySecondary
	}
	return 0
}

func (m *Player) GetStructsLoad() uint64 {
	if m != nil {
		return m.StructsLoad
	}
	return 0
}

func (m *Player) GetStorage() types.Coin {
	if m != nil {
		return m.Storage
	}
	return types.Coin{}
}

func init() {
	proto.RegisterType((*Player)(nil), "structs.structs.Player")
}

func init() { proto.RegisterFile("structs/structs/player.proto", fileDescriptor_ca9c11ebc41e4761) }

var fileDescriptor_ca9c11ebc41e4761 = []byte{
	// 410 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x64, 0x52, 0xbd, 0x0e, 0xd3, 0x30,
	0x10, 0x4e, 0xfa, 0x5f, 0xb7, 0x2a, 0xc2, 0xaa, 0x84, 0x5b, 0xa1, 0x34, 0xea, 0x94, 0x81, 0x26,
	0x14, 0x5e, 0x00, 0xca, 0x54, 0x89, 0x01, 0xa5, 0x1b, 0x0b, 0x72, 0x6c, 0x2b, 0x58, 0xa4, 0x71,
	0x64, 0x3b, 0xa8, 0x79, 0x0b, 0x9e, 0x83, 0x99, 0x87, 0xa8, 0xc4, 0x52, 0x31, 0x31, 0x01, 0x6a,
	0x5f, 0x04, 0xc5, 0x71, 0x2a, 0x7e, 0xa6, 0xbb, 0xef, 0xe7, 0xee, 0x74, 0xba, 0x03, 0x8f, 0x95,
	0x96, 0x25, 0xd1, 0x2a, 0x6a, 0x63, 0x91, 0xe1, 0x8a, 0xc9, 0xb0, 0x90, 0x42, 0x0b, 0xf8, 0xc0,
	0xb2, 0xa1, 0x8d, 0x4b, 0x8f, 0x08, 0x75, 0x14, 0x2a, 0x4a, 0xb0, 0x62, 0xd1, 0xc7, 0x6d, 0xc2,
	0x34, 0xde, 0x46, 0x44, 0xf0, 0xbc, 0x29, 0x58, 0x2e, 0x1a, 0xfd, 0x9d, 0x41, 0x51, 0x03, 0xac,
	0x34, 0x4f, 0x45, 0x2a, 0x1a, 0xbe, 0xce, 0x1a, 0x76, 0xfd, 0xb5, 0x0b, 0x06, 0x6f, 0xcc, 0x48,
	0x38, 0x03, 0x1d, 0x4e, 0x91, 0xeb, 0xbb, 0xc1, 0x38, 0xee, 0x70, 0x0a, 0xe7, 0xa0, 0xcf, 0x73,
	0xca, 0x4e, 0xa8, 0xe3, 0xbb, 0x41, 0x2f, 0x6e, 0x00, 0x44, 0x60, 0x98, 0x96, 0x3c, 0xa3, 0x7b,
	0x8a, 0xba, 0xc6, 0xda, 0x42, 0xb8, 0x06, 0x53, 0x55, 0x26, 0x4a, 0x63, 0xcd, 0x45, 0xbe, 0xa7,
	0xa8, 0x67, 0xe4, 0xbf, 0xb8, 0xba, 0x9a, 0x48, 0x86, 0xb5, 0x90, 0xa8, 0xdf, 0x54, 0x5b, 0x08,
	0x5f, 0x80, 0x59, 0x21, 0xf9, 0x11, 0xcb, 0xea, 0x25, 0xa5, 0x92, 0x29, 0x85, 0x06, 0xb5, 0x61,
	0x87, 0xbe, 0x7d, 0xd9, 0xcc, 0xed, 0x22, 0x56, 0x39, 0x68, 0xc9, 0xf3, 0x34, 0xfe, 0xc7, 0x0f,
	0x97, 0x60, 0x54, 0x64, 0x38, 0x67, 0x7a, 0x4f, 0xd1, 0xd0, 0x34, 0xbf, 0x63, 0x08, 0x41, 0x2f,
	0x13, 0x98, 0xa2, 0x91, 0x59, 0xc5, 0xe4, 0xb5, 0x9f, 0xe0, 0x02, 0x13, 0xae, 0x2b, 0x34, 0x36,
	0xfc, 0x1d, 0xc3, 0x27, 0xe0, 0x61, 0x9b, 0x1f, 0x18, 0x11, 0x39, 0xc5, 0xb2, 0x42, 0xc0, 0x98,
	0xfe, 0x17, 0xa0, 0x0f, 0x26, 0xf6, 0x40, 0xaf, 0xeb, 0x21, 0x13, 0xe3, 0xfb, 0x93, 0x82, 0x0c,
	0x0c, 0x95, 0x16, 0x12, 0xa7, 0x0c, 0x4d, 0x7d, 0x37, 0x98, 0x3c, 0x5b, 0x84, 0x76, 0xa7, 0xfa,
	0x92, 0xa1, 0xbd, 0x64, 0xf8, 0x4a, 0xf0, 0x7c, 0xf7, 0xf4, 0xfc, 0x63, 0xe5, 0x7c, 0xfe, 0xb9,
	0x0a, 0x52, 0xae, 0xdf, 0x97, 0x49, 0x48, 0xc4, 0xd1, 0x5e, 0xd2, 0x86, 0x8d, 0xa2, 0x1f, 0x22,
	0x5d, 0x15, 0x4c, 0x99, 0x02, 0x15, 0xb7, 0xbd, 0x77, 0xdb, 0xf3, 0xd5, 0x73, 0x2f, 0x57, 0xcf,
	0xfd, 0x75, 0xf5, 0xdc, 0x4f, 0x37, 0xcf, 0xb9, 0xdc, 0x3c, 0xe7, 0xfb, 0xcd, 0x73, 0xde, 0x3e,
	0x6a, 0xff, 0xeb, 0x74, 0xff, 0x34, 0xd3, 0x21, 0x19, 0x98, 0x3f, 0x78, 0xfe, 0x3b, 0x00, 0x00,
	0xff, 0xff, 0x7c, 0x38, 0x1b, 0x5b, 0x89, 0x02, 0x00, 0x00,
}

func (m *Player) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Player) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Player) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Storage.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintPlayer(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x62
	if m.StructsLoad != 0 {
		i = encodeVarintPlayer(dAtA, i, uint64(m.StructsLoad))
		i--
		dAtA[i] = 0x58
	}
	if m.CapacitySecondary != 0 {
		i = encodeVarintPlayer(dAtA, i, uint64(m.CapacitySecondary))
		i--
		dAtA[i] = 0x50
	}
	if m.Capacity != 0 {
		i = encodeVarintPlayer(dAtA, i, uint64(m.Capacity))
		i--
		dAtA[i] = 0x48
	}
	if m.Load != 0 {
		i = encodeVarintPlayer(dAtA, i, uint64(m.Load))
		i--
		dAtA[i] = 0x40
	}
	if len(m.PlanetId) > 0 {
		i -= len(m.PlanetId)
		copy(dAtA[i:], m.PlanetId)
		i = encodeVarintPlayer(dAtA, i, uint64(len(m.PlanetId)))
		i--
		dAtA[i] = 0x3a
	}
	if len(m.PrimaryAddress) > 0 {
		i -= len(m.PrimaryAddress)
		copy(dAtA[i:], m.PrimaryAddress)
		i = encodeVarintPlayer(dAtA, i, uint64(len(m.PrimaryAddress)))
		i--
		dAtA[i] = 0x32
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintPlayer(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x2a
	}
	if len(m.SubstationId) > 0 {
		i -= len(m.SubstationId)
		copy(dAtA[i:], m.SubstationId)
		i = encodeVarintPlayer(dAtA, i, uint64(len(m.SubstationId)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.GuildId) > 0 {
		i -= len(m.GuildId)
		copy(dAtA[i:], m.GuildId)
		i = encodeVarintPlayer(dAtA, i, uint64(len(m.GuildId)))
		i--
		dAtA[i] = 0x1a
	}
	if m.Index != 0 {
		i = encodeVarintPlayer(dAtA, i, uint64(m.Index))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintPlayer(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintPlayer(dAtA []byte, offset int, v uint64) int {
	offset -= sovPlayer(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Player) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovPlayer(uint64(l))
	}
	if m.Index != 0 {
		n += 1 + sovPlayer(uint64(m.Index))
	}
	l = len(m.GuildId)
	if l > 0 {
		n += 1 + l + sovPlayer(uint64(l))
	}
	l = len(m.SubstationId)
	if l > 0 {
		n += 1 + l + sovPlayer(uint64(l))
	}
	l = len(m.Creator)
	if l > 0 {
		n += 1 + l + sovPlayer(uint64(l))
	}
	l = len(m.PrimaryAddress)
	if l > 0 {
		n += 1 + l + sovPlayer(uint64(l))
	}
	l = len(m.PlanetId)
	if l > 0 {
		n += 1 + l + sovPlayer(uint64(l))
	}
	if m.Load != 0 {
		n += 1 + sovPlayer(uint64(m.Load))
	}
	if m.Capacity != 0 {
		n += 1 + sovPlayer(uint64(m.Capacity))
	}
	if m.CapacitySecondary != 0 {
		n += 1 + sovPlayer(uint64(m.CapacitySecondary))
	}
	if m.StructsLoad != 0 {
		n += 1 + sovPlayer(uint64(m.StructsLoad))
	}
	l = m.Storage.Size()
	n += 1 + l + sovPlayer(uint64(l))
	return n
}

func sovPlayer(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozPlayer(x uint64) (n int) {
	return sovPlayer(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Player) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPlayer
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
			return fmt.Errorf("proto: Player: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Player: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlayer
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
				return ErrInvalidLengthPlayer
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlayer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Index", wireType)
			}
			m.Index = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlayer
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Index |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field GuildId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlayer
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
				return ErrInvalidLengthPlayer
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlayer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.GuildId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SubstationId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlayer
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
				return ErrInvalidLengthPlayer
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlayer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SubstationId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Creator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlayer
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
				return ErrInvalidLengthPlayer
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlayer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Creator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PrimaryAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlayer
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
				return ErrInvalidLengthPlayer
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlayer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.PrimaryAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PlanetId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlayer
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
				return ErrInvalidLengthPlayer
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlayer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.PlanetId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 8:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Load", wireType)
			}
			m.Load = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlayer
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
		case 9:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Capacity", wireType)
			}
			m.Capacity = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlayer
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
		case 10:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field CapacitySecondary", wireType)
			}
			m.CapacitySecondary = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlayer
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.CapacitySecondary |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 11:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field StructsLoad", wireType)
			}
			m.StructsLoad = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlayer
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
		case 12:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Storage", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlayer
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
				return ErrInvalidLengthPlayer
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthPlayer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Storage.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPlayer(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthPlayer
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
func skipPlayer(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowPlayer
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
					return 0, ErrIntOverflowPlayer
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
					return 0, ErrIntOverflowPlayer
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
				return 0, ErrInvalidLengthPlayer
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupPlayer
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthPlayer
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthPlayer        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowPlayer          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupPlayer = fmt.Errorf("proto: unexpected end of group")
)
