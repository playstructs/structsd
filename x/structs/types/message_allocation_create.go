package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgAllocationCreate = "allocation_create"

var _ sdk.Msg = &MsgAllocationCreate{}

func NewMsgAllocationCreate(creator string, allocationId uint64, decision bool) *MsgAllocationCreate {
	return &MsgAllocationActivate{
		Creator:      creator,
		AllocationId: allocationId,
		Decision:     decision,
	}
}

func (msg *MsgAllocationCreate) Route() string {
	return RouterKey
}

func (msg *MsgAllocationCreate) Type() string {
	return TypeMsgAllocationCreate
}

func (msg *MsgAllocationCreate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgAllocationCreate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAllocationCreate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
