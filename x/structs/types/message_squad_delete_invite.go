package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSquadDeleteInvite = "squad_delete_invite"

var _ sdk.Msg = &MsgSquadDeleteInvite{}

func NewMsgSquadDeleteInvite(creator string, squadId uint64, playerId uint64) *MsgSquadDeleteInvite {
	return &MsgSquadDeleteInvite{
		Creator:  creator,
		SquadId: squadId,
		PlayerId: playerId,
	}
}

func (msg *MsgSquadDeleteInvite) Route() string {
	return RouterKey
}

func (msg *MsgSquadDeleteInvite) Type() string {
	return TypeMsgSquadDeleteInvite
}

func (msg *MsgSquadDeleteInvite) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSquadDeleteInvite) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSquadDeleteInvite) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
