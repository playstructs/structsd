package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgStructActivate = "struct_activate"

var _ sdk.Msg = &MsgStructActivate{}


func NewMsgStructActivate(creator string, structId string) *MsgStructActivate {
	return &MsgStructActivate{
		Creator:  creator,
		StructId: structId,
	}
}

func (msg *MsgStructActivate) Route() string {
	return RouterKey
}

func (msg *MsgStructActivate) Type() string {
	return TypeMsgStructActivate
}

func (msg *MsgStructActivate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStructActivate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStructActivate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
