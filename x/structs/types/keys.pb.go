// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: structs/structs/keys.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	math "math"
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

type ObjectType int32

const (
	ObjectType_guild      ObjectType = 0
	ObjectType_player     ObjectType = 1
	ObjectType_planet     ObjectType = 2
	ObjectType_reactor    ObjectType = 3
	ObjectType_substation ObjectType = 4
	ObjectType_struct     ObjectType = 5
	ObjectType_allocation ObjectType = 6
	ObjectType_infusion   ObjectType = 7
	ObjectType_address    ObjectType = 8
)

var ObjectType_name = map[int32]string{
	0: "guild",
	1: "player",
	2: "planet",
	3: "reactor",
	4: "substation",
	5: "struct",
	6: "allocation",
	7: "infusion",
	8: "address",
}

var ObjectType_value = map[string]int32{
	"guild":      0,
	"player":     1,
	"planet":     2,
	"reactor":    3,
	"substation": 4,
	"struct":     5,
	"allocation": 6,
	"infusion":   7,
	"address":    8,
}

func (x ObjectType) String() string {
	return proto.EnumName(ObjectType_name, int32(x))
}

func (ObjectType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{0}
}

type GridAttributeType int32

const (
	GridAttributeType_ore                    GridAttributeType = 0
	GridAttributeType_fuel                   GridAttributeType = 1
	GridAttributeType_capacity               GridAttributeType = 2
	GridAttributeType_load                   GridAttributeType = 3
	GridAttributeType_structsLoad            GridAttributeType = 4
	GridAttributeType_power                  GridAttributeType = 5
	GridAttributeType_connectionCapacity     GridAttributeType = 6
	GridAttributeType_connectionCount        GridAttributeType = 7
	GridAttributeType_allocationPointerStart GridAttributeType = 8
	GridAttributeType_allocationPointerEnd   GridAttributeType = 9
)

var GridAttributeType_name = map[int32]string{
	0: "ore",
	1: "fuel",
	2: "capacity",
	3: "load",
	4: "structsLoad",
	5: "power",
	6: "connectionCapacity",
	7: "connectionCount",
	8: "allocationPointerStart",
	9: "allocationPointerEnd",
}

var GridAttributeType_value = map[string]int32{
	"ore":                    0,
	"fuel":                   1,
	"capacity":               2,
	"load":                   3,
	"structsLoad":            4,
	"power":                  5,
	"connectionCapacity":     6,
	"connectionCount":        7,
	"allocationPointerStart": 8,
	"allocationPointerEnd":   9,
}

func (x GridAttributeType) String() string {
	return proto.EnumName(GridAttributeType_name, int32(x))
}

func (GridAttributeType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{1}
}

type AllocationType int32

const (
	AllocationType_static    AllocationType = 0
	AllocationType_dynamic   AllocationType = 1
	AllocationType_automated AllocationType = 2
)

var AllocationType_name = map[int32]string{
	0: "static",
	1: "dynamic",
	2: "automated",
}

var AllocationType_value = map[string]int32{
	"static":    0,
	"dynamic":   1,
	"automated": 2,
}

func (x AllocationType) String() string {
	return proto.EnumName(AllocationType_name, int32(x))
}

func (AllocationType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{2}
}

type GuildJoinBypassLevel int32

const (
	GuildJoinBypassLevel_closed       GuildJoinBypassLevel = 0
	GuildJoinBypassLevel_permissioned GuildJoinBypassLevel = 1
	GuildJoinBypassLevel_member       GuildJoinBypassLevel = 2
)

var GuildJoinBypassLevel_name = map[int32]string{
	0: "closed",
	1: "permissioned",
	2: "member",
}

var GuildJoinBypassLevel_value = map[string]int32{
	"closed":       0,
	"permissioned": 1,
	"member":       2,
}

func (x GuildJoinBypassLevel) String() string {
	return proto.EnumName(GuildJoinBypassLevel_name, int32(x))
}

func (GuildJoinBypassLevel) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{3}
}

