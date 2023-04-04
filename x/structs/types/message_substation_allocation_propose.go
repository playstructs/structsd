package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSubstationAllocationPropose = "substation_allocation_propose"

var _ sdk.Msg = &MsgSubstationAllocationPropose{}

func NewMsgSubstationAllocationPropose(creator string, id uint64, sourceType string, sourceId uint64, power string) *MsgSubstationAllocationPropose {
	return &MsgSubstationAllocationPropose{
		Creator:    creator,
		Id:         id,
		SourceType: sourceType,
		SourceId:   sourceId,
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
	return nil
}
