package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgPlayerCreateProxy = "player_create_proxy"

var _ sdk.Msg = &MsgPlayerCreateProxy{}

func NewMsgPlayerCreateProxy(creator string, address string, proof string) *MsgPlayerCreateProxy {
	return &MsgPlayerCreateProxy{
		Creator:      creator,
		Address:      address,
		Proof:        proof,
	}
}

func (msg *MsgPlayerCreateProxy) Route() string {
	return RouterKey
}

func (msg *MsgPlayerCreateProxy) Type() string {
	return TypeMsgPlayerCreateProxy
}

func (msg *MsgPlayerCreateProxy) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgPlayerCreateProxy) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgPlayerCreateProxy) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