type RegistrationStatus int32

const (
	RegistrationStatus_proposed RegistrationStatus = 0
	RegistrationStatus_approved RegistrationStatus = 1
	RegistrationStatus_denied   RegistrationStatus = 2
)

var RegistrationStatus_name = map[int32]string{
	0: "proposed",
	1: "approved",
	2: "denied",
}

var RegistrationStatus_value = map[string]int32{
	"proposed": 0,
	"approved": 1,
	"denied":   2,
}

func (x RegistrationStatus) String() string {
	return proto.EnumName(RegistrationStatus_name, int32(x))
}

func (RegistrationStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{4}
}

type Ambit int32

const (
	Ambit_water Ambit = 0
	Ambit_land  Ambit = 1
	Ambit_air   Ambit = 2
	Ambit_space Ambit = 3
)

var Ambit_name = map[int32]string{
	0: "water",
	1: "land",
	2: "air",
	3: "space",
}

var Ambit_value = map[string]int32{
	"water": 0,
	"land":  1,
	"air":   2,
	"space": 3,
}

func (x Ambit) String() string {
	return proto.EnumName(Ambit_name, int32(x))
}

func (Ambit) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{5}
}

type StructCategory int32

const (
	StructCategory_planetary StructCategory = 0
	StructCategory_fleet     StructCategory = 1
)

var StructCategory_name = map[int32]string{
	0: "planetary",
	1: "fleet",
}

var StructCategory_value = map[string]int32{
	"planetary": 0,
	"fleet":     1,
}

func (x StructCategory) String() string {
	return proto.EnumName(StructCategory_name, int32(x))
}

func (StructCategory) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{6}
}

type StructStatus int32

const (
	StructStatus_building  StructStatus = 0
	StructStatus_active    StructStatus = 1
	StructStatus_inactive  StructStatus = 2
	StructStatus_destroyed StructStatus = 3
)

var StructStatus_name = map[int32]string{
	0: "building",
	1: "active",
	2: "inactive",
	3: "destroyed",
}

var StructStatus_value = map[string]int32{
	"building":  0,
	"active":    1,
	"inactive":  2,
	"destroyed": 3,
}

func (x StructStatus) String() string {
	return proto.EnumName(StructStatus_name, int32(x))
}

func (StructStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{7}
}

type StructType int32

const (
	StructType_miningRig      StructType = 0
	StructType_refinery       StructType = 1
	StructType_smallGenerator StructType = 2
)

var StructType_name = map[int32]string{
	0: "miningRig",
	1: "refinery",
	2: "smallGenerator",
}

var StructType_value = map[string]int32{
	"miningRig":      0,
	"refinery":       1,
	"smallGenerator": 2,
}

func (x StructType) String() string {
	return proto.EnumName(StructType_name, int32(x))
}

func (StructType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{8}
}

func init() {
	proto.RegisterEnum("structs.structs.ObjectType", ObjectType_name, ObjectType_value)
	proto.RegisterEnum("structs.structs.GridAttributeType", GridAttributeType_name, GridAttributeType_value)
	proto.RegisterEnum("structs.structs.AllocationType", AllocationType_name, AllocationType_value)
	proto.RegisterEnum("structs.structs.GuildJoinBypassLevel", GuildJoinBypassLevel_name, GuildJoinBypassLevel_value)
	proto.RegisterEnum("structs.structs.RegistrationStatus", RegistrationStatus_name, RegistrationStatus_value)
	proto.RegisterEnum("structs.structs.Ambit", Ambit_name, Ambit_value)
	proto.RegisterEnum("structs.structs.StructCategory", StructCategory_name, StructCategory_value)
	proto.RegisterEnum("structs.structs.StructStatus", StructStatus_name, StructStatus_value)
	proto.RegisterEnum("structs.structs.StructType", StructType_name, StructType_value)
}

func init() { proto.RegisterFile("structs/structs/keys.proto", fileDescriptor_d2b5c851dc116405) }

