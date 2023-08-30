package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgStructRefineDeactivate = "struct_refine_deactivate"

var _ sdk.Msg = &MsgStructRefineDeactivate{}

func NewMsgStructRefineDeactivate(creator string, structId uint64) *MsgStructRefineDeactivate {
	return &MsgStructRefineDeactivate{
		Creator:  creator,
		StructId: structId,
	}
}

func (msg *MsgStructRefineDeactivate) Route() string {
	return RouterKey
}

func (msg *MsgStructRefineDeactivate) Type() string {
	return TypeMsgStructRefineDeactivate
}

func (msg *MsgStructRefineDeactivate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStructRefineDeactivate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStructRefineDeactivate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
