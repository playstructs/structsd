// Code generated by protoc-gen-go-pulsar. DO NOT EDIT.
package structs

import (
	_ "github.com/cosmos/gogoproto/gogoproto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.0
// 	protoc        (unknown)
// source: structs/structs/keys.proto

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

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

// Enum value maps for ObjectType.
var (
	ObjectType_name = map[int32]string{
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
	ObjectType_value = map[string]int32{
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
)

func (x ObjectType) Enum() *ObjectType {
	p := new(ObjectType)
	*p = x
	return p
}

func (x ObjectType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ObjectType) Descriptor() protoreflect.EnumDescriptor {
	return file_structs_structs_keys_proto_enumTypes[0].Descriptor()
}

func (ObjectType) Type() protoreflect.EnumType {
	return &file_structs_structs_keys_proto_enumTypes[0]
}

func (x ObjectType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ObjectType.Descriptor instead.
func (ObjectType) EnumDescriptor() ([]byte, []int) {
	return file_structs_structs_keys_proto_rawDescGZIP(), []int{0}
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

// Enum value maps for GridAttributeType.
var (
	GridAttributeType_name = map[int32]string{
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
	GridAttributeType_value = map[string]int32{
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
)

func (x GridAttributeType) Enum() *GridAttributeType {
	p := new(GridAttributeType)
	*p = x
	return p
}

func (x GridAttributeType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (GridAttributeType) Descriptor() protoreflect.EnumDescriptor {
	return file_structs_structs_keys_proto_enumTypes[1].Descriptor()
}

func (GridAttributeType) Type() protoreflect.EnumType {
	return &file_structs_structs_keys_proto_enumTypes[1]
}

func (x GridAttributeType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use GridAttributeType.Descriptor instead.
func (GridAttributeType) EnumDescriptor() ([]byte, []int) {
	return file_structs_structs_keys_proto_rawDescGZIP(), []int{1}
}

type AllocationType int32

const (
	AllocationType_static    AllocationType = 0
	AllocationType_dynamic   AllocationType = 1
	AllocationType_automated AllocationType = 2
)

// Enum value maps for AllocationType.
var (
	AllocationType_name = map[int32]string{
		0: "static",
		1: "dynamic",
		2: "automated",
	}
	AllocationType_value = map[string]int32{
		"static":    0,
		"dynamic":   1,
		"automated": 2,
	}
)

func (x AllocationType) Enum() *AllocationType {
	p := new(AllocationType)
	*p = x
	return p
}

func (x AllocationType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (AllocationType) Descriptor() protoreflect.EnumDescriptor {
	return file_structs_structs_keys_proto_enumTypes[2].Descriptor()
}

func (AllocationType) Type() protoreflect.EnumType {
	return &file_structs_structs_keys_proto_enumTypes[2]
}

func (x AllocationType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use AllocationType.Descriptor instead.
func (AllocationType) EnumDescriptor() ([]byte, []int) {
	return file_structs_structs_keys_proto_rawDescGZIP(), []int{2}
}

type GuildJoinBypassLevel int32

const (
	GuildJoinBypassLevel_closed       GuildJoinBypassLevel = 0 // Feature off
	GuildJoinBypassLevel_permissioned GuildJoinBypassLevel = 1 // Only those with permissions can do it
	GuildJoinBypassLevel_member       GuildJoinBypassLevel = 2 // All members of the guild can contribute
)

// Enum value maps for GuildJoinBypassLevel.
var (
	GuildJoinBypassLevel_name = map[int32]string{
		0: "closed",
		1: "permissioned",
		2: "member",
	}
	GuildJoinBypassLevel_value = map[string]int32{
		"closed":       0,
		"permissioned": 1,
		"member":       2,
	}
)

func (x GuildJoinBypassLevel) Enum() *GuildJoinBypassLevel {
	p := new(GuildJoinBypassLevel)
	*p = x
	return p
}

func (x GuildJoinBypassLevel) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (GuildJoinBypassLevel) Descriptor() protoreflect.EnumDescriptor {
	return file_structs_structs_keys_proto_enumTypes[3].Descriptor()
}

func (GuildJoinBypassLevel) Type() protoreflect.EnumType {
	return &file_structs_structs_keys_proto_enumTypes[3]
}

func (x GuildJoinBypassLevel) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use GuildJoinBypassLevel.Descriptor instead.
func (GuildJoinBypassLevel) EnumDescriptor() ([]byte, []int) {
	return file_structs_structs_keys_proto_rawDescGZIP(), []int{3}
}

type RegistrationStatus int32

const (
	RegistrationStatus_proposed RegistrationStatus = 0
	RegistrationStatus_approved RegistrationStatus = 1
	RegistrationStatus_denied   RegistrationStatus = 2
	RegistrationStatus_revoked  RegistrationStatus = 3
)

// Enum value maps for RegistrationStatus.
var (
	RegistrationStatus_name = map[int32]string{
		0: "proposed",
		1: "approved",
		2: "denied",
		3: "revoked",
	}
	RegistrationStatus_value = map[string]int32{
		"proposed": 0,
		"approved": 1,
		"denied":   2,
		"revoked":  3,
	}
)

func (x RegistrationStatus) Enum() *RegistrationStatus {
	p := new(RegistrationStatus)
	*p = x
	return p
}

func (x RegistrationStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (RegistrationStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_structs_structs_keys_proto_enumTypes[4].Descriptor()
}

func (RegistrationStatus) Type() protoreflect.EnumType {
	return &file_structs_structs_keys_proto_enumTypes[4]
}

func (x RegistrationStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use RegistrationStatus.Descriptor instead.
func (RegistrationStatus) EnumDescriptor() ([]byte, []int) {
	return file_structs_structs_keys_proto_rawDescGZIP(), []int{4}
}

type Ambit int32

const (
	Ambit_water Ambit = 0
	Ambit_land  Ambit = 1
	Ambit_air   Ambit = 2
	Ambit_space Ambit = 3
)

// Enum value maps for Ambit.
var (
	Ambit_name = map[int32]string{
		0: "water",
		1: "land",
		2: "air",
		3: "space",
	}
	Ambit_value = map[string]int32{
		"water": 0,
		"land":  1,
		"air":   2,
		"space": 3,
	}
)

func (x Ambit) Enum() *Ambit {
	p := new(Ambit)
	*p = x
	return p
}

func (x Ambit) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Ambit) Descriptor() protoreflect.EnumDescriptor {
	return file_structs_structs_keys_proto_enumTypes[5].Descriptor()
}

func (Ambit) Type() protoreflect.EnumType {
	return &file_structs_structs_keys_proto_enumTypes[5]
}

func (x Ambit) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Ambit.Descriptor instead.
func (Ambit) EnumDescriptor() ([]byte, []int) {
	return file_structs_structs_keys_proto_rawDescGZIP(), []int{5}
}

type StructCategory int32

const (
	StructCategory_planetary StructCategory = 0
	StructCategory_fleet     StructCategory = 1
)

// Enum value maps for StructCategory.
var (
	StructCategory_name = map[int32]string{
		0: "planetary",
		1: "fleet",
	}
	StructCategory_value = map[string]int32{
		"planetary": 0,
		"fleet":     1,
	}
)

func (x StructCategory) Enum() *StructCategory {
	p := new(StructCategory)
	*p = x
	return p
}

func (x StructCategory) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (StructCategory) Descriptor() protoreflect.EnumDescriptor {
	return file_structs_structs_keys_proto_enumTypes[6].Descriptor()
}

func (StructCategory) Type() protoreflect.EnumType {
	return &file_structs_structs_keys_proto_enumTypes[6]
}

func (x StructCategory) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use StructCategory.Descriptor instead.
func (StructCategory) EnumDescriptor() ([]byte, []int) {
	return file_structs_structs_keys_proto_rawDescGZIP(), []int{6}
}

type StructStatus int32

const (
	StructStatus_building  StructStatus = 0
	StructStatus_active    StructStatus = 1
	StructStatus_inactive  StructStatus = 2
	StructStatus_destroyed StructStatus = 3
)

// Enum value maps for StructStatus.
var (
	StructStatus_name = map[int32]string{
		0: "building",
		1: "active",
		2: "inactive",
		3: "destroyed",
	}
	StructStatus_value = map[string]int32{
		"building":  0,
		"active":    1,
		"inactive":  2,
		"destroyed": 3,
	}
)

func (x StructStatus) Enum() *StructStatus {
	p := new(StructStatus)
	*p = x
	return p
}

func (x StructStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (StructStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_structs_structs_keys_proto_enumTypes[7].Descriptor()
}

func (StructStatus) Type() protoreflect.EnumType {
	return &file_structs_structs_keys_proto_enumTypes[7]
}

func (x StructStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use StructStatus.Descriptor instead.
func (StructStatus) EnumDescriptor() ([]byte, []int) {
	return file_structs_structs_keys_proto_rawDescGZIP(), []int{7}
}

type StructType int32

const (
	StructType_miningRig      StructType = 0
	StructType_refinery       StructType = 1
	StructType_smallGenerator StructType = 2
)

// Enum value maps for StructType.
var (
	StructType_name = map[int32]string{
		0: "miningRig",
		1: "refinery",
		2: "smallGenerator",
	}
	StructType_value = map[string]int32{
		"miningRig":      0,
		"refinery":       1,
		"smallGenerator": 2,
	}
)

func (x StructType) Enum() *StructType {
	p := new(StructType)
	*p = x
	return p
}

func (x StructType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (StructType) Descriptor() protoreflect.EnumDescriptor {
	return file_structs_structs_keys_proto_enumTypes[8].Descriptor()
}

func (StructType) Type() protoreflect.EnumType {
	return &file_structs_structs_keys_proto_enumTypes[8]
}

func (x StructType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use StructType.Descriptor instead.
func (StructType) EnumDescriptor() ([]byte, []int) {
	return file_structs_structs_keys_proto_rawDescGZIP(), []int{8}
}

var File_structs_structs_keys_proto protoreflect.FileDescriptor

var file_structs_structs_keys_proto_rawDesc = []byte{
	0x0a, 0x1a, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x2f, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74,
	0x73, 0x2f, 0x6b, 0x65, 0x79, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0f, 0x73, 0x74,
	0x72, 0x75, 0x63, 0x74, 0x73, 0x2e, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x1a, 0x14, 0x67,
	0x6f, 0x67, 0x6f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x67, 0x6f, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2a, 0x83, 0x01, 0x0a, 0x0a, 0x6f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x09, 0x0a, 0x05, 0x67, 0x75, 0x69, 0x6c, 0x64, 0x10, 0x00, 0x12, 0x0a, 0x0a,
	0x06, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x10, 0x01, 0x12, 0x0a, 0x0a, 0x06, 0x70, 0x6c, 0x61,
	0x6e, 0x65, 0x74, 0x10, 0x02, 0x12, 0x0b, 0x0a, 0x07, 0x72, 0x65, 0x61, 0x63, 0x74, 0x6f, 0x72,
	0x10, 0x03, 0x12, 0x0e, 0x0a, 0x0a, 0x73, 0x75, 0x62, 0x73, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x10, 0x04, 0x12, 0x0a, 0x0a, 0x06, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x10, 0x05, 0x12, 0x0e,
	0x0a, 0x0a, 0x61, 0x6c, 0x6c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x10, 0x06, 0x12, 0x0c,
	0x0a, 0x08, 0x69, 0x6e, 0x66, 0x75, 0x73, 0x69, 0x6f, 0x6e, 0x10, 0x07, 0x12, 0x0b, 0x0a, 0x07,
	0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x10, 0x08, 0x2a, 0xbd, 0x01, 0x0a, 0x11, 0x67, 0x72,
	0x69, 0x64, 0x41, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x54, 0x79, 0x70, 0x65, 0x12,
	0x07, 0x0a, 0x03, 0x6f, 0x72, 0x65, 0x10, 0x00, 0x12, 0x08, 0x0a, 0x04, 0x66, 0x75, 0x65, 0x6c,
	0x10, 0x01, 0x12, 0x0c, 0x0a, 0x08, 0x63, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79, 0x10, 0x02,
	0x12, 0x08, 0x0a, 0x04, 0x6c, 0x6f, 0x61, 0x64, 0x10, 0x03, 0x12, 0x0f, 0x0a, 0x0b, 0x73, 0x74,
	0x72, 0x75, 0x63, 0x74, 0x73, 0x4c, 0x6f, 0x61, 0x64, 0x10, 0x04, 0x12, 0x09, 0x0a, 0x05, 0x70,
	0x6f, 0x77, 0x65, 0x72, 0x10, 0x05, 0x12, 0x16, 0x0a, 0x12, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63,
	0x74, 0x69, 0x6f, 0x6e, 0x43, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79, 0x10, 0x06, 0x12, 0x13,
	0x0a, 0x0f, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x75, 0x6e,
	0x74, 0x10, 0x07, 0x12, 0x1a, 0x0a, 0x16, 0x61, 0x6c, 0x6c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x53, 0x74, 0x61, 0x72, 0x74, 0x10, 0x08, 0x12,
	0x18, 0x0a, 0x14, 0x61, 0x6c, 0x6c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x50, 0x6f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x45, 0x6e, 0x64, 0x10, 0x09, 0x2a, 0x38, 0x0a, 0x0e, 0x61, 0x6c, 0x6c,
	0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0a, 0x0a, 0x06, 0x73,
	0x74, 0x61, 0x74, 0x69, 0x63, 0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x64, 0x79, 0x6e, 0x61, 0x6d,
	0x69, 0x63, 0x10, 0x01, 0x12, 0x0d, 0x0a, 0x09, 0x61, 0x75, 0x74, 0x6f, 0x6d, 0x61, 0x74, 0x65,
	0x64, 0x10, 0x02, 0x2a, 0x40, 0x0a, 0x14, 0x67, 0x75, 0x69, 0x6c, 0x64, 0x4a, 0x6f, 0x69, 0x6e,
	0x42, 0x79, 0x70, 0x61, 0x73, 0x73, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x12, 0x0a, 0x0a, 0x06, 0x63,
	0x6c, 0x6f, 0x73, 0x65, 0x64, 0x10, 0x00, 0x12, 0x10, 0x0a, 0x0c, 0x70, 0x65, 0x72, 0x6d, 0x69,
	0x73, 0x73, 0x69, 0x6f, 0x6e, 0x65, 0x64, 0x10, 0x01, 0x12, 0x0a, 0x0a, 0x06, 0x6d, 0x65, 0x6d,
	0x62, 0x65, 0x72, 0x10, 0x02, 0x2a, 0x49, 0x0a, 0x12, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x0c, 0x0a, 0x08, 0x70,
	0x72, 0x6f, 0x70, 0x6f, 0x73, 0x65, 0x64, 0x10, 0x00, 0x12, 0x0c, 0x0a, 0x08, 0x61, 0x70, 0x70,
	0x72, 0x6f, 0x76, 0x65, 0x64, 0x10, 0x01, 0x12, 0x0a, 0x0a, 0x06, 0x64, 0x65, 0x6e, 0x69, 0x65,
	0x64, 0x10, 0x02, 0x12, 0x0b, 0x0a, 0x07, 0x72, 0x65, 0x76, 0x6f, 0x6b, 0x65, 0x64, 0x10, 0x03,
	0x2a, 0x30, 0x0a, 0x05, 0x61, 0x6d, 0x62, 0x69, 0x74, 0x12, 0x09, 0x0a, 0x05, 0x77, 0x61, 0x74,
	0x65, 0x72, 0x10, 0x00, 0x12, 0x08, 0x0a, 0x04, 0x6c, 0x61, 0x6e, 0x64, 0x10, 0x01, 0x12, 0x07,
	0x0a, 0x03, 0x61, 0x69, 0x72, 0x10, 0x02, 0x12, 0x09, 0x0a, 0x05, 0x73, 0x70, 0x61, 0x63, 0x65,
	0x10, 0x03, 0x2a, 0x2a, 0x0a, 0x0e, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x43, 0x61, 0x74, 0x65,
	0x67, 0x6f, 0x72, 0x79, 0x12, 0x0d, 0x0a, 0x09, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x74, 0x61, 0x72,
	0x79, 0x10, 0x00, 0x12, 0x09, 0x0a, 0x05, 0x66, 0x6c, 0x65, 0x65, 0x74, 0x10, 0x01, 0x2a, 0x45,
	0x0a, 0x0c, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x0c,
	0x0a, 0x08, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x69, 0x6e, 0x67, 0x10, 0x00, 0x12, 0x0a, 0x0a, 0x06,
	0x61, 0x63, 0x74, 0x69, 0x76, 0x65, 0x10, 0x01, 0x12, 0x0c, 0x0a, 0x08, 0x69, 0x6e, 0x61, 0x63,
	0x74, 0x69, 0x76, 0x65, 0x10, 0x02, 0x12, 0x0d, 0x0a, 0x09, 0x64, 0x65, 0x73, 0x74, 0x72, 0x6f,
	0x79, 0x65, 0x64, 0x10, 0x03, 0x2a, 0x3d, 0x0a, 0x0a, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x54,
	0x79, 0x70, 0x65, 0x12, 0x0d, 0x0a, 0x09, 0x6d, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x52, 0x69, 0x67,
	0x10, 0x00, 0x12, 0x0c, 0x0a, 0x08, 0x72, 0x65, 0x66, 0x69, 0x6e, 0x65, 0x72, 0x79, 0x10, 0x01,
	0x12, 0x12, 0x0a, 0x0e, 0x73, 0x6d, 0x61, 0x6c, 0x6c, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
	0x6f, 0x72, 0x10, 0x02, 0x42, 0x9f, 0x01, 0x0a, 0x13, 0x63, 0x6f, 0x6d, 0x2e, 0x73, 0x74, 0x72,
	0x75, 0x63, 0x74, 0x73, 0x2e, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x42, 0x09, 0x4b, 0x65,
	0x79, 0x73, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x20, 0x63, 0x6f, 0x73, 0x6d, 0x6f,
	0x73, 0x73, 0x64, 0x6b, 0x2e, 0x69, 0x6f, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x73, 0x74, 0x72, 0x75,
	0x63, 0x74, 0x73, 0x2f, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0xa2, 0x02, 0x03, 0x53, 0x53,
	0x58, 0xaa, 0x02, 0x0f, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x2e, 0x53, 0x74, 0x72, 0x75,
	0x63, 0x74, 0x73, 0xca, 0x02, 0x0f, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x5c, 0x53, 0x74,
	0x72, 0x75, 0x63, 0x74, 0x73, 0xe2, 0x02, 0x1b, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x5c,
	0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64,
	0x61, 0x74, 0x61, 0xea, 0x02, 0x10, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x3a, 0x3a, 0x53,
	0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_structs_structs_keys_proto_rawDescOnce sync.Once
	file_structs_structs_keys_proto_rawDescData = file_structs_structs_keys_proto_rawDesc
)

func file_structs_structs_keys_proto_rawDescGZIP() []byte {
	file_structs_structs_keys_proto_rawDescOnce.Do(func() {
		file_structs_structs_keys_proto_rawDescData = protoimpl.X.CompressGZIP(file_structs_structs_keys_proto_rawDescData)
	})
	return file_structs_structs_keys_proto_rawDescData
}

var file_structs_structs_keys_proto_enumTypes = make([]protoimpl.EnumInfo, 9)
var file_structs_structs_keys_proto_goTypes = []interface{}{
	(ObjectType)(0),           // 0: structs.structs.objectType
	(GridAttributeType)(0),    // 1: structs.structs.gridAttributeType
	(AllocationType)(0),       // 2: structs.structs.allocationType
	(GuildJoinBypassLevel)(0), // 3: structs.structs.guildJoinBypassLevel
	(RegistrationStatus)(0),   // 4: structs.structs.registrationStatus
	(Ambit)(0),                // 5: structs.structs.ambit
	(StructCategory)(0),       // 6: structs.structs.structCategory
	(StructStatus)(0),         // 7: structs.structs.structStatus
	(StructType)(0),           // 8: structs.structs.structType
}
var file_structs_structs_keys_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_structs_structs_keys_proto_init() }
func file_structs_structs_keys_proto_init() {
	if File_structs_structs_keys_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_structs_structs_keys_proto_rawDesc,
			NumEnums:      9,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_structs_structs_keys_proto_goTypes,
		DependencyIndexes: file_structs_structs_keys_proto_depIdxs,
		EnumInfos:         file_structs_structs_keys_proto_enumTypes,
	}.Build()
	File_structs_structs_keys_proto = out.File
	file_structs_structs_keys_proto_rawDesc = nil
	file_structs_structs_keys_proto_goTypes = nil
	file_structs_structs_keys_proto_depIdxs = nil
}
