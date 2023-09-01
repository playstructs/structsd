package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSabotage = "sabotage"

var _ sdk.Msg = &MsgSabotage{}


func NewMsgSabotage(creator string, structId uint64, proof string, nonce string) *MsgSabotage {
	return &MsgSabotage{
		Creator:  creator,
		StructId: structId,
		Proof: proof,
		Nonce: nonce,
	}
}

func (msg *MsgSabotage) Route() string {
	return RouterKey
}

func (msg *MsgSabotage) Type() string {
	return TypeMsgSabotage
}

func (msg *MsgSabotage) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSabotage) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSabotage) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
