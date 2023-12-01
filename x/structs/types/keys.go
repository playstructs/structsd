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


    // Open, Infusion, Request, Invite
	GuildJoinType_Open              = 0
	GuildJoinType_InfusionMinimum   = 1
	GuildJoinType_Request           = 2
	GuildJoinType_Invite            = 3
	GuildJoinType_Invalid           = 4
)

const (
	SquadKey      = "Squad/value/"
	SquadCountKey = "Squad/count/"
	SquadPermissionKey = "Squad/permission/"
	SquadLeaderProposalKey = "Squad/leader/"
	SquadInviteKey = "Squad/invite/"
	SquadJoinRequestKey = "Squad/request/"

    // Open, Guild Member, Request, Invite Only
	SquadJoinType_Open          = 0
	SquadJoinType_GuildMember   = 1
	SquadJoinType_Request       = 2
	SquadJoinType_Invite        = 3
	SquadJoinType_Invalid       = 4

	SquadInviteStatus_Invalid   = 0
	SquadInviteStatus_Pending   = 1
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

