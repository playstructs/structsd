package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

)

const TypeMsgAddressRegister = "address_register"

var _ sdk.Msg = &MsgAddressRegister{}

func NewMsgAddressRegister(creator string, playerId string, address string) *MsgAddressRegister {
	return &MsgAddressRegister{
		Creator:     creator,
		PlayerId:    playerId,
		Address:     address,

	}
}

func (msg *MsgAddressRegister) Route() string {
	return RouterKey
}

func (msg *MsgAddressRegister) Type() string {
	return TypeMsgAddressRegister
}

func (msg *MsgAddressRegister) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgAddressRegister) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAddressRegister) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
