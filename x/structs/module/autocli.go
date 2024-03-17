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
                    RpcMethod:      "Guild",
                    Use:            "guild [guild id]",
                    Short:          "Show the details of a specific Allocation",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
                },
				{
                    RpcMethod:      "Infusion",
                    Use:            "infusion [destination id] [address]",
                    Short:          "Show the details of a specific Infusion",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "destinationId"},{ProtoField: "address"}},
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
                    RpcMethod:      "Reactor",
                    Use:            "reactor [reactor id]",
                    Short:          "Show the details of a specific Reactor",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
                },
				{
                    RpcMethod:      "Struct",
                    Use:            "struct [struct id]",
                    Short:          "Show the details of a specific Struct",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
                },
				{
                    RpcMethod:      "Substation",
                    Use:            "substation [substation id]",
                    Short:          "Show the details of a specific Substation",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
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
                    RpcMethod:      "GuildApproveRegister",
                    Use:            "guild-approve-register [approve] [guild id] [player id]",
                    Short:          "Provide a decision on an guild registration attempt",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "approve"},{ProtoField: "guildId"},{ProtoField: "playerId"}},
                },
                {
                    RpcMethod:      "GuildCreate",
                    Use:            "guild-create [endpoint] [substation id]",
                    Short:          "Create a guild from an account with an associated Reactor",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "endpoint"},{ProtoField: "entrySubstationId"}},
                },
                {
                    RpcMethod:      "GuildJoin",
                    Use:            "guild-join [guild id]",
                    Short:          "Join a guild, or at least try to",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "guildId"}},
                },
                {
                     RpcMethod:      "GuildJoinProxy",
                     Use:            "guild-join-proxy [address]",
                     Short:          "Join a non-player to the guild, or at least try to",
                     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
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
                     Use:            "struct-build-complete [struct type] [planet id] [slot]",
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
                // TODO GUIld commands onwards

				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
