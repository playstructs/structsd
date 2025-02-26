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
	ObjectType_fleet      ObjectType = 9
	ObjectType_provider   ObjectType = 10
	ObjectType_agreement  ObjectType = 11
)

var ObjectType_name = map[int32]string{
	0:  "guild",
	1:  "player",
	2:  "planet",
	3:  "reactor",
	4:  "substation",
	5:  "struct",
	6:  "allocation",
	7:  "infusion",
	8:  "address",
	9:  "fleet",
	10: "provider",
	11: "agreement",
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
	"fleet":      9,
	"provider":   10,
	"agreement":  11,
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
	GridAttributeType_lastAction             GridAttributeType = 11
	GridAttributeType_nonce                  GridAttributeType = 12
	GridAttributeType_ready                  GridAttributeType = 13
	GridAttributeType_checkpointBlock        GridAttributeType = 14
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
	11: "lastAction",
	12: "nonce",
	13: "ready",
	14: "checkpointBlock",
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
	"lastAction":             11,
	"nonce":                  12,
	"ready":                  13,
	"checkpointBlock":        14,
}

func (x GridAttributeType) String() string {
	return proto.EnumName(GridAttributeType_name, int32(x))
}

func (GridAttributeType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{1}
}

type AllocationType int32

const (
	AllocationType_static            AllocationType = 0
	AllocationType_dynamic           AllocationType = 1
	AllocationType_automated         AllocationType = 2
	AllocationType_providerAgreement AllocationType = 3
)

var AllocationType_name = map[int32]string{
	0: "static",
	1: "dynamic",
	2: "automated",
	3: "providerAgreement",
}

var AllocationType_value = map[string]int32{
	"static":            0,
	"dynamic":           1,
	"automated":         2,
	"providerAgreement": 3,
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
	Ambit_none  Ambit = 0
	Ambit_water Ambit = 1
	Ambit_land  Ambit = 2
	Ambit_air   Ambit = 3
	Ambit_space Ambit = 4
	Ambit_local Ambit = 5
)

var Ambit_name = map[int32]string{
	0: "none",
	1: "water",
	2: "land",
	3: "air",
	4: "space",
	5: "local",
}

var Ambit_value = map[string]int32{
	"none":  0,
	"water": 1,
	"land":  2,
	"air":   3,
	"space": 4,
	"local": 5,
}

func (x Ambit) String() string {
	return proto.EnumName(Ambit_name, int32(x))
}

func (Ambit) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{6}
}

type RaidStatus int32

const (
	RaidStatus_initiated        RaidStatus = 0
	RaidStatus_ongoing          RaidStatus = 2
	RaidStatus_attackerDefeated RaidStatus = 1
	RaidStatus_raidSuccessful   RaidStatus = 3
	RaidStatus_demilitarized    RaidStatus = 4
)

var RaidStatus_name = map[int32]string{
	0: "initiated",
	2: "ongoing",
	1: "attackerDefeated",
	3: "raidSuccessful",
	4: "demilitarized",
}

var RaidStatus_value = map[string]int32{
	"initiated":        0,
	"ongoing":          2,
	"attackerDefeated": 1,
	"raidSuccessful":   3,
	"demilitarized":    4,
}

func (x RaidStatus) String() string {
	return proto.EnumName(RaidStatus_name, int32(x))
}

func (RaidStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{7}
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
	return fileDescriptor_d2b5c851dc116405, []int{8}
}

type FleetStatus int32

const (
	FleetStatus_onStation FleetStatus = 0
	FleetStatus_away      FleetStatus = 1
)

var FleetStatus_name = map[int32]string{
	0: "onStation",
	1: "away",
}

var FleetStatus_value = map[string]int32{
	"onStation": 0,
	"away":      1,
}

func (x FleetStatus) String() string {
	return proto.EnumName(FleetStatus_name, int32(x))
}

func (FleetStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{9}
}

type StructAttributeType int32

const (
	StructAttributeType_health               StructAttributeType = 0
	StructAttributeType_status               StructAttributeType = 1
	StructAttributeType_blockStartBuild      StructAttributeType = 2
	StructAttributeType_blockStartOreMine    StructAttributeType = 3
	StructAttributeType_blockStartOreRefine  StructAttributeType = 4
	StructAttributeType_protectedStructIndex StructAttributeType = 5
	StructAttributeType_typeCount            StructAttributeType = 6
)

var StructAttributeType_name = map[int32]string{
	0: "health",
	1: "status",
	2: "blockStartBuild",
	3: "blockStartOreMine",
	4: "blockStartOreRefine",
	5: "protectedStructIndex",
	6: "typeCount",
}

var StructAttributeType_value = map[string]int32{
	"health":               0,
	"status":               1,
	"blockStartBuild":      2,
	"blockStartOreMine":    3,
	"blockStartOreRefine":  4,
	"protectedStructIndex": 5,
	"typeCount":            6,
}

func (x StructAttributeType) String() string {
	return proto.EnumName(StructAttributeType_name, int32(x))
}

func (StructAttributeType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{10}
}

type PlanetAttributeType int32

