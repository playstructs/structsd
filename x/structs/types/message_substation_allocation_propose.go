package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSubstationAllocationPropose = "substation_allocation_propose"

var _ sdk.Msg = &MsgSubstationAllocationPropose{}

func NewMsgSubstationAllocationPropose(creator string, sourceType ObjectType, sourceId uint64, destinationId uint64, power sdk.Int) *MsgSubstationAllocationPropose {
	return &MsgSubstationAllocationPropose{
		Creator:    creator,
		SourceType: sourceType,
		SourceId:   sourceId,
		DestinationId: destinationId,
		Power:      power,
	}
}

func (msg *MsgSubstationAllocationPropose) Route() string {
	return RouterKey
}

func (msg *MsgSubstationAllocationPropose) Type() string {
	return TypeMsgSubstationAllocationPropose
}

func (msg *MsgSubstationAllocationPropose) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSubstationAllocationPropose) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubstationAllocationPropose) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

    if (!IsValidAllocationConnectionType(msg.SourceType)){
        return sdkerrors.Wrapf(ErrAllocationSourceType, "invalid source type (%s) for allocating power from", msg.SourceType.String())
    }

	return nil
}
