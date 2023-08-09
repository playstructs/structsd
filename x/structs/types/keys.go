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
	InitialReactorOwnerEnergy = 20

	// Starting allocation for Reactor
    InitialReactorAllocation = 100

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

	ReactorPermissionKey = "Reactor/permission/"
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
)

const (
	InfusionKey      = "Infusion/value/"
	InfusionCountKey = "Infusion/count/"
)

const (
	GuildKey      = "Guild/value/"
	GuildCountKey = "Guild/count/"
	GuildPermissionKey = "Guild/permission/"
)

const (
	PlayerKey      = "Player/value/"
	PlayerCountKey = "Player/count/"
)

const (
	AddressKey      = "Address/value/"
	AddressCountKey = "Address/count/"
	AddressPlayerKey = "Address/player/"
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

