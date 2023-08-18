package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSubstationCreate = "substation_create"

var _ sdk.Msg = &MsgSubstationCreate{}

func NewMsgSubstationCreate(creator string, owner uint64, playerConnectionAllocation uint64) *MsgSubstationCreate {
	return &MsgSubstationCreate{
		Creator:                    creator,
		Owner:                      owner,
		PlayerConnectionAllocation: playerConnectionAllocation,
	}
}

func (msg *MsgSubstationCreate) Route() string {
	return RouterKey
}

func (msg *MsgSubstationCreate) Type() string {
	return TypeMsgSubstationCreate
}

func (msg *MsgSubstationCreate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSubstationCreate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubstationCreate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
