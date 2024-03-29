// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: structs/structs/struct.proto

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

// This will all need to be reworked but let's
// do some super basic crap here to get testnet up
type Struct struct {
	Id        string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Creator   string `protobuf:"bytes,2,opt,name=creator,proto3" json:"creator,omitempty"`
	Owner     string `protobuf:"bytes,3,opt,name=owner,proto3" json:"owner,omitempty"`
	Status    string `protobuf:"bytes,4,opt,name=status,proto3" json:"status,omitempty"`
	MaxHealth uint64 `protobuf:"varint,5,opt,name=maxHealth,proto3" json:"maxHealth,omitempty"`
	Health    uint64 `protobuf:"varint,6,opt,name=health,proto3" json:"health,omitempty"`
	// Planet or Fleet
	Category string `protobuf:"bytes,7,opt,name=category,proto3" json:"category,omitempty"`
	// What it is
	Type string `protobuf:"bytes,8,opt,name=type,proto3" json:"type,omitempty"`
	// Where it is
	Ambit    string `protobuf:"bytes,9,opt,name=ambit,proto3" json:"ambit,omitempty"`
	PlanetId string `protobuf:"bytes,10,opt,name=planetId,proto3" json:"planetId,omitempty"`
	Slot     uint64 `protobuf:"varint,11,opt,name=slot,proto3" json:"slot,omitempty"`
	// Struct Features
	MiningSystem              uint64 `protobuf:"varint,12,opt,name=miningSystem,proto3" json:"miningSystem,omitempty"`
	RefiningSystem            uint64 `protobuf:"varint,13,opt,name=refiningSystem,proto3" json:"refiningSystem,omitempty"`
	PowerSystem               uint64 `protobuf:"varint,14,opt,name=powerSystem,proto3" json:"powerSystem,omitempty"`
	BuildStartBlock           uint64 `protobuf:"varint,15,opt,name=buildStartBlock,proto3" json:"buildStartBlock,omitempty"`
	PassiveDraw               uint64 `protobuf:"varint,16,opt,name=passiveDraw,proto3" json:"passiveDraw,omitempty"`
	ActiveMiningSystemDraw    uint64 `protobuf:"varint,17,opt,name=activeMiningSystemDraw,proto3" json:"activeMiningSystemDraw,omitempty"`
	ActiveMiningSystemBlock   uint64 `protobuf:"varint,18,opt,name=activeMiningSystemBlock,proto3" json:"activeMiningSystemBlock,omitempty"`
	ActiveRefiningSystemDraw  uint64 `protobuf:"varint,19,opt,name=activeRefiningSystemDraw,proto3" json:"activeRefiningSystemDraw,omitempty"`
	ActiveRefiningSystemBlock uint64 `protobuf:"varint,20,opt,name=activeRefiningSystemBlock,proto3" json:"activeRefiningSystemBlock,omitempty"`
	MiningSystemStatus        string `protobuf:"bytes,21,opt,name=miningSystemStatus,proto3" json:"miningSystemStatus,omitempty"`
	RefiningSystemStatus      string `protobuf:"bytes,22,opt,name=refiningSystemStatus,proto3" json:"refiningSystemStatus,omitempty"`
	PowerSystemFuel           uint64 `protobuf:"varint,23,opt,name=powerSystemFuel,proto3" json:"powerSystemFuel,omitempty"`
	PowerSystemCapacity       uint64 `protobuf:"varint,24,opt,name=powerSystemCapacity,proto3" json:"powerSystemCapacity,omitempty"`
	PowerSystemLoad           uint64 `protobuf:"varint,25,opt,name=powerSystemLoad,proto3" json:"powerSystemLoad,omitempty"`
}

func (m *Struct) Reset()         { *m = Struct{} }
func (m *Struct) String() string { return proto.CompactTextString(m) }
func (*Struct) ProtoMessage()    {}
func (*Struct) Descriptor() ([]byte, []int) {
	return fileDescriptor_c62b965c884df764, []int{0}
}
func (m *Struct) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Struct) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Struct.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Struct) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Struct.Merge(m, src)
}
func (m *Struct) XXX_Size() int {
	return m.Size()
}
func (m *Struct) XXX_DiscardUnknown() {
	xxx_messageInfo_Struct.DiscardUnknown(m)
}

