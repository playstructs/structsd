package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSubstationAllocationDisconnect = "substation_allocation_disconnect"

var _ sdk.Msg = &MsgSubstationAllocationDisconnect{}

func NewMsgSubstationAllocationDisconnect(creator string, allocationId uint64) *MsgSubstationAllocationDisconnect {
	return &MsgSubstationAllocationDisconnect{
		Creator:      creator,
		AllocationId: allocationId,
	}
}

func (msg *MsgSubstationAllocationDisconnect) Route() string {
	return RouterKey
}

func (msg *MsgSubstationAllocationDisconnect) Type() string {
	return TypeMsgSubstationAllocationDisconnect
}

func (msg *MsgSubstationAllocationDisconnect) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSubstationAllocationDisconnect) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubstationAllocationDisconnect) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
