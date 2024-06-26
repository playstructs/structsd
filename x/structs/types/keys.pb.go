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
	GridAttributeType_proxyNonce             GridAttributeType = 10
)

var GridAttributeType_name = map[int32]string{
	0:  "ore",
	1:  "fuel",
	2:  "capacity",
	3:  "load",
	4:  "structsLoad",
	5:  "power",
	6:  "connectionCapacity",
	7:  "connectionCount",
	8:  "allocationPointerStart",
	9:  "allocationPointerEnd",
	10: "proxyNonce",
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
	"proxyNonce":             10,
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

type GuildJoinType int32

const (
	GuildJoinType_invite  GuildJoinType = 0
	GuildJoinType_request GuildJoinType = 1
	GuildJoinType_direct  GuildJoinType = 2
	GuildJoinType_proxy   GuildJoinType = 3
)

var GuildJoinType_name = map[int32]string{
	0: "invite",
	1: "request",
	2: "direct",
	3: "proxy",
}

var GuildJoinType_value = map[string]int32{
	"invite":  0,
	"request": 1,
	"direct":  2,
	"proxy":   3,
}

func (x GuildJoinType) String() string {
	return proto.EnumName(GuildJoinType_name, int32(x))
}

func (GuildJoinType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{4}
}

type RegistrationStatus int32

const (
	RegistrationStatus_proposed RegistrationStatus = 0
	RegistrationStatus_approved RegistrationStatus = 1
	RegistrationStatus_denied   RegistrationStatus = 2
	RegistrationStatus_revoked  RegistrationStatus = 3
)

var RegistrationStatus_name = map[int32]string{
	0: "proposed",
	1: "approved",
	2: "denied",
	3: "revoked",
}

var RegistrationStatus_value = map[string]int32{
	"proposed": 0,
	"approved": 1,
	"denied":   2,
	"revoked":  3,
}

func (x RegistrationStatus) String() string {
	return proto.EnumName(RegistrationStatus_name, int32(x))
}

func (RegistrationStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{5}
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
	return fileDescriptor_d2b5c851dc116405, []int{6}
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
	return fileDescriptor_d2b5c851dc116405, []int{7}
}

type StructStatus int32

const (
	StructStatus_building  StructStatus = 0
	StructStatus_online    StructStatus = 1
	StructStatus_offline   StructStatus = 2
	StructStatus_destroyed StructStatus = 3
)

var StructStatus_name = map[int32]string{
	0: "building",
	1: "online",
	2: "offline",
	3: "destroyed",
}

var StructStatus_value = map[string]int32{
	"building":  0,
	"online":    1,
	"offline":   2,
	"destroyed": 3,
}

func (x StructStatus) String() string {
	return proto.EnumName(StructStatus_name, int32(x))
}

func (StructStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{8}
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
	return fileDescriptor_d2b5c851dc116405, []int{9}
}

type PlanetStatus int32

const (
	PlanetStatus_active   PlanetStatus = 0
	PlanetStatus_complete PlanetStatus = 1
)

var PlanetStatus_name = map[int32]string{
	0: "active",
	1: "complete",
}

var PlanetStatus_value = map[string]int32{
	"active":   0,
	"complete": 1,
}

func (x PlanetStatus) String() string {
	return proto.EnumName(PlanetStatus_name, int32(x))
}

func (PlanetStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{10}
}

func init() {
	proto.RegisterEnum("structs.structs.ObjectType", ObjectType_name, ObjectType_value)
	proto.RegisterEnum("structs.structs.GridAttributeType", GridAttributeType_name, GridAttributeType_value)
	proto.RegisterEnum("structs.structs.AllocationType", AllocationType_name, AllocationType_value)
	proto.RegisterEnum("structs.structs.GuildJoinBypassLevel", GuildJoinBypassLevel_name, GuildJoinBypassLevel_value)
	proto.RegisterEnum("structs.structs.GuildJoinType", GuildJoinType_name, GuildJoinType_value)
	proto.RegisterEnum("structs.structs.RegistrationStatus", RegistrationStatus_name, RegistrationStatus_value)
	proto.RegisterEnum("structs.structs.Ambit", Ambit_name, Ambit_value)
	proto.RegisterEnum("structs.structs.StructCategory", StructCategory_name, StructCategory_value)
	proto.RegisterEnum("structs.structs.StructStatus", StructStatus_name, StructStatus_value)
	proto.RegisterEnum("structs.structs.StructType", StructType_name, StructType_value)
	proto.RegisterEnum("structs.structs.PlanetStatus", PlanetStatus_name, PlanetStatus_value)
}

func init() { proto.RegisterFile("structs/structs/keys.proto", fileDescriptor_d2b5c851dc116405) }

var fileDescriptor_d2b5c851dc116405 = []byte{
	// 638 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x64, 0x53, 0xcd, 0x6e, 0x14, 0x39,
	0x10, 0x9e, 0x9e, 0xc9, 0xfc, 0x55, 0x26, 0x13, 0xaf, 0x37, 0xca, 0xae, 0x72, 0x98, 0xf3, 0xaa,
	0x0f, 0xc9, 0xae, 0xf6, 0xc2, 0x05, 0x01, 0x09, 0x08, 0x81, 0x22, 0x84, 0x08, 0x27, 0x6e, 0x1e,
	0x77, 0x4d, 0xcb, 0xa4, 0xdb, 0x36, 0xe5, 0xea, 0x49, 0xfa, 0xcc, 0x0b, 0xf0, 0x58, 0x5c, 0x90,
	0x72, 0xe4, 0x88, 0x92, 0x17, 0x41, 0x6e, 0xcf, 0x24, 0x48, 0x9c, 0x6c, 0xd7, 0xef, 0xf7, 0xd5,
	0xe7, 0x82, 0xa3, 0xc0, 0xd4, 0x68, 0x0e, 0x27, 0xdb, 0xf3, 0x12, 0xdb, 0x70, 0xec, 0xc9, 0xb1,
	0x93, 0xfb, 0x1b, 0xdb, 0xf1, 0xe6, 0x3c, 0x3a, 0x28, 0x5d, 0xe9, 0x3a, 0xdf, 0x49, 0xbc, 0xa5,
	0xb0, 0xfc, 0x73, 0x06, 0xe0, 0x96, 0x1f, 0x51, 0xf3, 0xfb, 0xd6, 0xa3, 0x9c, 0xc2, 0xb0, 0x6c,
	0x4c, 0x55, 0x88, 0x9e, 0x04, 0x18, 0xf9, 0x4a, 0xb5, 0x48, 0x22, 0xdb, 0xdc, 0x2d, 0xb2, 0xe8,
	0xcb, 0x5d, 0x18, 0x13, 0x2a, 0xcd, 0x8e, 0xc4, 0x40, 0xce, 0x01, 0x42, 0xb3, 0x0c, 0xac, 0xd8,
	0x38, 0x2b, 0x76, 0x62, 0x60, 0xea, 0x27, 0x86, 0xd1, 0xa7, 0xaa, 0xca, 0xe9, 0xe4, 0x1b, 0xc9,
	0x19, 0x4c, 0x8c, 0x5d, 0x35, 0x21, 0xbe, 0xc6, 0xb1, 0x8c, 0x2a, 0x0a, 0xc2, 0x10, 0xc4, 0x24,
	0xff, 0x96, 0xc1, 0x1f, 0x25, 0x99, 0xe2, 0x19, 0x33, 0x99, 0x65, 0xc3, 0xd8, 0x81, 0x19, 0xc3,
	0xc0, 0x11, 0x8a, 0x9e, 0x9c, 0xc0, 0xce, 0xaa, 0xc1, 0x4a, 0x64, 0xb1, 0x86, 0x56, 0x5e, 0x69,
	0xc3, 0xad, 0xe8, 0x47, 0x7b, 0xe5, 0x54, 0x21, 0x06, 0x72, 0x1f, 0x76, 0x37, 0x3c, 0xcf, 0xa3,
	0x61, 0x27, 0x12, 0xf1, 0xee, 0x0a, 0x49, 0x0c, 0xe5, 0x21, 0x48, 0xed, 0xac, 0x45, 0x1d, 0x71,
	0x9c, 0x6d, 0xb3, 0x47, 0xf2, 0x4f, 0xd8, 0xff, 0xc5, 0xee, 0x1a, 0xcb, 0x62, 0x2c, 0x8f, 0xe0,
	0xf0, 0x01, 0xf4, 0x5b, 0x67, 0x2c, 0x23, 0x5d, 0xb0, 0x22, 0x16, 0x13, 0xf9, 0x37, 0x1c, 0xfc,
	0xe6, 0x7b, 0x61, 0x0b, 0x31, 0x8d, 0x54, 0x3d, 0xb9, 0xeb, 0xf6, 0x8d, 0xb3, 0x1a, 0x05, 0xe4,
	0x8f, 0x60, 0xfe, 0x10, 0xd9, 0x71, 0xe9, 0x06, 0xa3, 0xd8, 0x68, 0xd1, 0x8b, 0xd4, 0x8b, 0xd6,
	0xaa, 0xda, 0x68, 0x91, 0xc9, 0x3d, 0x98, 0xaa, 0x86, 0x5d, 0xad, 0x18, 0x0b, 0xd1, 0xcf, 0x9f,
	0xc2, 0x41, 0x27, 0xc0, 0x6b, 0x67, 0xec, 0x69, 0xeb, 0x55, 0x08, 0xe7, 0xb8, 0xc6, 0x2a, 0xe6,
	0xeb, 0xca, 0x05, 0x8c, 0xca, 0x08, 0x98, 0x79, 0xa4, 0xda, 0x84, 0x38, 0x4a, 0x2c, 0x92, 0x3e,
	0x35, 0xd6, 0x4b, 0x24, 0xd1, 0xcf, 0x9f, 0xc0, 0xde, 0x7d, 0x85, 0x6d, 0x6b, 0x63, 0xd7, 0x86,
	0x31, 0xb5, 0x26, 0xfc, 0xd4, 0x60, 0xe0, 0x94, 0x55, 0x18, 0x42, 0x1d, 0x55, 0x8d, 0xf3, 0x8a,
	0x0c, 0xc4, 0x20, 0x7f, 0x05, 0x92, 0xb0, 0x34, 0x81, 0xa9, 0x83, 0x7f, 0xc1, 0x8a, 0x9b, 0x10,
	0x27, 0xef, 0xc9, 0xf9, 0x0d, 0x84, 0x19, 0x4c, 0x94, 0xf7, 0xe4, 0xd6, 0xdb, 0xf6, 0x05, 0x5a,
	0x13, 0x09, 0xa4, 0x0e, 0x6b, 0x77, 0x89, 0x85, 0x18, 0xe4, 0xff, 0xc2, 0x50, 0xd5, 0x4b, 0xc3,
	0xb1, 0xfc, 0x95, 0x62, 0xa4, 0x24, 0x66, 0xa5, 0x6c, 0x4c, 0x1b, 0xc3, 0x40, 0x19, 0x4a, 0xcd,
	0x83, 0x57, 0x1a, 0xc5, 0x20, 0xcf, 0x61, 0x9e, 0x84, 0x3c, 0x53, 0x8c, 0xa5, 0xa3, 0x36, 0x0e,
	0x28, 0xfd, 0x3d, 0x45, 0xad, 0xe8, 0xc5, 0xd8, 0x55, 0x85, 0xc8, 0x22, 0xcb, 0x9f, 0xc3, 0x2c,
	0xc5, 0x3e, 0x40, 0x5c, 0x46, 0xe6, 0xc6, 0x96, 0xe9, 0xff, 0x3a, 0x5b, 0x19, 0x8b, 0x22, 0x8b,
	0xa0, 0xdc, 0x6a, 0xd5, 0x3d, 0xfa, 0xb1, 0x60, 0x81, 0x81, 0xc9, 0xb5, 0x1d, 0xc6, 0xc7, 0x00,
	0xa9, 0x4a, 0x37, 0xac, 0x3d, 0x98, 0xd6, 0xc6, 0x1a, 0x5b, 0xbe, 0x33, 0x65, 0xe2, 0x49, 0xb8,
	0x32, 0x16, 0xa9, 0x15, 0x99, 0x94, 0x30, 0x0f, 0xb5, 0xaa, 0xaa, 0x97, 0x68, 0x91, 0x54, 0xdc,
	0x80, 0x7e, 0xfe, 0x0f, 0xcc, 0x12, 0xbc, 0x0d, 0x08, 0x80, 0x91, 0xd2, 0x6c, 0xd6, 0x98, 0xb2,
	0xb5, 0xab, 0x7d, 0x85, 0x8c, 0x22, 0x3b, 0xfd, 0xef, 0xeb, 0xed, 0x22, 0xbb, 0xb9, 0x5d, 0x64,
	0x3f, 0x6e, 0x17, 0xd9, 0x97, 0xbb, 0x45, 0xef, 0xe6, 0x6e, 0xd1, 0xfb, 0x7e, 0xb7, 0xe8, 0x7d,
	0xf8, 0x6b, 0xbb, 0xbf, 0xd7, 0xf7, 0x9b, 0xcc, 0xad, 0xc7, 0xb0, 0x1c, 0x75, 0x4b, 0xfa, 0xff,
	0xcf, 0x00, 0x00, 0x00, 0xff, 0xff, 0x55, 0xe5, 0x1c, 0xb6, 0xe9, 0x03, 0x00, 0x00,
}
