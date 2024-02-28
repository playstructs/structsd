package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgStructRefineActivate = "struct_refine_activate"

var _ sdk.Msg = &MsgStructRefineActivate{}

func NewMsgStructRefineActivate(creator string, structId string) *MsgStructRefineActivate {
	return &MsgStructRefineActivate{
		Creator:  creator,
		StructId: structId,
	}
}

func (msg *MsgStructRefineActivate) Route() string {
	return RouterKey
}

func (msg *MsgStructRefineActivate) Type() string {
	return TypeMsgStructRefineActivate
}

func (msg *MsgStructRefineActivate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStructRefineActivate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStructRefineActivate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
