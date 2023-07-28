package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSubstationAllocationConnect= "substation_allocation_connect"

var _ sdk.Msg = &MsgSubstationAllocationConnect{}

func NewMsgSubstationAllocationConnect(creator string, allocationId uint64, destinationId uint64) *MsgSubstationAllocationConnect {
	return &MsgSubstationAllocationConnect{
		Creator:    creator,
		AllocationId: sourceType,
		DestinationId: destinationId,
	}
}

func (msg *MsgSubstationAllocationConnect) Route() string {
	return RouterKey
}

func (msg *MsgSubstationAllocationConnect) Type() string {
	return TypeMsgSubstationAllocationConnect
}

func (msg *MsgSubstationAllocationConnect) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSubstationAllocationConnect) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubstationAllocationConnect) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

    if (!IsValidAllocationConnectionType(msg.SourceType)){
        return sdkerrors.Wrapf(ErrAllocationSourceType, "invalid source type (%s) for allocating power from", msg.SourceType.String())
    }

	return nil
}
