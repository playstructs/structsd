package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgStructMineActivate = "struct_mine_activate"

var _ sdk.Msg = &MsgStructMineActivate{}

func NewMsgStructMineActivate(creator string, structId uint64) *MsgStructMineActivate {
	return &MsgStructMineActivate{
		Creator:  creator,
		StructId: structId,
	}
}

func (msg *MsgStructMineActivate) Route() string {
	return RouterKey
}

func (msg *MsgStructMineActivate) Type() string {
	return TypeMsgStructMineActivate
}

func (msg *MsgStructMineActivate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStructMineActivate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStructMineActivate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
