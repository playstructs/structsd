package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgAllocationCreate{}, "structs/AllocationCreate", nil)
	cdc.RegisterConcrete(&MsgReactorAllocationCreate{}, "structs/ReactorAllocationCreate", nil)
	cdc.RegisterConcrete(&MsgSubstationCreate{}, "structs/SubstationCreate", nil)
	cdc.RegisterConcrete(&MsgSubstationDelete{}, "structs/SubstationDelete", nil)
	cdc.RegisterConcrete(&MsgSubstationAllocationCreate{}, "structs/SubstationAllocationCreate", nil)
	cdc.RegisterConcrete(&MsgSubstationAllocationConnect{}, "structs/SubstationAllocationConnect", nil)
	cdc.RegisterConcrete(&MsgSubstationAllocationDisconnect{}, "structs/SubstationAllocationDisconnect", nil)
	cdc.RegisterConcrete(&MsgSubstationPlayerConnect{}, "structs/SubstationPlayerConnect", nil)
	cdc.RegisterConcrete(&MsgSubstationPlayerDisconnect{}, "structs/SubstationPlayerDisconnect", nil)
	cdc.RegisterConcrete(&MsgGuildCreate{}, "structs/GuildCreate", nil)
	cdc.RegisterConcrete(&MsgGuildApproveRegister{}, "structs/GuildApproveRegister", nil)
	cdc.RegisterConcrete(&MsgPlayerCreateProxy{}, "structs/PlayerCreateProxy", nil)
	cdc.RegisterConcrete(&MsgPlayerCreate{}, "structs/PlayerCreate", nil)
	cdc.RegisterConcrete(&MsgAddressRegister{}, "structs/AddressRegister", nil)
	cdc.RegisterConcrete(&MsgAddressApproveRegister{}, "structs/AddressApproveRegister", nil)
	cdc.RegisterConcrete(&MsgAddressRevoke{}, "structs/AddressRevoke", nil)
	cdc.RegisterConcrete(&MsgPlanetExplore{}, "structs/PlanetExplore", nil)
	cdc.RegisterConcrete(&MsgStructBuildInitiate{}, "structs/StructBuildInitiate", nil)
	cdc.RegisterConcrete(&MsgStructBuildComplete{}, "structs/StructBuildComplete", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAllocationCreate{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgReactorAllocationCreate{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubstationCreate{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubstationDelete{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubstationAllocationCreate{},
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
        &MsgGuildApproveRegister{},
    )
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgPlayerCreateProxy{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgPlayerCreate{},
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
        &MsgStructBuildInitiate{},
    )
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgStructBuildComplete{},
    )
	// this line is used by starport scaffolding # 3

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
