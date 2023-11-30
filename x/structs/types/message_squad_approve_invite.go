package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSquadApproveInvite = "squad_approve_invite"

var _ sdk.Msg = &MsgSquadApproveInvite{}

func NewMsgSquadApproveInvite(creator string, squadId uint64, playerId uint64, approve bool) *MsgSquadApproveInvite {
	return &MsgSquadApproveInvite{
		Creator:     creator,
		Approve:     approve,
		SquadId:     squadId,
        PlayerId:    playerId,
	}
}

func (msg *MsgSquadApproveInvite) Route() string {
	return RouterKey
}

func (msg *MsgSquadApproveInvite) Type() string {
	return TypeMsgSquadApproveInvite
}

func (msg *MsgSquadApproveInvite) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSquadApproveInvite) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSquadApproveInvite) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
