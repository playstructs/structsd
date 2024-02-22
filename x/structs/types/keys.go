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
	ReactorKey          = "Reactor/value/"
	ReactorCountKey     = "Reactor/count/"
	ReactorValidatorKey = "Reactor/validator/"
	ReactorCapacityKey  = "Reactor/capacity/"
	ReactorLoadKey      = "Reactor/load/"
	ReactorEnergyKey    = "Reactor/energy/"
	ReactorFuelKey      = "Reactor/fuel/"

	ReactorPermissionKey = "Reactor/permission/"

	ReactorFuelToEnergyConversion = 100
)

const (
	SubstationKey       = "Substation/value/"
	SubstationCountKey  = "Substation/count/"
	SubstationStatusKey = "Substation/status/"

	SubstationLoadKey                = "Substation/load/"
	SubstationAllocationLoadKey      = "Substation/allocationLoad/"
	SubstationConnectedPlayerLoadKey = "Substation/connectedPlayerLoad/"
	SubstationConnectedPlayerCount   = "Substation/connectedPlayerCount/"

	SubstationEnergyKey = "Substation/energy/"

	SubstationPermissionKey = "Substation/permission/"
)

const (
	AllocationKey      = "Allocation/value/"
	AllocationCountKey = "Allocation/count/"
	AllocationAutoResizeKey = "Allocation/autoResize/"
)

const (
	InfusionKey      = "Infusion/value/"
	InfusionCountKey = "Infusion/count/"
)

const (
	GuildKey      = "Guild/value/"
	GuildCountKey = "Guild/count/"
	GuildPermissionKey = "Guild/permission/"
	GuildRegistrationKey = "Guild/registration/"


    // Guild Membership Features
	GuildJoinBypassLevel_Closed        = 0 // Feature off
	GuildJoinBypassLevel_Permissioned  = 1 // Only those with permissions can do it
	GuildJoinBypassLevel_Member        = 2 // All members of the guild can contribute
	GuildJoinBypassLevel_Invalid       = 3 // All values this and above are invalid

)


const (
	PlayerKey      = "Player/value/"
	PlayerCountKey = "Player/count/"
	PlayerPermissionKey = "Player/permission/"
	PlayerLoadKey = "Player/load/"

	PlayerPassiveDraw = 25
)

const (
	AddressPlayerKey = "Address/player/"
	AddressPermissionKey = "Address/permission/"
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
	StructLoadKey      = "Struct/load/"
    StructEnergyKey    = "Struct/energy/"
    StructFuelKey      = "Struct/fuel/"

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
}

// Doing the same for GridAttributeType
var GridAttributeType_enum = map[string]GridAttributeType{
	"fuel":                     GridAttributeType_fuel,
	"capacity":                 GridAttributeType_capacity,
	"load":                     GridAttributeType_load,
	"power":                    GridAttributeType_power,
	"connectionCapacity":       GridAttributeType_connectionCapacity,
	"connectionCount":          GridAttributeType_connectionCount,
	"allocationPointerStart":   GridAttributeType_allocationPointerStart,
	"allocationPointerEnd":     GridAttributeType_allocationPointerEnd,
}