var fileDescriptor_d2b5c851dc116405 = []byte{
	// 569 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x64, 0x53, 0xbf, 0x6e, 0xd4, 0x4e,
	0x10, 0xb6, 0x7d, 0xb9, 0x7f, 0x93, 0xcb, 0x65, 0x7f, 0xfb, 0x8b, 0x02, 0x4a, 0x71, 0x0f, 0xe0,
	0x22, 0x01, 0xd1, 0x50, 0x80, 0x04, 0x89, 0x22, 0x24, 0x94, 0x02, 0x11, 0x2a, 0xba, 0xb1, 0x3d,
	0x67, 0x2d, 0xd8, 0xbb, 0xd6, 0xec, 0x38, 0xc1, 0x35, 0x2f, 0xc0, 0x0b, 0xd1, 0x53, 0xa6, 0xa4,
	0x44, 0xc9, 0x8b, 0xa0, 0xb5, 0xef, 0x38, 0x24, 0xaa, 0xdd, 0x9d, 0x6f, 0x76, 0x66, 0xbe, 0x6f,
	0x66, 0xe0, 0xc4, 0x0b, 0xb7, 0xb9, 0xf8, 0xb3, 0xed, 0xf9, 0x99, 0x3a, 0x7f, 0xda, 0xb0, 0x13,
	0xa7, 0x0f, 0x37, 0xb6, 0xd3, 0xcd, 0x79, 0x72, 0x54, 0xba, 0xd2, 0xf5, 0xd8, 0x59, 0xb8, 0x0d,
	0x6e, 0xe9, 0xd7, 0x18, 0xc0, 0x65, 0x9f, 0x28, 0x97, 0x0f, 0x5d, 0x43, 0x7a, 0x0e, 0xe3, 0xb2,
	0x35, 0x55, 0xa1, 0x22, 0x0d, 0x30, 0x69, 0x2a, 0xec, 0x88, 0x55, 0xbc, 0xb9, 0x5b, 0x12, 0x95,
	0xe8, 0x7d, 0x98, 0x32, 0x61, 0x2e, 0x8e, 0xd5, 0x48, 0x2f, 0x01, 0x7c, 0x9b, 0x79, 0x41, 0x31,
	0xce, 0xaa, 0xbd, 0xe0, 0x38, 0xe4, 0x53, 0xe3, 0x80, 0x61, 0x55, 0xb9, 0x7c, 0xc0, 0x26, 0x7a,
	0x01, 0x33, 0x63, 0xd7, 0xad, 0x0f, 0xaf, 0x69, 0x08, 0x83, 0x45, 0xc1, 0xe4, 0xbd, 0x9a, 0xa5,
	0xdf, 0x63, 0xf8, 0xaf, 0x64, 0x53, 0xbc, 0x16, 0x61, 0x93, 0xb5, 0x42, 0x7d, 0x31, 0x53, 0x18,
	0x39, 0x26, 0x15, 0xe9, 0x19, 0xec, 0xad, 0x5b, 0xaa, 0x54, 0x1c, 0x62, 0xe4, 0xd8, 0x60, 0x6e,
	0xa4, 0x53, 0x49, 0xb0, 0x57, 0x0e, 0x0b, 0x35, 0xd2, 0x87, 0xb0, 0xbf, 0xe1, 0x79, 0x15, 0x0c,
	0x7b, 0x81, 0x48, 0xe3, 0x6e, 0x89, 0xd5, 0x58, 0x1f, 0x83, 0xce, 0x9d, 0xb5, 0x94, 0x87, 0x3a,
	0x2e, 0xb6, 0xbf, 0x27, 0xfa, 0x7f, 0x38, 0xfc, 0xcb, 0xee, 0x5a, 0x2b, 0x6a, 0xaa, 0x4f, 0xe0,
	0x78, 0x57, 0xf4, 0x3b, 0x67, 0xac, 0x10, 0x5f, 0x0b, 0xb2, 0xa8, 0x99, 0x7e, 0x0c, 0x47, 0xff,
	0x60, 0x97, 0xb6, 0x50, 0xf3, 0xf4, 0x39, 0x2c, 0x77, 0x48, 0x5f, 0x7b, 0x2f, 0x04, 0x8a, 0xc9,
	0x55, 0x14, 0xa8, 0x16, 0x9d, 0xc5, 0xda, 0xe4, 0x2a, 0xd6, 0x07, 0x30, 0xc7, 0x56, 0x5c, 0x8d,
	0x42, 0x85, 0x4a, 0xd2, 0x57, 0x70, 0xd4, 0x0b, 0xfe, 0xd6, 0x19, 0x7b, 0xde, 0x35, 0xe8, 0xfd,
	0x15, 0xdd, 0x50, 0x15, 0xfe, 0xe7, 0x95, 0xf3, 0x14, 0x3a, 0xa1, 0x60, 0xd1, 0x10, 0xd7, 0xc6,
	0x07, 0xe9, 0xa8, 0x18, 0xfa, 0x51, 0x53, 0x9d, 0x11, 0xab, 0x24, 0x7d, 0x01, 0x9a, 0xa9, 0x34,
	0x5e, 0xb8, 0xcf, 0x7e, 0x2d, 0x28, 0xad, 0x0f, 0x42, 0x35, 0xec, 0x9a, 0x4d, 0x84, 0x05, 0xcc,
	0xb0, 0x69, 0xd8, 0xdd, 0x6c, 0x7f, 0x17, 0x64, 0x4d, 0x9f, 0xff, 0x09, 0x8c, 0xb1, 0xce, 0x8c,
	0x04, 0xc1, 0x6e, 0x51, 0x88, 0x07, 0xb9, 0x2b, 0xb4, 0xc1, 0x73, 0x0a, 0x23, 0x34, 0xac, 0x92,
	0x80, 0xfa, 0x06, 0x73, 0x52, 0xa3, 0x34, 0x85, 0xe5, 0x20, 0xf5, 0x05, 0x0a, 0x95, 0x8e, 0xbb,
	0x40, 0x69, 0x98, 0x0e, 0xe4, 0x4e, 0x45, 0xc1, 0x77, 0x5d, 0x11, 0x89, 0x8a, 0xd3, 0x4b, 0x58,
	0x0c, 0xbe, 0xbb, 0xaa, 0xb2, 0xc0, 0xd6, 0xd8, 0x72, 0x98, 0x30, 0xcc, 0xc5, 0xdc, 0xd0, 0xd0,
	0x58, 0x63, 0x37, 0xaf, 0x24, 0x44, 0x2c, 0xc8, 0x0b, 0xbb, 0x8e, 0x0a, 0x35, 0x4a, 0x5f, 0x02,
	0x0c, 0x61, 0x7a, 0x69, 0x0f, 0x60, 0x5e, 0x1b, 0x6b, 0x6c, 0xf9, 0xde, 0x94, 0x03, 0x37, 0xa6,
	0xb5, 0xb1, 0xc4, 0x9d, 0x8a, 0xb5, 0x86, 0xa5, 0xaf, 0xb1, 0xaa, 0xde, 0x90, 0x25, 0xc6, 0x30,
	0xa4, 0xc9, 0xf9, 0xd3, 0x1f, 0xf7, 0xab, 0xf8, 0xee, 0x7e, 0x15, 0xff, 0xba, 0x5f, 0xc5, 0xdf,
	0x1e, 0x56, 0xd1, 0xdd, 0xc3, 0x2a, 0xfa, 0xf9, 0xb0, 0x8a, 0x3e, 0x3e, 0xda, 0x2e, 0xce, 0x97,
	0x3f, 0x2b, 0x24, 0x5d, 0x43, 0x3e, 0x9b, 0xf4, 0xdb, 0xf1, 0xec, 0x77, 0x00, 0x00, 0x00, 0xff,
	0xff, 0xf3, 0xf0, 0xe3, 0x5d, 0x62, 0x03, 0x00, 0x00,
}
