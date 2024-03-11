package types

const (
	// ModuleName defines the module name
	ModuleName = "structs"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// StoreKey defines the transient module store key
	// Data stored only during block processing
	TStoreKey = "transient_structs"

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_structs"

	// Version defines the current version the IBC module supports
	Version = "structs-1"

	// PortID is the default port id that module binds to
	PortID = "structs"

	// Starting value for Keeper IDs
	KeeperStartValue = 1

	// Starting value for Reactor Owner Initialization
	InitialReactorOwnerEnergy = 100

    // Starting value for Reactor Owner Initialization
	InitialSubstationOwnerEnergy = 100


    DifficultyBuildAgeRange  = 1800  // 36000 // 1 days
    DifficultyActionAgeRange = 3600 // 252000 // 7 days

    DifficultySabotageRangeMine   = DifficultyActionAgeRange  // 36000 // 1 days
    DifficultySabotageRangeRefine = DifficultyActionAgeRange // 252000 // 7 days
    DifficultySabotageRangePower  = 252000 // 252000 // 7 days

)

var (
	// PortKey defines the key to store the port ID in store
	PortKey = KeyPrefix("structs-port-")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}


const (
    PermissionKey = "Permission/value/"
)

const (
    GridAttributeKey = "Grid/attribute/"
    GridCascadeQueue = "Grid/cascadeQueue/"
)

const (
	ReactorKey          = "Reactor/value/"
	ReactorCountKey     = "Reactor/count/"
	ReactorValidatorKey = "Reactor/validator/"


	ReactorFuelToEnergyConversion = 100
)

const (
	SubstationKey       = "Substation/value/"
	SubstationCountKey  = "Substation/count/"
	SubstationStatusKey = "Substation/status/"
)

const (
	AllocationKey      = "Allocation/value/"
	AllocationAutoResizeKey = "Allocation/autoResize/"
)

const (
	InfusionKey      = "Infusion/value/"
	InfusionCountKey = "Infusion/count/"
)

const (
	GuildKey      = "Guild/value/"
	GuildCountKey = "Guild/count/"
	GuildRegistrationKey = "Guild/registration/"
)


const (
	PlayerKey      = "Player/value/"
	PlayerCountKey = "Player/count/"

	PlayerPassiveDraw = 25
)

const (
	AddressPlayerKey = "Address/player/"
	AddressRegistrationKey = "Address/registration/"
)

const (
	PlanetKey                = "Planet/value/"
	PlanetCountKey           = "Planet/count/"
	PlanetRefinementCountKey = "Planet/refinement/"
	PlanetOreCountKey        = "Planet/ore/"

	// TODO Make these dynamic in the future
	PlanetStartingOre = 5
	PlanetStartingSlots = 4
)

const (
	StructKey      = "Struct/value/"
	StructCountKey  = "Struct/count/"

	StructFuelToEnergyConversion = 200
)



/*
 * Additional code needed for ObjectType enumeration that the proto
 * file doesn't seem to generate in keys.pb.go
 *
 * So this seems like as good a place as any for it.
 */
var ObjectType_enum = map[string]ObjectType{
	"guild":      ObjectType_guild,
	"player":     ObjectType_player,
	"planet":     ObjectType_planet,
	"reactor":    ObjectType_reactor,
	"substation": ObjectType_substation,
	"struct":     ObjectType_struct,
	"allocation": ObjectType_allocation,
	"infusion":   ObjectType_infusion,
	"address":    ObjectType_address,
}

// Doing the same for GridAttributeType
var GridAttributeType_enum = map[string]GridAttributeType{
    "ore":                      GridAttributeType_ore,
	"fuel":                     GridAttributeType_fuel,
	"capacity":                 GridAttributeType_capacity,
	"load":                     GridAttributeType_load,
	"structsLoad":              GridAttributeType_structsLoad,
	"power":                    GridAttributeType_power,
	"connectionCapacity":       GridAttributeType_connectionCapacity,
	"connectionCount":          GridAttributeType_connectionCount,
	"allocationPointerStart":   GridAttributeType_allocationPointerStart,
	"allocationPointerEnd":     GridAttributeType_allocationPointerEnd,
}


// Doing the same for AllocationType
var AllocationType_enum = map[string]AllocationType{
	"static":       AllocationType_static,
	"dynamic":      AllocationType_dynamic,
	"automated":    AllocationType_automated,

}


var GuildJoinBypassLevel_enum = map[string]GuildJoinBypassLevel {
	"closed":        GuildJoinBypassLevel_closed,
	"permissioned":  GuildJoinBypassLevel_permissioned,
	"member":        GuildJoinBypassLevel_member,
}


var RegistrationStatus_enum = map[string]RegistrationStatus {
	"proposed":  RegistrationStatus_proposed,
	"approved":  RegistrationStatus_approved,
	"denied":    RegistrationStatus_denied,
}



var Ambit_enum = map[string]Ambit {
    "water": Ambit_water,
    "land":  Ambit_land,
    "air":   Ambit_air,
    "space": Ambit_space,
}

var StructCategory_enum = map[string]StructCategory {
    "planetary":    StructCategory_planetary,
    "fleet":        StructCategory_fleet,
}


var StructStatus_enum = map[string]StructStatus {
    "building":     StructStatus_building,
    "active":       StructStatus_active,
    "inactive":     StructStatus_inactive,
    "destroyed":    StructStatus_destroyed,
}

var StructType_enum = map[string]StructType {
    "mining_rig": StructType_miningRig,
    "refinery": StructType_refinery,
    "small_generator": StructType_smallGenerator,
}

