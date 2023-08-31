package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgStructAllocationCreate = "struct_allocation_create"

var _ sdk.Msg = &MsgStructAllocationCreate{}

func NewMsgStructAllocationCreate(creator string, controller string, sourceId uint64, power uint64) *MsgStructAllocationCreate {
	return &MsgStructAllocationCreate{
		Creator:    creator,
		Controller: controller,
		SourceId:   sourceId,
		Power:      power,
	}
}

func (msg *MsgStructAllocationCreate) Route() string {
	return RouterKey
}

func (msg *MsgStructAllocationCreate) Type() string {
	return TypeMsgStructAllocationCreate
}

func (msg *MsgStructAllocationCreate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStructAllocationCreate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStructAllocationCreate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
