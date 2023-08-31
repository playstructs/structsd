package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgPlayerUpdatePrimaryAddress = "player_update_primary_address"

var _ sdk.Msg = &MsgPlayerUpdatePrimaryAddress{}

func NewMsgPlayerUpdatePrimaryAddress(creator string, primaryAddress string) *MsgPlayerUpdatePrimaryAddress {
	return &MsgPlayerUpdatePrimaryAddress{
		Creator: creator,
		PrimaryAddress: primaryAddress,
	}
}

func (msg *MsgPlayerUpdatePrimaryAddress) Route() string {
	return RouterKey
}

func (msg *MsgPlayerUpdatePrimaryAddress) Type() string {
	return TypeMsgPlayerUpdatePrimaryAddress
}

func (msg *MsgPlayerUpdatePrimaryAddress) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgPlayerUpdatePrimaryAddress) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgPlayerUpdatePrimaryAddress) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
