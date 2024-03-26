package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	// this line is used by starport scaffolding # 1
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// this line is used by starport scaffolding # 3

	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
	)

	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgAddressRegister{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgAddressApproveRegister{}, )
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgAddressRevoke{},)

	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgAllocationCreate{}, )

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildCreate{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildUpdateEndpoint{}, )
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildUpdateEntrySubstationId{}, )
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildUpdateJoinInfusionMinimum{}, )
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildUpdateJoinInfusionMinimumBypassByInvite{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildUpdateJoinInfusionMinimumBypassByRequest{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildUpdateOwnerId{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildMembershipInvite{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildMembershipInviteApprove{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildMembershipInviteDeny{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildMembershipInviteRevoke{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildMembershipJoin{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildMembershipJoinProxy{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildMembershipKick{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildMembershipRequest{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildMembershipRequestApprove{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildMembershipRequestDeny{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildMembershipRequestRevoke{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgPermissionGrantOnObject{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgPermissionGrantOnAddress{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgPermissionRevokeOnObject{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgPermissionRevokeOnAddress{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgPlanetExplore{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgPlayerUpdatePrimaryAddress{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructActivate{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructBuildInitiate{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructBuildComplete{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructMineActivate{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructMineDeactivate{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructMine{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructRefineActivate{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructRefineDeactivate{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructRefine{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructInfuse{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSubstationCreate{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSubstationDelete{}, )
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSubstationAllocationConnect{}, )
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSubstationAllocationDisconnect{}, )
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSubstationPlayerConnect{}, )
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSubstationPlayerDisconnect{}, )
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSubstationPlayerMigrate{}, )

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSabotage{},)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	// ModuleCdc references the global x/ibc-transfer module codec. Note, the codec
	// should ONLY be used in certain instances of tests and for JSON encoding.
	//
	// The actual codec used for serialization should be provided to x/ibc transfer and
	// defined at the application level.
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
