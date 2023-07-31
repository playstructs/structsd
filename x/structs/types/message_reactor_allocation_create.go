package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgReactorAllocationCreate = "reactor_allocation_create"

var _ sdk.Msg = &MsgReactorAllocationCreate{}

func NewMsgReactorAllocationCreate(creator string, controller string, sourceId uint64, power uint64) *MsgReactorAllocationCreate {
	return &MsgReactorAllocationCreate{
		Creator:    creator,
		Controller: controller,
		SourceId:   sourceId,
		Power:      power,
	}
}

func (msg *MsgReactorAllocationCreate) Route() string {
	return RouterKey
}

func (msg *MsgReactorAllocationCreate) Type() string {
	return TypeMsgReactorAllocationCreate
}

func (msg *MsgReactorAllocationCreate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgReactorAllocationCreate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgReactorAllocationCreate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}