package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgAddressRevoke = "address_revoke"

var _ sdk.Msg = &MsgAddressRevoke{}

func NewMsgAddressRevoke(creator string, address string) *MsgAddressRevoke {
	return &MsgAddressRevoke{
		Creator:  creator,
		Address:  address,
	}
}

func (msg *MsgAddressRevoke) Route() string {
	return RouterKey
}

func (msg *MsgAddressRevoke) Type() string {
	return TypeMsgAddressRevoke
}

func (msg *MsgAddressRevoke) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgAddressRevoke) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAddressRevoke) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
