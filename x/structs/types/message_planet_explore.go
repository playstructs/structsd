package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgPlanetExplore = "planet_explore"

var _ sdk.Msg = &MsgPlanetExplore{}

func NewMsgPlanetExplore(creator string) *MsgPlanetExplore {
	return &MsgPlanetExplore{
		Creator:  creator,
	}
}

func (msg *MsgPlanetExplore) Route() string {
	return RouterKey
}

func (msg *MsgPlanetExplore) Type() string {
	return TypeMsgPlanetExplore
}

func (msg *MsgPlanetExplore) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgPlanetExplore) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgPlanetExplore) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
