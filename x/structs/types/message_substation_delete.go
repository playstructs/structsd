package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSubstationDelete = "substation_delete"

var _ sdk.Msg = &MsgSubstationDelete{}

func NewMsgSubstationDelete(creator string, substationId string, migrationSubstationId string) *MsgSubstationDelete {
	return &MsgSubstationDelete{
		Creator:               creator,
		SubstationId:          substationId,
		MigrationSubstationId: migrationSubstationId,
	}
}

func (msg *MsgSubstationDelete) Route() string {
	return RouterKey
}

func (msg *MsgSubstationDelete) Type() string {
	return TypeMsgSubstationDelete
}

func (msg *MsgSubstationDelete) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSubstationDelete) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubstationDelete) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
