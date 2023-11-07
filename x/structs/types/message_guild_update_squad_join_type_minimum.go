package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgGuildUpdateSquadJoinTypeMinimum = "guild_update_squad_join_type_minimum"

var _ sdk.Msg = &MsgGuildUpdateSquadJoinTypeMinimum{}

func NewMsgGuildUpdateSquadJoinTypeMinimum(creator string, id uint64, squadJoinTypeMinimum uint64) *MsgGuildUpdateSquadJoinTypeMinimum {
	return &MsgGuildUpdateSquadJoinTypeMinimum{
		Creator:  creator,
		Id: id,
		SquadJoinTypeMinimum: squadJoinTypeMinimum,
	}
}

func (msg *MsgGuildUpdateSquadJoinTypeMinimum) Route() string {
	return RouterKey
}

func (msg *MsgGuildUpdateSquadJoinTypeMinimum) Type() string {
	return TypeMsgGuildUpdateSquadJoinTypeMinimum
}

func (msg *MsgGuildUpdateSquadJoinTypeMinimum) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgGuildUpdateSquadJoinTypeMinimum) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgGuildUpdateSquadJoinTypeMinimum) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
