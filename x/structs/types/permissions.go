package types

type Permission uint64

/*
	Play is....
		All Struct Actions
		All Fleet Actions
		All Planetary Actions

	Admin is Owner (this let's us have multiple owners)

*/

const (
	PermPlay    Permission = 1 << iota // 1
	PermAdmin                          // 2

	PermUpdate                         // 4
	PermDelete                         // 8

	PermTokenTransfer // 16
	PermTokenInfuse   // 32
	PermTokenMigrate  // 64
	PermTokenDefuse   // 128

	// 256 - Create, Update, Delete, Incl Provider Create
	PermSourceAllocation

	// 512 — Guild Membership (Player), Guild membership (Guild)
	PermGuildMembership

    // Substation Connection (Player, Substation)
    PermSubstationConnection

	// Substation / Allocation
	PermAllocationConnection

	// Guild — Banking
	PermGuildTokenBurn // Burn & confiscate tokens
	PermGuildTokenMint // Mint guild tokens

	// Guild — Settings
	PermGuildEndpointUpdate // Update guild endpoint
	PermGuildJoinConstraintsUpdate // Update join infusion minimum and bypass settings
	PermGuildSubstationUpdate // Update entry substation

	// Provider
	PermProviderWithdraw // Withdraw provider balance
	PermProviderOpen // Agreement restriction

	// Reactor
	PermReactorGuildCreate // Create a guild from a reactor

	// Hash (proof-of-work submitted by address)
	PermHashBuild  // Struct build completion
	PermHashMine   // Ore miner completion
	PermHashRefine // Ore refinery completion
	PermHashRaid   // Planet raid completion

	// Guild — UGC (User Generated Content)
	PermGuildUGCUpdate // Update name/pfp on guild-owned objects
)

// ── Composites ──────────────────────────────────────────
const (
	Permissionless Permission = 0

	PermAssetsAll = PermTokenTransfer | PermTokenInfuse | PermTokenMigrate | PermTokenDefuse

	PermHashAll = PermHashBuild | PermHashMine | PermHashRefine | PermHashRaid

    PermAgreementAll = PermAdmin | PermUpdate | PermDelete

    PermProviderAll = PermAdmin | PermUpdate | PermDelete |	PermProviderWithdraw |	PermProviderOpen

    PermGuildAll = PermAdmin | PermUpdate | PermDelete | PermGuildMembership |
                    PermGuildEndpointUpdate | PermGuildJoinConstraintsUpdate | PermGuildSubstationUpdate |
                    PermGuildTokenBurn | PermGuildTokenMint | PermProviderOpen | PermGuildUGCUpdate

    PermSubstationAll = PermAdmin | PermUpdate | PermDelete | PermSubstationConnection | PermSourceAllocation

    PermReactorAll = PermAdmin | PermUpdate | PermDelete | PermSourceAllocation | PermReactorGuildCreate

    PermAllocationAll = PermAdmin | PermUpdate | PermDelete | PermAllocationConnection


    // 2^25 - 1 (33,554,431)
	PermAll = PermPlay | PermAdmin | PermUpdate | PermDelete |
		PermTokenTransfer | PermTokenInfuse | PermTokenMigrate | PermTokenDefuse |
		PermSourceAllocation | PermGuildMembership | PermSubstationConnection |
		PermAllocationConnection |
		PermGuildTokenBurn | PermGuildTokenMint |
		PermGuildEndpointUpdate | PermGuildJoinConstraintsUpdate | PermGuildSubstationUpdate |
		PermProviderWithdraw | PermProviderOpen |
		PermReactorGuildCreate |
		PermHashBuild | PermHashMine | PermHashRefine | PermHashRaid |
		PermGuildUGCUpdate

    PermPlayerAll = PermAll

)

var PermissionLabel = map[Permission]string{
	Permissionless:                 "permissionless",
	PermPlay:                       "play",
	PermAdmin:                      "admin",
	PermUpdate:                     "update",
	PermDelete:                     "delete",
	PermTokenTransfer:              "token_transfer",
	PermTokenInfuse:                "token_infuse",
	PermTokenMigrate:               "token_migrate",
	PermTokenDefuse:                "token_defuse",
	PermSourceAllocation:           "source_allocation",
	PermGuildMembership:            "guild_membership",
	PermSubstationConnection:       "substation_connection",
	PermAllocationConnection:       "allocation_connection",
	PermGuildTokenBurn:             "guild_token_burn",
	PermGuildTokenMint:             "guild_token_mint",
	PermGuildEndpointUpdate:        "guild_endpoint_update",
	PermGuildJoinConstraintsUpdate: "guild_join_constraints_update",
	PermGuildSubstationUpdate:      "guild_substation_update",
	PermProviderWithdraw:           "provider_withdraw",
	PermProviderOpen:               "provider_open",
	PermReactorGuildCreate:         "reactor_guild_create",
	PermHashBuild:                  "hash_build",
	PermHashMine:                   "hash_mine",
	PermHashRefine:                 "hash_refine",
	PermHashRaid:                   "hash_raid",
	PermGuildUGCUpdate:             "guild_ugc_update",
	PermAll:                        "all",
}

func init() {
	if uint64(PermAll)>>PermissionBitCount != 0 {
		panic("PermAll exceeds PermissionBitCount -- update PermissionBitCount in keys.go")
	}
}

var Permission_enum = map[string]Permission{
	"permissionless":                 Permissionless,
	"play":                           PermPlay,
	"admin":                          PermAdmin,
	"update":                         PermUpdate,
	"delete":                         PermDelete,
	"token_transfer":                 PermTokenTransfer,
	"token_infuse":                   PermTokenInfuse,
	"token_migrate":                 PermTokenMigrate,
	"token_defuse":                  PermTokenDefuse,
	"source_allocation":             PermSourceAllocation,
	"guild_membership":              PermGuildMembership,
	"substation_connection":         PermSubstationConnection,
	"allocation_connection":         PermAllocationConnection,
	"guild_token_burn":              PermGuildTokenBurn,
	"guild_token_mint":              PermGuildTokenMint,
	"guild_endpoint_update":         PermGuildEndpointUpdate,
	"guild_join_constraints_update": PermGuildJoinConstraintsUpdate,
	"guild_substation_update":       PermGuildSubstationUpdate,
	"provider_withdraw":             PermProviderWithdraw,
	"provider_open":                 PermProviderOpen,
	"reactor_guild_create":          PermReactorGuildCreate,
	"hash_build":                    PermHashBuild,
	"hash_mine":                     PermHashMine,
	"hash_refine":                   PermHashRefine,
	"hash_raid":                     PermHashRaid,
	"guild_ugc_update":              PermGuildUGCUpdate,
	"all":                           PermAll,
}