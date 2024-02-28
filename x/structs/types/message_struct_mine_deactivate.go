package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgStructMineDeactivate = "struct_mine_deactivate"

var _ sdk.Msg = &MsgStructMineDeactivate{}

func NewMsgStructMineDeactivate(creator string, structId string) *MsgStructMineDeactivate {
	return &MsgStructMineDeactivate{
		Creator:  creator,
		StructId: structId,
	}
}

func (msg *MsgStructMineDeactivate) Route() string {
	return RouterKey
}

func (msg *MsgStructMineDeactivate) Type() string {
	return TypeMsgStructMineDeactivate
}

func (msg *MsgStructMineDeactivate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStructMineDeactivate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStructMineDeactivate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