var xxx_messageInfo_Struct proto.InternalMessageInfo

func (m *Struct) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Struct) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

func (m *Struct) GetOwner() string {
	if m != nil {
		return m.Owner
	}
	return ""
}

func (m *Struct) GetStatus() string {
	if m != nil {
		return m.Status
	}
	return ""
}

func (m *Struct) GetMaxHealth() uint64 {
	if m != nil {
		return m.MaxHealth
	}
	return 0
}

func (m *Struct) GetHealth() uint64 {
	if m != nil {
		return m.Health
	}
	return 0
}

func (m *Struct) GetCategory() string {
	if m != nil {
		return m.Category
	}
	return ""
}

func (m *Struct) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

func (m *Struct) GetAmbit() string {
	if m != nil {
		return m.Ambit
	}
	return ""
}

func (m *Struct) GetPlanetId() string {
	if m != nil {
		return m.PlanetId
	}
	return ""
}

func (m *Struct) GetSlot() uint64 {
	if m != nil {
		return m.Slot
	}
	return 0
}

func (m *Struct) GetMiningSystem() uint64 {
	if m != nil {
		return m.MiningSystem
	}
	return 0
}

func (m *Struct) GetRefiningSystem() uint64 {
	if m != nil {
		return m.RefiningSystem
	}
	return 0
}

func (m *Struct) GetPowerSystem() uint64 {
	if m != nil {
		return m.PowerSystem
	}
	return 0
}

func (m *Struct) GetBuildStartBlock() uint64 {
	if m != nil {
		return m.BuildStartBlock
	}
	return 0
}

func (m *Struct) GetPassiveDraw() uint64 {
	if m != nil {
		return m.PassiveDraw
	}
	return 0
}

func (m *Struct) GetActiveMiningSystemDraw() uint64 {
	if m != nil {
		return m.ActiveMiningSystemDraw
	}
	return 0
}

func (m *Struct) GetActiveMiningSystemBlock() uint64 {
	if m != nil {
		return m.ActiveMiningSystemBlock
	}
	return 0
}

func (m *Struct) GetActiveRefiningSystemDraw() uint64 {
	if m != nil {
		return m.ActiveRefiningSystemDraw
	}
	return 0
}

func (m *Struct) GetActiveRefiningSystemBlock() uint64 {
	if m != nil {
		return m.ActiveRefiningSystemBlock
	}
	return 0
}

func (m *Struct) GetMiningSystemStatus() string {
	if m != nil {
		return m.MiningSystemStatus
	}
	return ""
}

func (m *Struct) GetRefiningSystemStatus() string {
	if m != nil {
		return m.RefiningSystemStatus
	}
	return ""
}

func (m *Struct) GetPowerSystemFuel() uint64 {
	if m != nil {
		return m.PowerSystemFuel
	}
	return 0
}

func (m *Struct) GetPowerSystemCapacity() uint64 {
	if m != nil {
		return m.PowerSystemCapacity
	}
	return 0
}

func (m *Struct) GetPowerSystemLoad() uint64 {
	if m != nil {
		return m.PowerSystemLoad
	}
	return 0
}

func init() {
	proto.RegisterType((*Struct)(nil), "structs.structs.Struct")
}

func init() { proto.RegisterFile("structs/structs/struct.proto", fileDescriptor_c62b965c884df764) }

