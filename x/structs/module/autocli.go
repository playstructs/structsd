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
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
                    RpcMethod:      "Planet",
                    Use:            "planet [planet id]",
                    Short:          "Show the details of a specific Planet",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
                },
                {
                    RpcMethod:      "PlanetAll",
                    Use:            "planet-all",
                    Short:          "Returns all Planets",
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
                    RpcMethod:      "AddressApproveRegister",
                    Use:            "address-approve-register [approve] [address] [permissions]",
                    Short:          "Provide a decision on an address registration attempt",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{ {ProtoField: "approve"},{ProtoField: "address"},{ProtoField: "permissions", Optional: true }},
                },
                {
                    RpcMethod:      "AddressRegister",
                    Use:            "address-register [player id] [address]",
                    Short:          "Submit a claim on an address, relating it to a player account",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{ {ProtoField: "playerId"},{ProtoField: "address"}},
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
                     RpcMethod:      "PlanetExplore",
                     Use:            "planet-explore",
                     Short:          "Explore a new planet",
                 },
                 {
                     RpcMethod:      "PlayerUpdatePrimaryAddress",
                     Use:            "player-update-primary-address [address]",
                     Short:          "Revoke a set of permissions on from an address",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "primaryAddress"}},
                 },
                 {
                     RpcMethod:      "Sabotage",
                     Use:            "sabotage [struct id] [proof] [nonce]",
                     Short:          "Sabotage a struct!",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"},{ProtoField: "proof"},{ProtoField: "nonce"}},
                 },

                 {
                     RpcMethod:      "StructActivate",
                     Use:            "struct-activate [struct id]",
                     Short:          "Bring a Struct online",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"}},
                 },
                 {
                     RpcMethod:      "StructBuildComplete",
                     Use:            "struct-build-complete [struct id] [proof] [nonce]",
                     Short:          "Bring a Struct online",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"},{ProtoField: "proof"},{ProtoField: "nonce"}},
                 },
                 {
                     RpcMethod:      "StructBuildInitiate",
                     Use:            "struct-build-initiate [struct type] [planet id] [slot]",
                     Short:          "Initiate the construction of a new Struct",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structType"},{ProtoField: "planetId"},{ProtoField: "slot"}},
                 },
                 {
                     RpcMethod:      "StructInfuse",
                     Use:            "struct-infuse [struct id] [infusion amount]",
                     Short:          "Infuse Alpha into a generating Struct (cannot be undone!)",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"},{ProtoField: "infuseAmount"}},
                 },
                 {
                     RpcMethod:      "StructMine",
                     Use:            "struct-mine [struct id] [proof] [nonce]",
                     Short:          "Complete a Struct mining action",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"},{ProtoField: "proof"},{ProtoField: "nonce"}},
                 },
                 {
                     RpcMethod:      "StructMineActivate",
                     Use:            "struct-mine-activate [struct id]",
                     Short:          "Bring a Struct mining system online",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"}},
                 },
                 {
                     RpcMethod:      "StructMineDeactivate",
                     Use:            "struct-mine-deactivate [struct id]",
                     Short:          "Bring a Struct mining system offline",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"}},
                 },
                 {
                     RpcMethod:      "StructRefine",
                     Use:            "struct-refine [struct id] [proof] [nonce]",
                     Short:          "Complete a Struct refining action",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"},{ProtoField: "proof"},{ProtoField: "nonce"}},
                 },
                 {
                     RpcMethod:      "StructRefineActivate",
                     Use:            "struct-refine-activate [struct id]",
                     Short:          "Bring a Struct refinery system online",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "structId"}},
                 },
                 {
                     RpcMethod:      "StructRefineDeactivate",
                     Use:            "struct-refine-deactivate [struct id]",
                     Short:          "Bring a Struct refinery system offline",
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
