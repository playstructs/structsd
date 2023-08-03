package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgPlayerCreate = "player_create"

var _ sdk.Msg = &MsgPlayerCreate{}

func NewMsgPlayerCreate(creator string, guildId uint64) *MsgPlayerCreate {
	return &MsgPlayerCreate{
		Creator: creator,
		GuildId: guildId,
	}
}

func (msg *MsgPlayerCreate) Route() string {
	return RouterKey
}

func (msg *MsgPlayerCreate) Type() string {
	return TypeMsgPlayerCreate
}

func (msg *MsgPlayerCreate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgPlayerCreate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgPlayerCreate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
