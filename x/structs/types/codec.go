package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgAllocationCreate{}, "structs/AllocationCreate", nil)
	cdc.RegisterConcrete(&MsgSubstationCreate{}, "structs/SubstationCreate", nil)
	cdc.RegisterConcrete(&MsgSubstationDelete{}, "structs/SubstationDelete", nil)
	cdc.RegisterConcrete(&MsgSubstationAllocationConnect{}, "structs/SubstationAllocationConnect", nil)
	cdc.RegisterConcrete(&MsgSubstationAllocationDisconnect{}, "structs/SubstationAllocationDisconnect", nil)
	cdc.RegisterConcrete(&MsgSubstationPlayerConnect{}, "structs/SubstationPlayerConnect", nil)
	cdc.RegisterConcrete(&MsgSubstationPlayerDisconnect{}, "structs/SubstationPlayerDisconnect", nil)
	cdc.RegisterConcrete(&MsgGuildCreate{}, "structs/GuildCreate", nil)
	cdc.RegisterConcrete(&MsgGuildUpdateEndpoint{}, "structs/GuildUpdateEndpoint", nil)
	cdc.RegisterConcrete(&MsgGuildUpdateEntrySubstationId{}, "structs/GuildUpdateEntrySubstationId", nil)
	cdc.RegisterConcrete(&MsgGuildUpdateJoinInfusionMinimum{}, "structs/GuildUpdateJoinInfusionMinimum", nil)
	cdc.RegisterConcrete(&MsgGuildUpdateJoinInfusionMinimumBypassByInvite{}, "structs/GuildUpdateJoinInfusionMinimumBypassByInvite", nil)
	cdc.RegisterConcrete(&MsgGuildUpdateJoinInfusionMinimumBypassByRequest{}, "structs/GuildUpdateJoinInfusionMinimumBypassByRequest", nil)
	cdc.RegisterConcrete(&MsgGuildUpdateOwnerId{}, "structs/GuildUpdateOwnerId", nil)
	cdc.RegisterConcrete(&MsgGuildApproveRegister{}, "structs/GuildApproveRegister", nil)

	cdc.RegisterConcrete(&MsgGuildJoinProxy{}, "structs/GuildJoinProxy", nil)
	cdc.RegisterConcrete(&MsgGuildJoin{}, "structs/GuildJoin", nil)
	cdc.RegisterConcrete(&MsgAddressRegister{}, "structs/AddressRegister", nil)
	cdc.RegisterConcrete(&MsgAddressApproveRegister{}, "structs/AddressApproveRegister", nil)
	cdc.RegisterConcrete(&MsgAddressRevoke{}, "structs/AddressRevoke", nil)
	cdc.RegisterConcrete(&MsgPlanetExplore{}, "structs/PlanetExplore", nil)
	cdc.RegisterConcrete(&MsgStructActivate{}, "structs/StructActivate", nil)
	cdc.RegisterConcrete(&MsgStructBuildInitiate{}, "structs/StructBuildInitiate", nil)
	cdc.RegisterConcrete(&MsgStructBuildComplete{}, "structs/StructBuildComplete", nil)
	cdc.RegisterConcrete(&MsgStructMineActivate{}, "structs/StructMineActivate", nil)
    cdc.RegisterConcrete(&MsgStructMineDeactivate{}, "structs/StructMineDeactivate", nil)
    cdc.RegisterConcrete(&MsgStructMine{}, "structs/StructMine", nil)
	cdc.RegisterConcrete(&MsgStructRefineActivate{}, "structs/StructRefineActivate", nil)
    cdc.RegisterConcrete(&MsgStructRefineDeactivate{}, "structs/StructRefineDeactivate", nil)
    cdc.RegisterConcrete(&MsgStructRefine{}, "structs/StructRefine", nil)

    cdc.RegisterConcrete(&MsgPermissionGrantOnObject{}, "structs/PermissionGrantOnObject", nil)
    cdc.RegisterConcrete(&MsgPermissionGrantOnAddress{}, "structs/PermissionGrantOnAddress", nil)
    cdc.RegisterConcrete(&MsgPermissionRevokeOnObject{}, "structs/PermissionRevokeOnObject", nil)
    cdc.RegisterConcrete(&MsgPermissionRevokeOnAddress{}, "structs/PermissionRevokeOnAddress", nil)

    cdc.RegisterConcrete(&MsgSabotage{}, "structs/Sabotage", nil)


	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAllocationCreate{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubstationCreate{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubstationDelete{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubstationAllocationConnect{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubstationAllocationDisconnect{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubstationPlayerConnect{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubstationPlayerDisconnect{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgGuildCreate{},
	)
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgGuildUpdateEndpoint{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgGuildUpdateEntrySubstationId{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgGuildUpdateJoinInfusionMinimum{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgGuildUpdateJoinInfusionMinimumBypassByInvite{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgGuildUpdateJoinInfusionMinimumBypassByRequest{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgGuildUpdateOwnerId{},
    )


    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgGuildApproveRegister{},
    )
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgGuildJoinProxy{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgGuildJoin{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgPlayerUpdatePrimaryAddress{},
    )
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAddressRegister{},
	)
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgAddressApproveRegister{},
    )
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAddressRevoke{},
	)
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgPlanetExplore{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgStructActivate{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgStructBuildInitiate{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgStructBuildComplete{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgStructMineActivate{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgStructMineDeactivate{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgStructMine{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgStructRefineActivate{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgStructRefineDeactivate{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgStructRefine{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgStructInfuse{},
    )

    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgPermissionGrantOnObject{},
    )

    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgPermissionGrantOnAddress{},
    )

    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgPermissionRevokeOnObject{},
    )

    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgPermissionRevokeOnAddress{},
    )

    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgSabotage{},
    )

	// this line is used by starport scaffolding # 3

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