const (
	PlanetAttributeType_planetaryShield                                            PlanetAttributeType = 0
	PlanetAttributeType_repairNetworkQuantity                                      PlanetAttributeType = 1
	PlanetAttributeType_defensiveCannonQuantity                                    PlanetAttributeType = 2
	PlanetAttributeType_coordinatedGlobalShieldNetworkQuantity                     PlanetAttributeType = 3
	PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkQuantity               PlanetAttributeType = 4
	PlanetAttributeType_advancedLowOrbitBallisticsInterceptorNetworkQuantity       PlanetAttributeType = 5
	PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkSuccessRateNumerator   PlanetAttributeType = 6
	PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkSuccessRateDenominator PlanetAttributeType = 7
	PlanetAttributeType_orbitalJammingStationQuantity                              PlanetAttributeType = 8
	PlanetAttributeType_advancedOrbitalJammingStationQuantity                      PlanetAttributeType = 9
	PlanetAttributeType_blockStartRaid                                             PlanetAttributeType = 10
)

var PlanetAttributeType_name = map[int32]string{
	0:  "planetaryShield",
	1:  "repairNetworkQuantity",
	2:  "defensiveCannonQuantity",
	3:  "coordinatedGlobalShieldNetworkQuantity",
	4:  "lowOrbitBallisticsInterceptorNetworkQuantity",
	5:  "advancedLowOrbitBallisticsInterceptorNetworkQuantity",
	6:  "lowOrbitBallisticsInterceptorNetworkSuccessRateNumerator",
	7:  "lowOrbitBallisticsInterceptorNetworkSuccessRateDenominator",
	8:  "orbitalJammingStationQuantity",
	9:  "advancedOrbitalJammingStationQuantity",
	10: "blockStartRaid",
}

var PlanetAttributeType_value = map[string]int32{
	"planetaryShield":                                            0,
	"repairNetworkQuantity":                                      1,
	"defensiveCannonQuantity":                                    2,
	"coordinatedGlobalShieldNetworkQuantity":                     3,
	"lowOrbitBallisticsInterceptorNetworkQuantity":               4,
	"advancedLowOrbitBallisticsInterceptorNetworkQuantity":       5,
	"lowOrbitBallisticsInterceptorNetworkSuccessRateNumerator":   6,
	"lowOrbitBallisticsInterceptorNetworkSuccessRateDenominator": 7,
	"orbitalJammingStationQuantity":                              8,
	"advancedOrbitalJammingStationQuantity":                      9,
	"blockStartRaid":                                             10,
}

func (x PlanetAttributeType) String() string {
	return proto.EnumName(PlanetAttributeType_name, int32(x))
}

func (PlanetAttributeType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{11}
}

type TechWeaponSystem int32

const (
	TechWeaponSystem_primaryWeapon   TechWeaponSystem = 0
	TechWeaponSystem_secondaryWeapon TechWeaponSystem = 1
)

var TechWeaponSystem_name = map[int32]string{
	0: "primaryWeapon",
	1: "secondaryWeapon",
}

var TechWeaponSystem_value = map[string]int32{
	"primaryWeapon":   0,
	"secondaryWeapon": 1,
}

func (x TechWeaponSystem) String() string {
	return proto.EnumName(TechWeaponSystem_name, int32(x))
}

func (TechWeaponSystem) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{12}
}

type TechWeaponControl int32

const (
	TechWeaponControl_noWeaponControl TechWeaponControl = 0
	TechWeaponControl_guided          TechWeaponControl = 1
	TechWeaponControl_unguided        TechWeaponControl = 2
)

var TechWeaponControl_name = map[int32]string{
	0: "noWeaponControl",
	1: "guided",
	2: "unguided",
}

var TechWeaponControl_value = map[string]int32{
	"noWeaponControl": 0,
	"guided":          1,
	"unguided":        2,
}

func (x TechWeaponControl) String() string {
	return proto.EnumName(TechWeaponControl_name, int32(x))
}

func (TechWeaponControl) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{13}
}

type TechActiveWeaponry int32

const (
	TechActiveWeaponry_noActiveWeaponry TechActiveWeaponry = 0
	TechActiveWeaponry_guidedWeaponry   TechActiveWeaponry = 1
	TechActiveWeaponry_unguidedWeaponry TechActiveWeaponry = 2
	TechActiveWeaponry_attackRun        TechActiveWeaponry = 3
	TechActiveWeaponry_selfDestruct     TechActiveWeaponry = 4
)

var TechActiveWeaponry_name = map[int32]string{
	0: "noActiveWeaponry",
	1: "guidedWeaponry",
	2: "unguidedWeaponry",
	3: "attackRun",
	4: "selfDestruct",
}

var TechActiveWeaponry_value = map[string]int32{
	"noActiveWeaponry": 0,
	"guidedWeaponry":   1,
	"unguidedWeaponry": 2,
	"attackRun":        3,
	"selfDestruct":     4,
}

func (x TechActiveWeaponry) String() string {
	return proto.EnumName(TechActiveWeaponry_name, int32(x))
}

func (TechActiveWeaponry) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{14}
}

type TechPassiveWeaponry int32

const (
	TechPassiveWeaponry_noPassiveWeaponry     TechPassiveWeaponry = 0
	TechPassiveWeaponry_counterAttack         TechPassiveWeaponry = 1
	TechPassiveWeaponry_strongCounterAttack   TechPassiveWeaponry = 2
	TechPassiveWeaponry_advancedCounterAttack TechPassiveWeaponry = 3
	TechPassiveWeaponry_lastResort            TechPassiveWeaponry = 4
)

var TechPassiveWeaponry_name = map[int32]string{
	0: "noPassiveWeaponry",
	1: "counterAttack",
	2: "strongCounterAttack",
	3: "advancedCounterAttack",
	4: "lastResort",
}

