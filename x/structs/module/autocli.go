package structs

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "structs/api/structs/structs"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
                    RpcMethod:      "GetBlockHeight",
                    Use:            "block-height",
                    Short:          "Get the current Block Height",
                },
				{
                    RpcMethod:      "Address",
                    Use:            "address [address]",
                    Short:          "Show the details of a specific Address",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
                },
                {
                    RpcMethod:      "AddressAll",
                    Use:            "address-all",
                    Short:          "Returns all Addresses",
                },
				{
                    RpcMethod:      "AddressAllByPlayer",
                    Use:            "address-all-by-player [player id]",
                    Short:          "Returns all Addresses for a specific Player",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "playerId"}},
                },
                {
                    RpcMethod:      "Allocation",
                    Use:            "allocation [allocation id]",
                    Short:          "Show the details of a specific Allocation",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
                },
                {
                    RpcMethod:      "AllocationAll",
                    Use:            "allocation-all",
                    Short:          "Returns all Allocations",
                },
				{
                    RpcMethod:      "Fleet",
                    Use:            "fleet [fleet id]",
                    Short:          "Show the details of a specific Fleet",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
                },
				{
                    RpcMethod:      "FleetByIndex",
                    Use:            "fleet-by-index [index]",
                    Short:          "Show the details of a specific Fleet, as looked up by the index",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "index"}},
                },
                {
                    RpcMethod:      "FleetAll",
                    Use:            "fleet-all",
                    Short:          "Returns all Fleets",
                },
				{
                    RpcMethod:      "Grid",
                    Use:            "grid [grid attribute id]",
                    Short:          "Show the details of a specific Grid Attribute",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "attributeId"}},
                },
                {
                    RpcMethod:      "GridAll",
                    Use:            "grid-all",
                    Short:          "Returns all Grid Attributes",
                },
				{
                    RpcMethod:      "Guild",
                    Use:            "guild [guild id]",
                    Short:          "Show the details of a specific Allocation",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
                },
                {
                    RpcMethod:      "GuildAll",
                    Use:            "guild-all",
                    Short:          "Returns all Guilds",
                },
                {
                    RpcMethod:      "GuildMembershipApplication",
                    Use:            "guild-membership-application [guild id] [player id]",
                    Short:          "Show the details of a specific Membership Application",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "guildId"},{ProtoField: "playerId"}},
                },
                {
                    RpcMethod:      "GuildMembershipApplicationAll",
                    Use:            "guild-membership-application-all",
                    Short:          "Returns all Guild Membership Applications",
                },
				{
                    RpcMethod:      "Infusion",
                    Use:            "infusion [destination id] [address]",
                    Short:          "Show the details of a specific Infusion",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "destinationId"},{ProtoField: "address"}},
                },
                {
                    RpcMethod:      "InfusionAll",
                    Use:            "infusion-all",
                    Short:          "Returns all Infusions",
                },
                {
                    RpcMethod:      "InfusionAllByDestination",
                    Use:            "infusion-all-by-destination [destination id]",
                    Short:          "Returns all Infusions to a specific destination",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "destinationId"}},
                },
				{
					RpcMethod:      "Params",
					Use:            "params",
					Short:          "Shows the parameters of the module",
				},
				{
                    RpcMethod:      "Permission",
                    Use:            "permission [permission id]",
                    Short:          "Show the details of a specific Permission",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "permissionId"}},
                },
                {
                    RpcMethod:      "PermissionAll",
                    Use:            "permission-all",
                    Short:          "Returns all Permissions",
                },
				{
                    RpcMethod:      "PermissionByObject",
                    Use:            "permission-by-object [object id]",
                    Short:          "Show the details of a specific Permission",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "objectId"}},
                },
				{
                    RpcMethod:      "PermissionByPlayer",
                    Use:            "permission-by-player [player id]",
                    Short:          "Show the details of a specific Permission",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "playerId"}},
                },
				{
                    RpcMethod:      "Planet",
                    Use:            "planet [planet id]",
                    Short:          "Show the details of a specific Planet",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
                },
				{
                    RpcMethod:      "PlanetAllByPlayer",
                    Use:            "planet-all-by-player [player id]",
                    Short:          "Show all Planets belonging to a Player",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "playerId"}},
                },
                {
                    RpcMethod:      "PlanetAll",
                    Use:            "planet-all",
                    Short:          "Returns all Planets",
                },
				{
                    RpcMethod:      "PlanetAttribute",
                    Use:            "planet-attribute [planet id] [attribute type]",
                    Short:          "Show the details of a specific Planet Attribute",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "planetId"},{ProtoField: "attributeType"}},
                },

                {
                    RpcMethod:      "Player",
                    Use:            "player [player id]",
                    Short:          "Show the details of a specific Player",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
                },
                {
                    RpcMethod:      "PlayerAll",
                    Use:            "player-all",
                    Short:          "Returns all Players",
                },
				{
                    RpcMethod:      "Reactor",
                    Use:            "reactor [reactor id]",
                    Short:          "Show the details of a specific Reactor",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
                },
                {
                    RpcMethod:      "ReactorAll",
                    Use:            "reactor-all",
                    Short:          "Returns all Reactors",
                },
				{
                    RpcMethod:      "Struct",
                    Use:            "struct [struct id]",
                    Short:          "Show the details of a specific Struct",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
                },
                {
                    RpcMethod:      "StructAll",
                    Use:            "struct-all",
                    Short:          "Returns all Structs",
                },
				{
                    RpcMethod:      "StructAttribute",
                    Use:            "struct-attribute [struct id] [attribute type]",
                    Short:          "Show the details of a specific Struct Attribute",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"},{ProtoField: "attributeType"}},
                },
				{
                    RpcMethod:      "StructType",
                    Use:            "struct-type [struct type id]",
                    Short:          "Show the details of a specific Struct Type",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
                },
                {
                    RpcMethod:      "StructTypeAll",
                    Use:            "struct-type-all",
                    Short:          "Returns all Struct Types",
                },
				{
                    RpcMethod:      "Substation",
                    Use:            "substation [substation id]",
                    Short:          "Show the details of a specific Substation",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
                },
                {
                    RpcMethod:      "SubstationAll",
                    Use:            "substation-all",
                    Short:          "Returns all Substations",
                },
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
                {
                    RpcMethod:      "AddressRegister",
                    Use:            "address-register [address] [proof pubkey] [proof signature] [permissions] ",
                    Short:          "Submit a claim on an address, relating it to a player account",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"},{ProtoField: "proofPubKey"},{ProtoField: "proofSignature"},{ProtoField: "permissions"}},
                },
                {
                    RpcMethod:      "AddressRevoke",
                    Use:            "address-revoke [address]",
                    Short:          "Remove an address from a player account",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
                },
                {
                    RpcMethod:      "AllocationCreate",
                    Use:            "allocation-create [source id] [power]",
                    Short:          "Create an Allocation of energy from a power source",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "sourceObjectId"}, {ProtoField: "power"}},
                },
                {
                    RpcMethod:      "AllocationDelete",
                    Use:            "allocation-delete [allocation id]",
                    Short:          "Delete a dynamic Allocation",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "allocationId"}},
                },
                {
                    RpcMethod:      "AllocationTransfer",
                    Use:            "allocation-transfer [allocation id] [new controller address]",
                    Short:          "Transfer an Allocation to a different account",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "allocationId"}, {ProtoField: "controller"}},
                },
                {
                    RpcMethod:      "AllocationUpdate",
                    Use:            "allocation-update [allocation id] [power]",
                    Short:          "Update a dynamic Allocation",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "allocationId"}, {ProtoField: "power"}},
                },
                {
                    RpcMethod:      "FleetMove",
                    Use:            "fleet-move [fleet id] [destination location id]",
                    Short:          "Move a fleet from one planet to another",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "fleetId"}, {ProtoField: "destinationLocationId"}},
                },
                {
                    RpcMethod:      "GuildCreate",
                    Use:            "guild-create [endpoint] [substation id]",
                    Short:          "Create a guild from an account with an associated Reactor",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "endpoint"},{ProtoField: "entrySubstationId"}},
                },
                {
                    RpcMethod:      "GuildMembershipInvite",
                    Use:            "guild-membership-invite [player id]",
                    Short:          "Invite a player to a guild",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "playerId"}},
                },
                {
                    RpcMethod:      "GuildMembershipInviteApprove",
                    Use:            "guild-membership-invite-approve [guild id]",
                    Short:          "Accept an invitation to a guild",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "guildId"}},
                },
                {
                    RpcMethod:      "GuildMembershipInviteDeny",
                    Use:            "guild-membership-invite-deny [guild id]",
                    Short:          "Deny an invitation to a guild",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "guildId"}},
                },
                {
                    RpcMethod:      "GuildMembershipInviteRevoke",
                    Use:            "guild-membership [guild id] [player id]",
                    Short:          "Cancel an invite to a player",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "guildId"},{ProtoField: "playerId"}},
                },
                {
                    RpcMethod:      "GuildMembershipJoin",
                    Use:            "guild-membership-join [guild id] [infusion id,infusion id 2,...]",
                    Short:          "Join a guild with enough infusions to meet minimum requirements",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "guildId"},{ProtoField: "infusionId"}},
                },
                {
                    RpcMethod:      "GuildMembershipJoinProxy",
                    Use:            "guild-membership-join-proxy [address] [proof pubkey] [proof signature]",
                    Short:          "Add an account a guild and connect them with some power",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"},{ProtoField: "proofPubKey"},{ProtoField: "proofSignature"}},
                },
                {
                    RpcMethod:      "GuildMembershipKick",
                    Use:            "guild-membership-kick [player id]",
                    Short:          "Kick a player from a guild ",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "playerId"}},
                },
                {
                    RpcMethod:      "GuildMembershipRequest",
                    Use:            "guild-membership-request [guild id]",
                    Short:          "Request entry to a guild",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "guildId"}},
                },
                {
                    RpcMethod:      "GuildMembershipRequestApprove",
                    Use:            "guild-membership-request-approve [player id]",
                    Short:          "Accept a request from a player to join the guild",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "playerId"}},
                },
                {
                    RpcMethod:      "GuildMembershipRequestDeny",
                    Use:            "guild-membership-request-deny [player id]",
                    Short:          "Deny a request to join a guild",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "playerId"}},
                },
                {
                    RpcMethod:      "GuildMembershipRequestRevoke",
                    Use:            "guild-membership-request-revoke [guild id] [player id]",
                    Short:          "Destroy an application to join a guild",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "guildId"},{ProtoField: "playerId"}},
                },
                 {
                     RpcMethod:      "GuildUpdateEndpoint",
                     Use:            "guild-update-endpoint [guild id] [endpoint]",
                     Short:          "Update the endpoint Guild setting",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "guildId"},{ProtoField: "endpoint"}},
                 },
                 {
                     RpcMethod:      "GuildUpdateEntrySubstationId",
                     Use:            "guild-update-entry-substation-id [guild id] [entry substation id]",
                     Short:          "Update the entry substation Guild setting",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "guildId"},{ProtoField: "entrySubstationId"}},
                 },
                 {
                     RpcMethod:      "GuildUpdateJoinInfusionMinimum",
                     Use:            "guild-update-join-infusion-minimum [guild id] [join infusion minimum]",
                     Short:          "Update the infusion minimum Guild setting",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "guildId"},{ProtoField: "joinInfusionMinimum"}},
                 },
                 {
                     RpcMethod:      "GuildUpdateJoinInfusionMinimumBypassByRequest",
                     Use:            "guild-update-join-infusion-minimum-by-request [guild id] [join bypass level]",
                     Short:          "Update the minimum bypass level for requests Guild setting",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "guildId"},{ProtoField: "guildJoinBypassLevel"}},
                 },
                 {
                     RpcMethod:      "GuildUpdateJoinInfusionMinimumBypassByInvite",
                     Use:            "guild-update-join-infusion-minimum-by-invite [guild id] [join bypass level]",
                     Short:          "Update the minimum bypass level for invites Guild setting",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "guildId"},{ProtoField: "guildJoinBypassLevel"}},
                 },
                 {
                     RpcMethod:      "GuildUpdateOwnerId",
                     Use:            "guild-update-owner-id [guild id] [owner id]",
                     Short:          "Update the owner of the Guild",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "guildId"},{ProtoField: "owner"}},
                 },
                 {
                     RpcMethod:      "PermissionGrantOnObject",
                     Use:            "permission-grant-on-object [object id] [player id] [permissions]",
                     Short:          "Grant a set of permissions on an object to a player",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "objectId"},{ProtoField: "playerId"},{ProtoField: "permissions"}},
                 },
                 {
                     RpcMethod:      "PermissionGrantOnAddress",
                     Use:            "permission-grant-on-address [address] [permissions]",
                     Short:          "Grant a set of permissions to an address",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"},{ProtoField: "permissions"}},
                 },
                 {
                     RpcMethod:      "PermissionRevokeOnObject",
                     Use:            "permission-revoke-on-object [object id] [player id] [permissions]",
                     Short:          "Revoke a set of permissions on an object from a player",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "objectId"},{ProtoField: "playerId"},{ProtoField: "permissions"}},
                 },
                 {
                     RpcMethod:      "PermissionRevokeOnAddress",
                     Use:            "permission-revoke-on-address [address] [permissions]",
                     Short:          "Revoke a set of permissions on from an address",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"},{ProtoField: "permissions"}},
                 },
                 {
                     RpcMethod:      "PermissionSetOnObject",
                     Use:            "permission-set-on-object [object id] [player id] [permissions]",
                     Short:          "Clear previous permissions and apply a new full set on an object from a player",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "objectId"},{ProtoField: "playerId"},{ProtoField: "permissions"}},
                 },
                 {
                     RpcMethod:      "PermissionSetOnAddress",
                     Use:            "permission-set-on-address [address] [permissions]",
                     Short:          "Clear previous permissions and apply a new full set on from an address",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"},{ProtoField: "permissions"}},
                 },
                 {
                     RpcMethod:      "PlanetExplore",
                     Use:            "planet-explore",
                     Short:          "Explore a new planet",
                 },
                 {
                    RpcMethod:      "PlanetRaidComplete",
                    Use:            "planet-raid-complete [fleet id] [proof] [nonce]",
                    Short:          "Complete a Planet Raid",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "fleetId"},{ProtoField: "proof"},{ProtoField: "nonce"}},
                 },
                 {
                     RpcMethod:      "PlayerUpdatePrimaryAddress",
                     Use:            "player-update-primary-address [address]",
                     Short:          "Revoke a set of permissions on from an address",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "primaryAddress"}},
                 },
                 {
                     RpcMethod:      "StructActivate",
                     Use:            "struct-activate [struct id]",
                     Short:          "Bring a Struct online",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"}},
                 },
                 {
                     RpcMethod:      "StructDeactivate",
                     Use:            "struct-deactivate [struct id]",
                     Short:          "Take a Struct offline",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"}},
                 },
                 {
                     RpcMethod:      "StructAttack",
                     Use:            "struct-attack [operating struct id] [target struct id,target struct id2,...] [weapon system]",
                     Short:          "Attack a Struct with a Struct",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "operatingStructId"},{ProtoField: "targetStructId"},{ProtoField: "weaponSystem"}},
                 },
                 {
                     RpcMethod:      "StructBuildComplete",
                     Use:            "struct-build-complete [struct id] [proof] [nonce]",
                     Short:          "Bring a Struct online",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"},{ProtoField: "proof"},{ProtoField: "nonce"}},
                 },
                 {
                     RpcMethod:      "StructBuildInitiate",
                     Use:            "struct-build-initiate [player id] [struct type id] [location type] [operating ambit] [slot]",
                     Short:          "Initiate the construction of a Struct",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "playerId"},{ProtoField: "structTypeId"},{ProtoField: "locationType"},{ProtoField: "operatingAmbit"},{ProtoField: "slot", Optional: true }},
                 },
                 {
                     RpcMethod:      "StructDefenseClear",
                     Use:            "struct-defense-clear [defender struct id]",
                     Short:          "Clear the defensive relationship for a defending Struct",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "defenderStructId"}},
                 },
                 {
                     RpcMethod:      "StructDefenseSet",
                     Use:            "struct-defense-set [defender struct id] [protected struct id]",
                     Short:          "Set a defensive relationship for a Struct",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "defenderStructId"},{ProtoField: "protectedStructId"}},
                 },
                 {
                     RpcMethod:      "StructGeneratorInfuse",
                     Use:            "struct-generator-infuse [struct id] [infusion amount]",
                     Short:          "Infuse Alpha into a generating Struct (cannot be undone!)",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"},{ProtoField: "infuseAmount"}},
                 },
                 {
                     RpcMethod:      "StructMove",
                     Use:            "struct-move [struct id] [location type] [ambit] [slot]",
                     Short:          "Move a Struct to a different ambit, slot, or location",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"},{ProtoField: "locationType"},{ProtoField: "ambit"},{ProtoField: "slot", Optional: true }},
                 },
                 {
                     RpcMethod:      "StructOreMinerComplete",
                     Use:            "struct-ore-mine-complete [struct id] [proof] [nonce]",
                     Short:          "Complete a Struct mining action",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"},{ProtoField: "proof"},{ProtoField: "nonce"}},
                 },
                 {
                     RpcMethod:      "StructOreRefineryComplete",
                     Use:            "struct-ore-refine-complete [struct id] [proof] [nonce]",
                     Short:          "Complete a Struct refining action",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"},{ProtoField: "proof"},{ProtoField: "nonce"}},
                 },
                 {
                     RpcMethod:      "StructStealthActivate",
                     Use:            "struct-stealth-activate [struct id]",
                     Short:          "Activate the Stealth systems on a Struct",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"}},
                 },
                 {
                     RpcMethod:      "StructStealthDeactivate",
                     Use:            "struct-stealth-deactivate [struct id]",
                     Short:          "Deactivate the Stealth systems on a Struct",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"}},
                 },
                 {
                     RpcMethod:      "SubstationAllocationConnect",
                     Use:            "substation-allocation-connect [allocation id] [destination id]",
                     Short:          "Connect an Allocation to a Substation",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "allocationId"},{ProtoField: "destinationId"}},
                 },
                 {
                     RpcMethod:      "SubstationAllocationDisconnect",
                     Use:            "substation-allocation-disconnect [allocation id]",
                     Short:          "Disconnect an Allocation from a Substation",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "allocationId"}},
                 },
                 {
                     RpcMethod:      "SubstationCreate",
                     Use:            "substation-create [owner id] [allocation id]",
                     Short:          "Create a new Substation with an initial allocation",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "owner"},{ProtoField: "allocationId"}},
                 },
                 {
                     RpcMethod:      "SubstationDelete",
                     Use:            "substation-delete [substation id] [migration substation id]",
                     Short:          "Delete a Substation",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "substationId"},{ProtoField: "migrationSubstationId", Optional: true }},
                 },
                 {
                     RpcMethod:      "SubstationPlayerConnect",
                     Use:            "substation-player-connect [substation id] [player id]",
                     Short:          "Connect a Player to a Substation",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "substationId"},{ProtoField: "playerId"}},
                 },
                 {
                     RpcMethod:      "SubstationPlayerDisconnect",
                     Use:            "substation-player-disconnect [player id]",
                     Short:          "Disconnect a Player from a Substation",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "playerId"}},
                 },
                 {
                     RpcMethod:      "SubstationPlayerMigrate",
                     Use:            "substation-player-migrate [substation id] [player id,player id2,...]",
                     Short:          "Migrate a list of Players to another Substation",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "substationId"},{ProtoField: "playerId"}},
                 },

				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
