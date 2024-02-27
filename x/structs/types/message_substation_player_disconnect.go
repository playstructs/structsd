package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSubstationPlayerDisconnect = "substation_player_disconnect"

var _ sdk.Msg = &MsgSubstationPlayerDisconnect{}

func NewMsgSubstationPlayerDisconnect(creator string, playerId string) *MsgSubstationPlayerDisconnect {
	return &MsgSubstationPlayerDisconnect{
		Creator:  creator,
		PlayerId: playerId,
	}
}

func (msg *MsgSubstationPlayerDisconnect) Route() string {
	return RouterKey
}

func (msg *MsgSubstationPlayerDisconnect) Type() string {
	return TypeMsgSubstationPlayerDisconnect
}

func (msg *MsgSubstationPlayerDisconnect) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSubstationPlayerDisconnect) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubstationPlayerDisconnect) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