var TechPassiveWeaponry_value = map[string]int32{
	"noPassiveWeaponry":     0,
	"counterAttack":         1,
	"strongCounterAttack":   2,
	"advancedCounterAttack": 3,
	"lastResort":            4,
}

func (x TechPassiveWeaponry) String() string {
	return proto.EnumName(TechPassiveWeaponry_name, int32(x))
}

func (TechPassiveWeaponry) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{15}
}

type TechUnitDefenses int32

const (
	TechUnitDefenses_noUnitDefenses       TechUnitDefenses = 0
	TechUnitDefenses_defensiveManeuver    TechUnitDefenses = 1
	TechUnitDefenses_signalJamming        TechUnitDefenses = 2
	TechUnitDefenses_armour               TechUnitDefenses = 3
	TechUnitDefenses_indirectCombatModule TechUnitDefenses = 4
	TechUnitDefenses_stealthMode          TechUnitDefenses = 5
	TechUnitDefenses_perimeterFencing     TechUnitDefenses = 6
	TechUnitDefenses_reinforcedWalls      TechUnitDefenses = 7
)

var TechUnitDefenses_name = map[int32]string{
	0: "noUnitDefenses",
	1: "defensiveManeuver",
	2: "signalJamming",
	3: "armour",
	4: "indirectCombatModule",
	5: "stealthMode",
	6: "perimeterFencing",
	7: "reinforcedWalls",
}

var TechUnitDefenses_value = map[string]int32{
	"noUnitDefenses":       0,
	"defensiveManeuver":    1,
	"signalJamming":        2,
	"armour":               3,
	"indirectCombatModule": 4,
	"stealthMode":          5,
	"perimeterFencing":     6,
	"reinforcedWalls":      7,
}

func (x TechUnitDefenses) String() string {
	return proto.EnumName(TechUnitDefenses_name, int32(x))
}

func (TechUnitDefenses) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{16}
}

type TechOreReserveDefenses int32

const (
	TechOreReserveDefenses_noOreReserveDefenses              TechOreReserveDefenses = 0
	TechOreReserveDefenses_coordinatedReserveResponseTracker TechOreReserveDefenses = 1
	TechOreReserveDefenses_rapidResponsePackage              TechOreReserveDefenses = 2
	TechOreReserveDefenses_activeScanning                    TechOreReserveDefenses = 3
	TechOreReserveDefenses_monitoringStation                 TechOreReserveDefenses = 4
	TechOreReserveDefenses_oreBunker                         TechOreReserveDefenses = 5
)

var TechOreReserveDefenses_name = map[int32]string{
	0: "noOreReserveDefenses",
	1: "coordinatedReserveResponseTracker",
	2: "rapidResponsePackage",
	3: "activeScanning",
	4: "monitoringStation",
	5: "oreBunker",
}

var TechOreReserveDefenses_value = map[string]int32{
	"noOreReserveDefenses":              0,
	"coordinatedReserveResponseTracker": 1,
	"rapidResponsePackage":              2,
	"activeScanning":                    3,
	"monitoringStation":                 4,
	"oreBunker":                         5,
}

func (x TechOreReserveDefenses) String() string {
	return proto.EnumName(TechOreReserveDefenses_name, int32(x))
}

func (TechOreReserveDefenses) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{17}
}

type TechPlanetaryDefenses int32

const (
	TechPlanetaryDefenses_noPlanetaryDefense                  TechPlanetaryDefenses = 0
	TechPlanetaryDefenses_defensiveCannon                     TechPlanetaryDefenses = 1
	TechPlanetaryDefenses_lowOrbitBallisticInterceptorNetwork TechPlanetaryDefenses = 2
)

var TechPlanetaryDefenses_name = map[int32]string{
	0: "noPlanetaryDefense",
	1: "defensiveCannon",
	2: "lowOrbitBallisticInterceptorNetwork",
}

var TechPlanetaryDefenses_value = map[string]int32{
	"noPlanetaryDefense":                  0,
	"defensiveCannon":                     1,
	"lowOrbitBallisticInterceptorNetwork": 2,
}

func (x TechPlanetaryDefenses) String() string {
	return proto.EnumName(TechPlanetaryDefenses_name, int32(x))
}

func (TechPlanetaryDefenses) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{18}
}

type TechStorageFacilities int32

const (
	TechStorageFacilities_noStorageFacilities TechStorageFacilities = 0
	TechStorageFacilities_dock                TechStorageFacilities = 1
	TechStorageFacilities_hanger              TechStorageFacilities = 2
	TechStorageFacilities_fleetBase           TechStorageFacilities = 3
)

var TechStorageFacilities_name = map[int32]string{
	0: "noStorageFacilities",
	1: "dock",
	2: "hanger",
	3: "fleetBase",
}

var TechStorageFacilities_value = map[string]int32{
	"noStorageFacilities": 0,
	"dock":                1,
	"hanger":              2,
	"fleetBase":           3,
}

func (x TechStorageFacilities) String() string {
	return proto.EnumName(TechStorageFacilities_name, int32(x))
}

func (TechStorageFacilities) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{19}
}

type TechPlanetaryMining int32

const (
	TechPlanetaryMining_noPlanetaryMining TechPlanetaryMining = 0
	TechPlanetaryMining_oreMiningRig      TechPlanetaryMining = 1
)

var TechPlanetaryMining_name = map[int32]string{
	0: "noPlanetaryMining",
	1: "oreMiningRig",
}

