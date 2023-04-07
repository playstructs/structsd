package types

const (
	// ModuleName defines the module name
	ModuleName = "structs"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

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
	ReactorKey          = "Reactor/value/"
	ReactorCountKey     = "Reactor/count/"
	ReactorValidatorKey = "Reactor/validator/"
)

const (
	SubstationKey      = "Substation/value/"
	SubstationCountKey = "Substation/count/"
)

const (
	AllocationKey      = "Allocation/value/"
	AllocationCountKey = "Allocation/count/"
)

const (
	AllocationProposalKey      = "AllocationProposal/value/"
	AllocationProposalCountKey = "AllocationProposal/count/"
)
