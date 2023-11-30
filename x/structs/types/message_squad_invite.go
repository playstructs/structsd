package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSquadInvite = "squad_invite"

var _ sdk.Msg = &MsgSquadInvite{}

func NewMsgSquadInvite(creator string, squadId uint64, playerId uint64) *MsgSquadInvite {
	return &MsgSquadInvite{
		Creator:  creator,
		SquadId: squadId,
		PlayerId: playerId,
	}
}

func (msg *MsgSquadInvite) Route() string {
	return RouterKey
}

func (msg *MsgSquadInvite) Type() string {
	return TypeMsgSquadInvite
}

func (msg *MsgSquadInvite) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSquadInvite) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSquadInvite) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