var TechPlanetaryMining_value = map[string]int32{
	"noPlanetaryMining": 0,
	"oreMiningRig":      1,
}

func (x TechPlanetaryMining) String() string {
	return proto.EnumName(TechPlanetaryMining_name, int32(x))
}

func (TechPlanetaryMining) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{20}
}

type TechPlanetaryRefineries int32

const (
	TechPlanetaryRefineries_noPlanetaryRefinery TechPlanetaryRefineries = 0
	TechPlanetaryRefineries_oreRefinery         TechPlanetaryRefineries = 1
)

var TechPlanetaryRefineries_name = map[int32]string{
	0: "noPlanetaryRefinery",
	1: "oreRefinery",
}

var TechPlanetaryRefineries_value = map[string]int32{
	"noPlanetaryRefinery": 0,
	"oreRefinery":         1,
}

func (x TechPlanetaryRefineries) String() string {
	return proto.EnumName(TechPlanetaryRefineries_name, int32(x))
}

func (TechPlanetaryRefineries) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{21}
}

type TechPowerGeneration int32

const (
	TechPowerGeneration_noPowerGeneration TechPowerGeneration = 0
	TechPowerGeneration_smallGenerator    TechPowerGeneration = 1
	TechPowerGeneration_mediumGenerator   TechPowerGeneration = 2
	TechPowerGeneration_largeGenerator    TechPowerGeneration = 3
)

var TechPowerGeneration_name = map[int32]string{
	0: "noPowerGeneration",
	1: "smallGenerator",
	2: "mediumGenerator",
	3: "largeGenerator",
}

var TechPowerGeneration_value = map[string]int32{
	"noPowerGeneration": 0,
	"smallGenerator":    1,
	"mediumGenerator":   2,
	"largeGenerator":    3,
}

func (x TechPowerGeneration) String() string {
	return proto.EnumName(TechPowerGeneration_name, int32(x))
}

func (TechPowerGeneration) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{22}
}

type ProviderAccessPolicy int32

const (
	ProviderAccessPolicy_openMarket   ProviderAccessPolicy = 0
	ProviderAccessPolicy_guildMarket  ProviderAccessPolicy = 1
	ProviderAccessPolicy_closedMarket ProviderAccessPolicy = 2
)

var ProviderAccessPolicy_name = map[int32]string{
	0: "openMarket",
	1: "guildMarket",
	2: "closedMarket",
}

var ProviderAccessPolicy_value = map[string]int32{
	"openMarket":   0,
	"guildMarket":  1,
	"closedMarket": 2,
}

func (x ProviderAccessPolicy) String() string {
	return proto.EnumName(ProviderAccessPolicy_name, int32(x))
}

func (ProviderAccessPolicy) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d2b5c851dc116405, []int{23}
}

func init() {
	proto.RegisterEnum("structs.structs.ObjectType", ObjectType_name, ObjectType_value)
	proto.RegisterEnum("structs.structs.GridAttributeType", GridAttributeType_name, GridAttributeType_value)
	proto.RegisterEnum("structs.structs.AllocationType", AllocationType_name, AllocationType_value)
	proto.RegisterEnum("structs.structs.GuildJoinBypassLevel", GuildJoinBypassLevel_name, GuildJoinBypassLevel_value)
	proto.RegisterEnum("structs.structs.GuildJoinType", GuildJoinType_name, GuildJoinType_value)
	proto.RegisterEnum("structs.structs.RegistrationStatus", RegistrationStatus_name, RegistrationStatus_value)
	proto.RegisterEnum("structs.structs.Ambit", Ambit_name, Ambit_value)
	proto.RegisterEnum("structs.structs.RaidStatus", RaidStatus_name, RaidStatus_value)
	proto.RegisterEnum("structs.structs.PlanetStatus", PlanetStatus_name, PlanetStatus_value)
	proto.RegisterEnum("structs.structs.FleetStatus", FleetStatus_name, FleetStatus_value)
	proto.RegisterEnum("structs.structs.StructAttributeType", StructAttributeType_name, StructAttributeType_value)
	proto.RegisterEnum("structs.structs.PlanetAttributeType", PlanetAttributeType_name, PlanetAttributeType_value)
	proto.RegisterEnum("structs.structs.TechWeaponSystem", TechWeaponSystem_name, TechWeaponSystem_value)
	proto.RegisterEnum("structs.structs.TechWeaponControl", TechWeaponControl_name, TechWeaponControl_value)
	proto.RegisterEnum("structs.structs.TechActiveWeaponry", TechActiveWeaponry_name, TechActiveWeaponry_value)
	proto.RegisterEnum("structs.structs.TechPassiveWeaponry", TechPassiveWeaponry_name, TechPassiveWeaponry_value)
	proto.RegisterEnum("structs.structs.TechUnitDefenses", TechUnitDefenses_name, TechUnitDefenses_value)
	proto.RegisterEnum("structs.structs.TechOreReserveDefenses", TechOreReserveDefenses_name, TechOreReserveDefenses_value)
	proto.RegisterEnum("structs.structs.TechPlanetaryDefenses", TechPlanetaryDefenses_name, TechPlanetaryDefenses_value)
	proto.RegisterEnum("structs.structs.TechStorageFacilities", TechStorageFacilities_name, TechStorageFacilities_value)
	proto.RegisterEnum("structs.structs.TechPlanetaryMining", TechPlanetaryMining_name, TechPlanetaryMining_value)
	proto.RegisterEnum("structs.structs.TechPlanetaryRefineries", TechPlanetaryRefineries_name, TechPlanetaryRefineries_value)
	proto.RegisterEnum("structs.structs.TechPowerGeneration", TechPowerGeneration_name, TechPowerGeneration_value)
	proto.RegisterEnum("structs.structs.ProviderAccessPolicy", ProviderAccessPolicy_name, ProviderAccessPolicy_value)
}

