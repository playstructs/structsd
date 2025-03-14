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
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgAddressRevoke{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgAgreementOpen{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgAgreementClose{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgAgreementCapacityIncrease{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgAgreementCapacityDecrease{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgAgreementDurationIncrease{},)

	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgAllocationCreate{}, )
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgAllocationDelete{}, )
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgAllocationUpdate{}, )
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgAllocationTransfer{}, )

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildCreate{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildBankMint{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildBankRedeem{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGuildBankConfiscateAndBurn{},)
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
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgPermissionSetOnObject{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgPermissionSetOnAddress{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgPlanetExplore{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgPlanetRaidComplete{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgPlayerUpdatePrimaryAddress{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgPlayerResume{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgProviderCreate{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgProviderWithdrawBalance{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgProviderUpdateCapacityMinimum{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgProviderUpdateCapacityMaximum{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgProviderUpdateDurationMinimum{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgProviderUpdateDurationMaximum{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgProviderUpdateAccessPolicy{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgProviderGuildGrant{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgProviderGuildRevoke{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgProviderDelete{},)


    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructActivate{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructDeactivate{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructBuildInitiate{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructBuildComplete{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructBuildCancel{},)
    // Not MVP
    //registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructBuildCompleteAndStash{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructDefenseSet{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructDefenseClear{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructMove{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructAttack{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructStealthActivate{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructStealthDeactivate{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructGeneratorInfuse{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructOreMinerComplete{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgStructOreRefineryComplete{},)

    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSubstationCreate{},)
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSubstationDelete{}, )
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSubstationAllocationConnect{}, )
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSubstationAllocationDisconnect{}, )
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSubstationPlayerConnect{}, )
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSubstationPlayerDisconnect{}, )
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSubstationPlayerMigrate{}, )

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
