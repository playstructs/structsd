package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgReactorAllocationActivate{}, "structs/ReactorAllocationActivate", nil)
	cdc.RegisterConcrete(&MsgSubstationCreate{}, "structs/SubstationCreate", nil)
	cdc.RegisterConcrete(&MsgSubstationDelete{}, "structs/SubstationDelete", nil)
	cdc.RegisterConcrete(&MsgSubstationAllocationPropose{}, "structs/SubstationAllocationPropose", nil)
	cdc.RegisterConcrete(&MsgSubstationAllocationActivate{}, "structs/SubstationAllocationActivate", nil)
	cdc.RegisterConcrete(&MsgSubstationAllocationDisconnect{}, "structs/SubstationAllocationDisconnect", nil)
	cdc.RegisterConcrete(&MsgSubstationPlayerConnect{}, "structs/SubstationPlayerConnect", nil)
	cdc.RegisterConcrete(&MsgSubstationPlayerDisconnect{}, "structs/SubstationPlayerDisconnect", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgReactorAllocationActivate{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubstationCreate{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubstationDelete{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubstationAllocationPropose{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubstationAllocationActivate{},
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
	// this line is used by starport scaffolding # 3

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