func init() { proto.RegisterFile("structs/structs/keys.proto", fileDescriptor_d2b5c851dc116405) }

var fileDescriptor_d2b5c851dc116405 = []byte{
	// 1519 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x56, 0x4f, 0x6f, 0x64, 0x47,
	0x11, 0x9f, 0x37, 0xe3, 0xb1, 0x3d, 0x65, 0x7b, 0xdd, 0x6e, 0x7b, 0xff, 0xb0, 0x08, 0x4b, 0x11,
	0x4a, 0x80, 0x27, 0x94, 0x80, 0xe0, 0x10, 0x45, 0x28, 0xe0, 0xf1, 0x26, 0xd1, 0x46, 0xeb, 0xdd,
	0xc5, 0x4e, 0x14, 0x89, 0x5b, 0x4f, 0xbf, 0x9a, 0xe7, 0x66, 0xde, 0xeb, 0x7a, 0x74, 0xf7, 0x1b,
	0xef, 0x70, 0xe5, 0xc2, 0x91, 0x3b, 0x1f, 0x01, 0x71, 0xe1, 0x53, 0x70, 0xcc, 0x11, 0x71, 0x42,
	0xbb, 0x5f, 0x04, 0x55, 0xf7, 0x9b, 0x19, 0x8f, 0x2d, 0x21, 0x72, 0x9a, 0x79, 0xbf, 0xea, 0xae,
	0xfa, 0x75, 0xf5, 0xaf, 0xaa, 0x1a, 0x9e, 0xfa, 0xe0, 0x5a, 0x1d, 0xfc, 0x47, 0xcb, 0xdf, 0x19,
	0x2e, 0xfc, 0x87, 0x8d, 0xa3, 0x40, 0xf2, 0xb0, 0xc3, 0x3e, 0xec, 0x7e, 0x9f, 0x9e, 0x94, 0x54,
	0x52, 0xb4, 0x7d, 0xc4, 0xff, 0xd2, 0xb2, 0xfc, 0x6f, 0x19, 0x00, 0x4d, 0x7e, 0x8f, 0x3a, 0x7c,
	0xb5, 0x68, 0x50, 0x8e, 0x60, 0x58, 0xb6, 0xa6, 0x2a, 0x44, 0x4f, 0x02, 0x6c, 0x37, 0x95, 0x5a,
	0xa0, 0x13, 0x59, 0xf7, 0xdf, 0x62, 0x10, 0x7d, 0xb9, 0x07, 0x3b, 0x0e, 0x95, 0x0e, 0xe4, 0xc4,
	0x40, 0x3e, 0x00, 0xf0, 0xed, 0xc4, 0x07, 0x15, 0x0c, 0x59, 0xb1, 0xc5, 0x0b, 0x53, 0x3c, 0x31,
	0x64, 0x9b, 0xaa, 0x2a, 0xd2, 0xc9, 0xb6, 0x2d, 0xf7, 0x61, 0xd7, 0xd8, 0x69, 0xeb, 0xf9, 0x6b,
	0x87, 0xdd, 0xa8, 0xa2, 0x70, 0xe8, 0xbd, 0xd8, 0xe5, 0xb0, 0xd3, 0x0a, 0x31, 0x88, 0x11, 0xaf,
	0x6a, 0x1c, 0xcd, 0x4d, 0x81, 0x4e, 0x80, 0x3c, 0x80, 0x91, 0x2a, 0x1d, 0x62, 0x8d, 0x36, 0x88,
	0xbd, 0xfc, 0xcf, 0x7d, 0x38, 0x2a, 0x9d, 0x29, 0xce, 0x42, 0x70, 0x66, 0xd2, 0x06, 0x8c, 0xa4,
	0x77, 0x60, 0x40, 0x0e, 0x45, 0x4f, 0xee, 0xc2, 0xd6, 0xb4, 0xc5, 0x4a, 0x64, 0xec, 0x45, 0xab,
	0x46, 0x69, 0x13, 0x16, 0xa2, 0xcf, 0x78, 0x45, 0xaa, 0x10, 0x03, 0x79, 0x08, 0x7b, 0x5d, 0x3e,
	0x5e, 0x30, 0xb0, 0xc5, 0x91, 0x1b, 0xba, 0x41, 0x27, 0x86, 0xf2, 0x11, 0x48, 0x4d, 0xd6, 0xa2,
	0x66, 0xbe, 0xe7, 0xcb, 0xdd, 0xdb, 0xf2, 0x18, 0x0e, 0x6f, 0xe1, 0xd4, 0xda, 0x20, 0x76, 0xe4,
	0x53, 0x78, 0xb4, 0x3e, 0xdc, 0x6b, 0x32, 0x36, 0xa0, 0xbb, 0x0a, 0xca, 0x05, 0xb1, 0x2b, 0x9f,
	0xc0, 0xc9, 0x3d, 0xdb, 0x67, 0xb6, 0x10, 0x23, 0x4e, 0x49, 0xe3, 0xe8, 0xcd, 0xe2, 0x25, 0x59,
	0x8d, 0x02, 0xf8, 0xbb, 0x52, 0x3e, 0x9c, 0x45, 0xd7, 0x62, 0x8f, 0xd9, 0xd8, 0x68, 0xda, 0xe7,
	0xbf, 0x0e, 0x55, 0xb1, 0x10, 0x07, 0x91, 0xc0, 0x35, 0xea, 0x59, 0xc3, 0xae, 0xc6, 0x15, 0xe9,
	0x99, 0x78, 0x90, 0xbf, 0x82, 0x07, 0xeb, 0x20, 0x31, 0x0d, 0x31, 0xf7, 0x2a, 0x18, 0x2d, 0x7a,
	0x9c, 0xdd, 0x62, 0x61, 0x55, 0x6d, 0xb4, 0xc8, 0x62, 0x12, 0xdb, 0x40, 0xb5, 0x0a, 0x58, 0x88,
	0xbe, 0x7c, 0x08, 0x47, 0xcb, 0x0c, 0x9f, 0xad, 0x72, 0x3b, 0xc8, 0x7f, 0x03, 0x27, 0xf1, 0xea,
	0xbf, 0x24, 0x63, 0xc7, 0x8b, 0x46, 0x79, 0xff, 0x02, 0xe7, 0x58, 0xb1, 0x5b, 0x5d, 0x91, 0x47,
	0xd6, 0x84, 0x80, 0xfd, 0x06, 0x5d, 0x6d, 0x3c, 0x5f, 0x22, 0x16, 0x49, 0x19, 0x35, 0xd6, 0x13,
	0x74, 0xa2, 0x9f, 0xff, 0x1a, 0x0e, 0x56, 0x1e, 0x96, 0x8c, 0x8c, 0x9d, 0x9b, 0x80, 0x89, 0x91,
	0xc3, 0x3f, 0xb4, 0xe8, 0x43, 0xda, 0x55, 0x18, 0x87, 0x9a, 0xf5, 0xc4, 0x37, 0xc0, 0x39, 0x11,
	0x83, 0xfc, 0x39, 0x48, 0x87, 0xa5, 0xf1, 0xc1, 0xc5, 0x53, 0x5d, 0x05, 0x15, 0x5a, 0xdf, 0x29,
	0xa2, 0xe9, 0x28, 0xec, 0xc3, 0xae, 0x6a, 0x98, 0xff, 0x32, 0x7c, 0x81, 0xd6, 0xc4, 0x73, 0xc5,
	0x08, 0x73, 0x9a, 0x61, 0x21, 0x06, 0xf9, 0x67, 0x30, 0x54, 0xf5, 0xc4, 0x04, 0xbe, 0x7b, 0x4b,
	0x96, 0x19, 0x8c, 0x60, 0x78, 0xa3, 0x42, 0xd4, 0x33, 0x0b, 0x42, 0x59, 0xde, 0xb4, 0x03, 0x03,
	0x65, 0x58, 0xc9, 0x23, 0x18, 0xfa, 0x46, 0x69, 0x4c, 0x9a, 0xe0, 0xc4, 0x56, 0x62, 0x98, 0x23,
	0x80, 0x53, 0xa6, 0xe8, 0x98, 0x1c, 0xc0, 0xc8, 0x58, 0x13, 0x4c, 0x4c, 0x64, 0x3c, 0x12, 0xd9,
	0x92, 0x8c, 0x2d, 0x45, 0x5f, 0x9e, 0x80, 0x50, 0x21, 0x28, 0x3d, 0x43, 0xf7, 0x0c, 0xa7, 0x18,
	0x97, 0x64, 0x52, 0xc2, 0x83, 0xb8, 0xbf, 0xd5, 0x1a, 0xbd, 0x9f, 0xb6, 0x95, 0x18, 0xc8, 0x23,
	0x38, 0x28, 0xb0, 0x36, 0x95, 0x09, 0xca, 0x99, 0x3f, 0x62, 0x21, 0xb6, 0xf2, 0x1f, 0xc3, 0x7e,
	0xaa, 0xaf, 0x2e, 0x10, 0xc0, 0xb6, 0xd2, 0xc1, 0xcc, 0x31, 0x1d, 0x58, 0x53, 0xdd, 0x54, 0x18,
	0x50, 0x64, 0xf9, 0x07, 0xb0, 0x17, 0x2b, 0x65, 0xcd, 0x28, 0xe5, 0x89, 0xf5, 0x13, 0x0b, 0x40,
	0xdd, 0xa8, 0x85, 0xc8, 0xf2, 0xbf, 0x66, 0x70, 0x9c, 0x94, 0xbe, 0x59, 0x2b, 0x00, 0xdb, 0xd7,
	0xa8, 0xaa, 0x70, 0x9d, 0x2a, 0xdc, 0x47, 0x37, 0x22, 0x63, 0x8d, 0x4d, 0x58, 0x59, 0x51, 0xc3,
	0xe3, 0xd8, 0x02, 0xa2, 0x52, 0xd6, 0xe0, 0x2b, 0x87, 0x17, 0xc6, 0xa2, 0x18, 0xc8, 0xc7, 0x70,
	0xbc, 0x01, 0x5f, 0xe2, 0x94, 0x0d, 0x5b, 0x2c, 0x7c, 0xee, 0x2a, 0xa8, 0x03, 0x16, 0x57, 0x31,
	0xf8, 0x73, 0x5b, 0xe0, 0x1b, 0x31, 0x64, 0x9e, 0x61, 0xd1, 0x60, 0xaa, 0x9e, 0xed, 0xfc, 0xdf,
	0x03, 0x38, 0x4e, 0x07, 0xde, 0x64, 0x77, 0x0c, 0x87, 0x09, 0x56, 0x6e, 0x71, 0x75, 0x6d, 0x30,
	0x36, 0xa2, 0xef, 0xc1, 0x43, 0x87, 0x8d, 0x32, 0xee, 0x25, 0x86, 0x1b, 0x72, 0xb3, 0xdf, 0xb6,
	0xca, 0x06, 0x2e, 0xcd, 0x4c, 0x7e, 0x1f, 0x1e, 0x17, 0x38, 0x45, 0xeb, 0xcd, 0x1c, 0xcf, 0x95,
	0xb5, 0x64, 0x57, 0xc6, 0xbe, 0xcc, 0xe1, 0x03, 0x4d, 0xe4, 0x0a, 0x63, 0xf9, 0x32, 0xbe, 0xa8,
	0x68, 0xa2, 0xaa, 0xe4, 0xf4, 0xae, 0xa3, 0x81, 0xfc, 0x19, 0xfc, 0xb4, 0xa2, 0x9b, 0x57, 0x6e,
	0x62, 0xc2, 0x58, 0x55, 0x95, 0xf1, 0xc1, 0x68, 0xff, 0x9c, 0x0b, 0x57, 0x63, 0x13, 0xe8, 0x5e,
	0xe8, 0x2d, 0xf9, 0x31, 0xfc, 0x52, 0x15, 0x73, 0x65, 0x35, 0x16, 0x2f, 0xbe, 0xcb, 0xce, 0xa1,
	0xfc, 0x15, 0x7c, 0xfc, 0xff, 0xc4, 0xea, 0x34, 0x73, 0xa9, 0x02, 0xbe, 0x6c, 0x6b, 0x74, 0x8a,
	0x3b, 0xee, 0xb6, 0xfc, 0x14, 0x3e, 0xf9, 0x8e, 0xbb, 0x9f, 0xa1, 0xa5, 0x9a, 0x93, 0x40, 0x4e,
	0xec, 0xc8, 0xf7, 0xe0, 0x07, 0xc4, 0x9b, 0x55, 0xf5, 0xa5, 0xaa, 0x6b, 0x63, 0xcb, 0x4e, 0x3d,
	0x2b, 0x82, 0xbb, 0xf2, 0x27, 0xf0, 0xfe, 0xf2, 0x68, 0xaf, 0xfe, 0xe7, 0xd2, 0x11, 0xeb, 0x7b,
	0x2d, 0x85, 0x4b, 0x65, 0x0a, 0x01, 0xf9, 0x27, 0x20, 0x02, 0xea, 0xeb, 0x6f, 0x50, 0x35, 0x64,
	0xaf, 0x16, 0x3e, 0x60, 0xcd, 0x9a, 0x6f, 0x9c, 0xa9, 0x95, 0x5b, 0x24, 0x58, 0xf4, 0xf8, 0xae,
	0x3d, 0x6a, 0xb2, 0xc5, 0x1a, 0xcc, 0xf2, 0x31, 0x1c, 0xad, 0xf7, 0x9e, 0x93, 0x0d, 0x8e, 0x2a,
	0x5e, 0x69, 0x69, 0x03, 0x4a, 0xe2, 0x2d, 0x5b, 0x53, 0xc4, 0x2a, 0xdb, 0x87, 0xdd, 0xd6, 0x76,
	0x5f, 0xfd, 0xbc, 0x05, 0xc9, 0x3e, 0xce, 0x62, 0x01, 0xa5, 0x6d, 0x6e, 0xc1, 0xf5, 0x69, 0x69,
	0x13, 0x13, 0x3d, 0xe6, 0x9f, 0xf6, 0xad, 0xb0, 0x8c, 0x57, 0x2e, 0xbd, 0xad, 0xd0, 0x7e, 0x6c,
	0xa2, 0xb1, 0xbe, 0x2f, 0x5b, 0x2b, 0x06, 0xdc, 0x09, 0x3d, 0x56, 0xd3, 0x67, 0xd8, 0x8d, 0xbb,
	0xad, 0xfc, 0x4f, 0x19, 0x1c, 0x73, 0xdc, 0xd7, 0xca, 0xfb, 0xdb, 0x81, 0x1f, 0xc2, 0x91, 0xa5,
	0x3b, 0xa0, 0xe8, 0x71, 0x46, 0x34, 0x57, 0x03, 0xba, 0xb3, 0xe8, 0x56, 0x64, 0x5c, 0x57, 0x3e,
	0x38, 0xb2, 0xe5, 0xf9, 0x86, 0xa1, 0xcf, 0x15, 0xb0, 0xbc, 0x90, 0x4d, 0xd3, 0x60, 0x39, 0x41,
	0x2e, 0xd1, 0x93, 0x63, 0x16, 0xff, 0xc8, 0x52, 0xf6, 0xbf, 0xb6, 0x26, 0x3c, 0x8b, 0xa5, 0x81,
	0x9e, 0x4f, 0x69, 0xe9, 0x36, 0x22, 0x7a, 0x4c, 0x6b, 0x55, 0x3a, 0x17, 0xca, 0x62, 0x3b, 0x8f,
	0x9d, 0xf1, 0x08, 0x0e, 0xbc, 0x29, 0xed, 0xea, 0xca, 0x45, 0x3f, 0x36, 0x23, 0x57, 0x53, 0xcb,
	0x5d, 0xf2, 0x09, 0x9c, 0x18, 0x9b, 0x5a, 0xf7, 0x39, 0xd5, 0x13, 0x15, 0x2e, 0xa8, 0x68, 0x2b,
	0xae, 0xfd, 0x38, 0x59, 0x63, 0x67, 0xb9, 0xa0, 0x02, 0xc5, 0x90, 0xd3, 0xd8, 0xa0, 0x33, 0x35,
	0x06, 0x74, 0x9f, 0xa3, 0xd5, 0xec, 0x2c, 0x0e, 0x53, 0x87, 0xc6, 0x4e, 0xc9, 0x69, 0x2c, 0xbe,
	0x51, 0x55, 0xe5, 0xc5, 0x4e, 0xfe, 0xf7, 0x0c, 0x1e, 0x31, 0xe9, 0xd8, 0x4b, 0x3c, 0xba, 0x39,
	0xae, 0xa8, 0x3f, 0x81, 0x13, 0x4b, 0xf7, 0x71, 0xd1, 0x93, 0xef, 0xc3, 0x7b, 0xb7, 0xca, 0xbb,
	0xb3, 0x5f, 0xa2, 0x6f, 0xc8, 0x7a, 0xfc, 0xca, 0xc5, 0x4e, 0x2c, 0x32, 0x76, 0xe0, 0x54, 0x63,
	0x8a, 0xa5, 0xe5, 0xb5, 0xd2, 0x33, 0x55, 0xa2, 0xe8, 0x73, 0x56, 0x52, 0x93, 0xbd, 0xd2, 0xca,
	0x5a, 0xa6, 0x37, 0xe0, 0xac, 0xd4, 0x64, 0x4d, 0x20, 0xb7, 0x96, 0xbb, 0xd8, 0x8a, 0x6d, 0xd6,
	0xe1, 0xb8, 0xb5, 0xec, 0x73, 0x98, 0xd7, 0xf0, 0x30, 0xde, 0xf4, 0xb2, 0x55, 0xad, 0xd8, 0x3e,
	0x02, 0x69, 0xe9, 0x2e, 0x9c, 0xb4, 0x7e, 0xa7, 0x4f, 0x89, 0x4c, 0xfe, 0x08, 0x7e, 0x78, 0xaf,
	0x92, 0xef, 0x17, 0xb2, 0xe8, 0xe7, 0x5f, 0xa7, 0x70, 0x57, 0x81, 0x9c, 0x2a, 0xf1, 0x73, 0xa5,
	0x79, 0x76, 0x18, 0xf4, 0x2c, 0x18, 0x4b, 0xf7, 0xe0, 0x34, 0x07, 0x0a, 0x8a, 0x9a, 0xe2, 0x7e,
	0xaf, 0x6c, 0xc9, 0xf3, 0x99, 0x4f, 0x11, 0x67, 0xc7, 0x58, 0x79, 0x14, 0x83, 0xfc, 0xd3, 0x4e,
	0xaf, 0x4b, 0xba, 0x17, 0x86, 0x93, 0xd0, 0xe9, 0x75, 0x13, 0x4c, 0xa3, 0x9f, 0xe2, 0x04, 0x30,
	0xb6, 0xbc, 0x34, 0xa5, 0xc8, 0xf2, 0x73, 0x78, 0xbc, 0xb1, 0x3f, 0x8d, 0x01, 0xb7, 0x22, 0x76,
	0xd7, 0xc0, 0xaa, 0x3f, 0x84, 0x3d, 0x5a, 0x0e, 0x0c, 0x2e, 0xb6, 0xdc, 0x74, 0x24, 0xf8, 0x0d,
	0xf6, 0x05, 0x5a, 0x4c, 0x93, 0xbf, 0x23, 0xb1, 0x09, 0xa6, 0x72, 0xf5, 0xb5, 0xaa, 0xaa, 0x0e,
	0x24, 0x97, 0x26, 0x57, 0x8d, 0x85, 0x69, 0xeb, 0x35, 0x18, 0xef, 0xb6, 0x52, 0xae, 0xc4, 0x35,
	0xc6, 0xaf, 0x8b, 0x93, 0xd5, 0xbb, 0x27, 0x36, 0xc7, 0xd7, 0x54, 0x19, 0xbd, 0xe0, 0x12, 0xa2,
	0x06, 0xed, 0x85, 0x72, 0x33, 0x0c, 0x89, 0x63, 0x7c, 0xc6, 0x74, 0x40, 0xc6, 0x47, 0x4f, 0x2f,
	0xa0, 0x0e, 0xe9, 0x8f, 0x7f, 0xfe, 0xcf, 0xb7, 0xa7, 0xd9, 0xb7, 0x6f, 0x4f, 0xb3, 0xff, 0xbc,
	0x3d, 0xcd, 0xfe, 0xf2, 0xee, 0xb4, 0xf7, 0xed, 0xbb, 0xd3, 0xde, 0xbf, 0xde, 0x9d, 0xf6, 0x7e,
	0xf7, 0x78, 0xf9, 0x14, 0x7f, 0xb3, 0x7a, 0x94, 0xf3, 0xe0, 0xf3, 0x93, 0xed, 0xf8, 0xde, 0xfe,
	0xc5, 0x7f, 0x03, 0x00, 0x00, 0xff, 0xff, 0x51, 0x1e, 0x7e, 0xdf, 0xb4, 0x0b, 0x00, 0x00,
}
