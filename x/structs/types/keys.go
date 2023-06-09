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
)

var (
	// PortKey defines the key to store the port ID in store
	PortKey = KeyPrefix("structs-port-")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

const (
	ReactorKey              = "Reactor/value/"
	ReactorCountKey         = "Reactor/count/"
	ReactorValidatorKey     = "Reactor/validator/"
)

const (
	SubstationKey               = "Substation/value/"
	SubstationCountKey          = "Substation/count/"
	SubstationStatusKey         = "Substation/status/"

)

const (
	AllocationKey      = "Allocation/value/"
	AllocationCountKey = "Allocation/count/"
	AllocationStatusKey = "Allocation/Status/"

	AllocationStatus_Online = 1
	AllocationStatus_Offline = 0
)

const (
	AllocationProposalKey      = "AllocationProposal/value/"
	AllocationProposalCountKey = "AllocationProposal/count/"
)


/*
 * Additional code needed for ObjectType enumeration that the proto
 * file doesn't seem to generate in keys.pb.go
 *
 * So this seems like as good a place as any for it.
 */
var ObjectType_enum = map[string]ObjectType{
	"faction":    ObjectType_faction,
	"player":     ObjectType_player,
	"planet":     ObjectType_planet,
	"reactor":    ObjectType_reactor,
	"substation": ObjectType_substation,
	"struct":     ObjectType_struct,
}
