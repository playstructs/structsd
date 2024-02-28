package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgStructBuildInitiate = "struct_build_initiate"

var _ sdk.Msg = &MsgStructBuildInitiate{}

func NewMsgStructBuildInitiate(creator string, structType string, planetId string, slot uint64) *MsgStructBuildInitiate {
	return &MsgStructBuildInitiate{
		Creator:  creator,
		StructType: structType,
		PlanetId: planetId,
		Slot: slot,
	}
}

func (msg *MsgStructBuildInitiate) Route() string {
	return RouterKey
}

func (msg *MsgStructBuildInitiate) Type() string {
	return TypeMsgStructBuildInitiate
}

func (msg *MsgStructBuildInitiate) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStructBuildInitiate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStructBuildInitiate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
