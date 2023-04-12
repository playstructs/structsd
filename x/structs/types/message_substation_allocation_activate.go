package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSubstationAllocationActivate = "substation_allocation_activate"

var _ sdk.Msg = &MsgSubstationAllocationActivate{}

func NewMsgSubstationAllocationActivate(creator string, allocationId uint64, decision bool) *MsgSubstationAllocationActivate {
	return &MsgSubstationAllocationActivate{
		Creator:      creator,
		AllocationId: allocationId,
		Decision:     decision,
	}
}

func (msg *MsgSubstationAllocationActivate) Route() string {
	return RouterKey
}

func (msg *MsgSubstationAllocationActivate) Type() string {
	return TypeMsgSubstationAllocationActivate
}

func (msg *MsgSubstationAllocationActivate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSubstationAllocationActivate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubstationAllocationActivate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	return nil
}
