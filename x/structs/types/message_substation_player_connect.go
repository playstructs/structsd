package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSubstationPlayerConnect = "substation_player_connect"

var _ sdk.Msg = &MsgSubstationPlayerConnect{}

func NewMsgSubstationPlayerConnect(creator string, substationId string, playerId string) *MsgSubstationPlayerConnect {
	return &MsgSubstationPlayerConnect{
		Creator:      creator,
		SubstationId: substationId,
		PlayerId:     playerId,
	}
}

func (msg *MsgSubstationPlayerConnect) Route() string {
	return RouterKey
}

func (msg *MsgSubstationPlayerConnect) Type() string {
	return TypeMsgSubstationPlayerConnect
}

func (msg *MsgSubstationPlayerConnect) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSubstationPlayerConnect) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubstationPlayerConnect) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