var fileDescriptor_c62b965c884df764 = []byte{
	// 468 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x53, 0xcd, 0x6e, 0xd3, 0x40,
	0x10, 0x8e, 0x43, 0x9a, 0x34, 0xd3, 0x92, 0xc0, 0x34, 0x24, 0x53, 0x54, 0x59, 0x51, 0x0f, 0x28,
	0xa7, 0xf0, 0x27, 0x21, 0x84, 0x38, 0x15, 0x84, 0x40, 0x82, 0x4b, 0x72, 0xe3, 0xb6, 0xb1, 0x97,
	0x76, 0x85, 0x93, 0xb5, 0x76, 0x37, 0x4d, 0xf3, 0x16, 0x3c, 0x00, 0x0f, 0xc4, 0xb1, 0x47, 0x8e,
	0x28, 0x79, 0x11, 0xe4, 0x59, 0xa7, 0x75, 0xfe, 0x4e, 0xbb, 0xdf, 0xcf, 0x7c, 0x3b, 0xd6, 0x78,
	0xe0, 0xcc, 0x3a, 0x33, 0x8d, 0x9c, 0x7d, 0xbe, 0x7e, 0xf6, 0x53, 0xa3, 0x9d, 0xc6, 0x66, 0xce,
	0xf6, 0xf3, 0xf3, 0xfc, 0x77, 0x0d, 0xaa, 0x43, 0xbe, 0x63, 0x03, 0xca, 0x2a, 0xa6, 0xa0, 0x1b,
	0xf4, 0xea, 0x83, 0xb2, 0x8a, 0x91, 0xa0, 0x16, 0x19, 0x29, 0x9c, 0x36, 0x54, 0x66, 0x72, 0x05,
	0xb1, 0x05, 0x07, 0x7a, 0x36, 0x91, 0x86, 0x1e, 0x30, 0xef, 0x01, 0xb6, 0xa1, 0x6a, 0x9d, 0x70,
	0x53, 0x4b, 0x15, 0xa6, 0x73, 0x84, 0x67, 0x50, 0x1f, 0x8b, 0x9b, 0xcf, 0x52, 0x24, 0xee, 0x8a,
	0x0e, 0xba, 0x41, 0xaf, 0x32, 0xb8, 0x27, 0xb2, 0xaa, 0x2b, 0x2f, 0x55, 0x59, 0xca, 0x11, 0x3e,
	0x85, 0xc3, 0x48, 0x38, 0x79, 0xa9, 0xcd, 0x9c, 0x6a, 0x9c, 0x77, 0x87, 0x11, 0xa1, 0xe2, 0xe6,
	0xa9, 0xa4, 0x43, 0xe6, 0xf9, 0x9e, 0xf5, 0x24, 0xc6, 0x23, 0xe5, 0xa8, 0xee, 0x7b, 0x62, 0x90,
	0xa5, 0xa4, 0x89, 0x98, 0x48, 0xf7, 0x25, 0x26, 0xf0, 0x29, 0x2b, 0x9c, 0xa5, 0xd8, 0x44, 0x3b,
	0x3a, 0xe2, 0x77, 0xf9, 0x8e, 0xe7, 0x70, 0x3c, 0x56, 0x13, 0x35, 0xb9, 0x1c, 0xce, 0xad, 0x93,
	0x63, 0x3a, 0x66, 0x6d, 0x8d, 0xc3, 0x67, 0xd0, 0x30, 0xf2, 0x47, 0xd1, 0xf5, 0x90, 0x5d, 0x1b,
	0x2c, 0x76, 0xe1, 0x28, 0xd5, 0x33, 0x69, 0x72, 0x53, 0x83, 0x4d, 0x45, 0x0a, 0x7b, 0xd0, 0x1c,
	0x4d, 0x55, 0x12, 0x0f, 0x9d, 0x30, 0xee, 0x22, 0xd1, 0xd1, 0x4f, 0x6a, 0xb2, 0x6b, 0x93, 0xe6,
	0x2c, 0x61, 0xad, 0xba, 0x96, 0x1f, 0x8d, 0x98, 0xd1, 0xa3, 0x3c, 0xeb, 0x9e, 0xc2, 0x37, 0xd0,
	0x16, 0x91, 0x53, 0xd7, 0xf2, 0x5b, 0xa1, 0x07, 0x36, 0x3f, 0x66, 0xf3, 0x1e, 0x15, 0xdf, 0x42,
	0x67, 0x5b, 0xf1, 0xbd, 0x20, 0x17, 0xee, 0x93, 0xf1, 0x1d, 0x90, 0x97, 0x06, 0x6b, 0xdf, 0xcd,
	0x6f, 0x9e, 0x70, 0xe9, 0x5e, 0x1d, 0xdf, 0xc3, 0xe9, 0x2e, 0xcd, 0xbf, 0xdb, 0xe2, 0xe2, 0xfd,
	0x06, 0xec, 0x03, 0x16, 0x27, 0x32, 0xf4, 0x7f, 0xdd, 0x13, 0x9e, 0xef, 0x0e, 0x05, 0x5f, 0x41,
	0x6b, 0x7d, 0x36, 0x79, 0x45, 0x9b, 0x2b, 0x76, 0x6a, 0xd9, 0x6c, 0x0a, 0xa3, 0xfa, 0x34, 0x95,
	0x09, 0x75, 0xfc, 0x6c, 0x36, 0x68, 0x7c, 0x01, 0x27, 0x05, 0xea, 0x83, 0x48, 0x45, 0xa4, 0xdc,
	0x9c, 0x88, 0xdd, 0xbb, 0xa4, 0x8d, 0xec, 0xaf, 0x5a, 0xc4, 0x74, 0xba, 0x95, 0x9d, 0xd1, 0x17,
	0x2f, 0xff, 0x2c, 0xc2, 0xe0, 0x76, 0x11, 0x06, 0xff, 0x16, 0x61, 0xf0, 0x6b, 0x19, 0x96, 0x6e,
	0x97, 0x61, 0xe9, 0xef, 0x32, 0x2c, 0x7d, 0xef, 0xac, 0xf6, 0xfb, 0xe6, 0x6e, 0xd3, 0xb3, 0x3d,
	0xb0, 0xa3, 0x2a, 0x6f, 0xfa, 0xeb, 0xff, 0x01, 0x00, 0x00, 0xff, 0xff, 0x7e, 0x74, 0x9a, 0x6d,
	0x09, 0x04, 0x00, 0x00,
}

