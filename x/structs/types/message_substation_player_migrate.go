package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSubstationPlayerMigrate = "substation_player_migrate"

var _ sdk.Msg = &MsgSubstationPlayerMigrate{}

func NewMsgSubstationPlayerMigrate(creator string, substationId string, playerId []string) *MsgSubstationPlayerMigrate {
	return &MsgSubstationPlayerMigrate{
		Creator:      creator,
		SubstationId: substationId,
		PlayerId:     playerId,
	}
}

func (msg *MsgSubstationPlayerMigrate) Route() string {
	return RouterKey
}

func (msg *MsgSubstationPlayerMigrate) Type() string {
	return TypeMsgSubstationPlayerMigrate
}

func (msg *MsgSubstationPlayerMigrate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSubstationPlayerMigrate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubstationPlayerMigrate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
