package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSubstationAllocationCreate = "substation_allocation_create"

var _ sdk.Msg = &MsgSubstationAllocationCreate{}

func NewMsgSubstationAllocationCreate(creator string, allocationId uint64, decision bool) *MsgSubstationAllocationCreate {
	return &MsgReactorAllocationActivate{
		Creator:      creator,
		AllocationId: allocationId,
		Decision:     decision,
	}
}

func (msg *MsgSubstationAllocationCreate) Route() string {
	return RouterKey
}

func (msg *MsgSubstationAllocationCreate) Type() string {
	return TypeMsgSubstationAllocationCreate
}

func (msg *MsgSubstationAllocationCreate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSubstationAllocationCreate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubstationAllocationCreate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