func (m *Struct) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Struct) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Struct) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.PowerSystemLoad != 0 {
		i = encodeVarintStruct(dAtA, i, uint64(m.PowerSystemLoad))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xc8
	}
	if m.PowerSystemCapacity != 0 {
		i = encodeVarintStruct(dAtA, i, uint64(m.PowerSystemCapacity))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xc0
	}
	if m.PowerSystemFuel != 0 {
		i = encodeVarintStruct(dAtA, i, uint64(m.PowerSystemFuel))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xb8
	}
	if len(m.RefiningSystemStatus) > 0 {
		i -= len(m.RefiningSystemStatus)
		copy(dAtA[i:], m.RefiningSystemStatus)
		i = encodeVarintStruct(dAtA, i, uint64(len(m.RefiningSystemStatus)))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xb2
	}
	if len(m.MiningSystemStatus) > 0 {
		i -= len(m.MiningSystemStatus)
		copy(dAtA[i:], m.MiningSystemStatus)
		i = encodeVarintStruct(dAtA, i, uint64(len(m.MiningSystemStatus)))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xaa
	}
	if m.ActiveRefiningSystemBlock != 0 {
		i = encodeVarintStruct(dAtA, i, uint64(m.ActiveRefiningSystemBlock))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xa0
	}
	if m.ActiveRefiningSystemDraw != 0 {
		i = encodeVarintStruct(dAtA, i, uint64(m.ActiveRefiningSystemDraw))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x98
	}
	if m.ActiveMiningSystemBlock != 0 {
		i = encodeVarintStruct(dAtA, i, uint64(m.ActiveMiningSystemBlock))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x90
	}
	if m.ActiveMiningSystemDraw != 0 {
		i = encodeVarintStruct(dAtA, i, uint64(m.ActiveMiningSystemDraw))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x88
	}
	if m.PassiveDraw != 0 {
		i = encodeVarintStruct(dAtA, i, uint64(m.PassiveDraw))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x80
	}
	if m.BuildStartBlock != 0 {
		i = encodeVarintStruct(dAtA, i, uint64(m.BuildStartBlock))
		i--
		dAtA[i] = 0x78
	}
	if m.PowerSystem != 0 {
		i = encodeVarintStruct(dAtA, i, uint64(m.PowerSystem))
		i--
		dAtA[i] = 0x70
	}
	if m.RefiningSystem != 0 {
		i = encodeVarintStruct(dAtA, i, uint64(m.RefiningSystem))
		i--
		dAtA[i] = 0x68
	}
	if m.MiningSystem != 0 {
		i = encodeVarintStruct(dAtA, i, uint64(m.MiningSystem))
		i--
		dAtA[i] = 0x60
	}
	if m.Slot != 0 {
		i = encodeVarintStruct(dAtA, i, uint64(m.Slot))
		i--
		dAtA[i] = 0x58
	}
	if len(m.PlanetId) > 0 {
		i -= len(m.PlanetId)
		copy(dAtA[i:], m.PlanetId)
		i = encodeVarintStruct(dAtA, i, uint64(len(m.PlanetId)))
		i--
		dAtA[i] = 0x52
	}
	if len(m.Ambit) > 0 {
		i -= len(m.Ambit)
		copy(dAtA[i:], m.Ambit)
		i = encodeVarintStruct(dAtA, i, uint64(len(m.Ambit)))
		i--
		dAtA[i] = 0x4a
	}
	if len(m.Type) > 0 {
		i -= len(m.Type)
		copy(dAtA[i:], m.Type)
		i = encodeVarintStruct(dAtA, i, uint64(len(m.Type)))
		i--
		dAtA[i] = 0x42
	}
	if len(m.Category) > 0 {
		i -= len(m.Category)
		copy(dAtA[i:], m.Category)
		i = encodeVarintStruct(dAtA, i, uint64(len(m.Category)))
		i--
		dAtA[i] = 0x3a
	}
	if m.Health != 0 {
		i = encodeVarintStruct(dAtA, i, uint64(m.Health))
		i--
		dAtA[i] = 0x30
	}
	if m.MaxHealth != 0 {
		i = encodeVarintStruct(dAtA, i, uint64(m.MaxHealth))
		i--
		dAtA[i] = 0x28
	}
	if len(m.Status) > 0 {
		i -= len(m.Status)
		copy(dAtA[i:], m.Status)
		i = encodeVarintStruct(dAtA, i, uint64(len(m.Status)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.Owner) > 0 {
		i -= len(m.Owner)
		copy(dAtA[i:], m.Owner)
		i = encodeVarintStruct(dAtA, i, uint64(len(m.Owner)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintStruct(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintStruct(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintStruct(dAtA []byte, offset int, v uint64) int {
	offset -= sovStruct(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Struct) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovStruct(uint64(l))
	}
	l = len(m.Creator)
	if l > 0 {
		n += 1 + l + sovStruct(uint64(l))
	}
	l = len(m.Owner)
	if l > 0 {
		n += 1 + l + sovStruct(uint64(l))
	}
	l = len(m.Status)
	if l > 0 {
		n += 1 + l + sovStruct(uint64(l))
	}
	if m.MaxHealth != 0 {
		n += 1 + sovStruct(uint64(m.MaxHealth))
	}
	if m.Health != 0 {
		n += 1 + sovStruct(uint64(m.Health))
	}
	l = len(m.Category)
	if l > 0 {
		n += 1 + l + sovStruct(uint64(l))
	}
	l = len(m.Type)
	if l > 0 {
		n += 1 + l + sovStruct(uint64(l))
	}
	l = len(m.Ambit)
	if l > 0 {
		n += 1 + l + sovStruct(uint64(l))
	}
	l = len(m.PlanetId)
	if l > 0 {
		n += 1 + l + sovStruct(uint64(l))
	}
	if m.Slot != 0 {
		n += 1 + sovStruct(uint64(m.Slot))
	}
	if m.MiningSystem != 0 {
		n += 1 + sovStruct(uint64(m.MiningSystem))
	}
	if m.RefiningSystem != 0 {
		n += 1 + sovStruct(uint64(m.RefiningSystem))
	}
	if m.PowerSystem != 0 {
		n += 1 + sovStruct(uint64(m.PowerSystem))
	}
	if m.BuildStartBlock != 0 {
		n += 1 + sovStruct(uint64(m.BuildStartBlock))
	}
	if m.PassiveDraw != 0 {
		n += 2 + sovStruct(uint64(m.PassiveDraw))
	}
	if m.ActiveMiningSystemDraw != 0 {
		n += 2 + sovStruct(uint64(m.ActiveMiningSystemDraw))
	}
	if m.ActiveMiningSystemBlock != 0 {
		n += 2 + sovStruct(uint64(m.ActiveMiningSystemBlock))
	}
	if m.ActiveRefiningSystemDraw != 0 {
		n += 2 + sovStruct(uint64(m.ActiveRefiningSystemDraw))
	}
	if m.ActiveRefiningSystemBlock != 0 {
		n += 2 + sovStruct(uint64(m.ActiveRefiningSystemBlock))
	}
	l = len(m.MiningSystemStatus)
	if l > 0 {
		n += 2 + l + sovStruct(uint64(l))
	}
	l = len(m.RefiningSystemStatus)
	if l > 0 {
		n += 2 + l + sovStruct(uint64(l))
	}
	if m.PowerSystemFuel != 0 {
		n += 2 + sovStruct(uint64(m.PowerSystemFuel))
	}
	if m.PowerSystemCapacity != 0 {
		n += 2 + sovStruct(uint64(m.PowerSystemCapacity))
	}
	if m.PowerSystemLoad != 0 {
		n += 2 + sovStruct(uint64(m.PowerSystemLoad))
	}
	return n
}

func sovStruct(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozStruct(x uint64) (n int) {
	return sovStruct(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Struct) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowStruct
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
			return fmt.Errorf("proto: Struct: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Struct: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
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
				return ErrInvalidLengthStruct
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthStruct
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Creator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
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
				return ErrInvalidLengthStruct
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthStruct
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Creator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Owner", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
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
				return ErrInvalidLengthStruct
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthStruct
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Owner = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Status", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
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
				return ErrInvalidLengthStruct
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthStruct
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Status = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaxHealth", wireType)
			}
			m.MaxHealth = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MaxHealth |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Health", wireType)
			}
			m.Health = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Health |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Category", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
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
				return ErrInvalidLengthStruct
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthStruct
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Category = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Type", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
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
				return ErrInvalidLengthStruct
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthStruct
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Type = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Ambit", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
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
				return ErrInvalidLengthStruct
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthStruct
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Ambit = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PlanetId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
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
				return ErrInvalidLengthStruct
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthStruct
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.PlanetId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 11:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Slot", wireType)
			}
			m.Slot = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Slot |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 12:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MiningSystem", wireType)
			}
			m.MiningSystem = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MiningSystem |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 13:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field RefiningSystem", wireType)
			}
			m.RefiningSystem = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.RefiningSystem |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 14:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field PowerSystem", wireType)
			}
			m.PowerSystem = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.PowerSystem |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 15:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field BuildStartBlock", wireType)
			}
			m.BuildStartBlock = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.BuildStartBlock |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 16:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field PassiveDraw", wireType)
			}
			m.PassiveDraw = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.PassiveDraw |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 17:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ActiveMiningSystemDraw", wireType)
			}
			m.ActiveMiningSystemDraw = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ActiveMiningSystemDraw |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 18:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ActiveMiningSystemBlock", wireType)
			}
			m.ActiveMiningSystemBlock = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ActiveMiningSystemBlock |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 19:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ActiveRefiningSystemDraw", wireType)
			}
			m.ActiveRefiningSystemDraw = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ActiveRefiningSystemDraw |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 20:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ActiveRefiningSystemBlock", wireType)
			}
			m.ActiveRefiningSystemBlock = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ActiveRefiningSystemBlock |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 21:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MiningSystemStatus", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
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
				return ErrInvalidLengthStruct
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthStruct
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MiningSystemStatus = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 22:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RefiningSystemStatus", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
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
				return ErrInvalidLengthStruct
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthStruct
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.RefiningSystemStatus = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 23:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field PowerSystemFuel", wireType)
			}
			m.PowerSystemFuel = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.PowerSystemFuel |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 24:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field PowerSystemCapacity", wireType)
			}
			m.PowerSystemCapacity = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.PowerSystemCapacity |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 25:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field PowerSystemLoad", wireType)
			}
			m.PowerSystemLoad = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStruct
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.PowerSystemLoad |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipStruct(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthStruct
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
func skipStruct(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowStruct
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
					return 0, ErrIntOverflowStruct
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
					return 0, ErrIntOverflowStruct
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
				return 0, ErrInvalidLengthStruct
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupStruct
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthStruct
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthStruct        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowStruct          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupStruct = fmt.Errorf("proto: unexpected end of group")
)
