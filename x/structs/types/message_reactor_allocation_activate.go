package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgReactorAllocationActivate = "reactor_allocation_activate"

var _ sdk.Msg = &MsgReactorAllocationActivate{}

func NewMsgReactorAllocationActivate(creator string, allocationId uint64, decision bool) *MsgReactorAllocationActivate {
	return &MsgReactorAllocationActivate{
		Creator:      creator,
		AllocationId: allocationId,
		Decision:     decision,
	}
}

func (msg *MsgReactorAllocationActivate) Route() string {
	return RouterKey
}

func (msg *MsgReactorAllocationActivate) Type() string {
	return TypeMsgReactorAllocationActivate
}

func (msg *MsgReactorAllocationActivate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgReactorAllocationActivate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgReactorAllocationActivate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
