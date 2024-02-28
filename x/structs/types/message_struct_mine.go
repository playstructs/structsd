package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgStructMine = "struct_mine"

var _ sdk.Msg = &MsgStructMine{}


func NewMsgStructMine(creator string, structId string, proof string, nonce string) *MsgStructMine {
	return &MsgStructMine{
		Creator:  creator,
		StructId: structId,
		Proof: proof,
		Nonce: nonce,
	}
}

func (msg *MsgStructMine) Route() string {
	return RouterKey
}

func (msg *MsgStructMine) Type() string {
	return TypeMsgStructMine
}

func (msg *MsgStructMine) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStructMine) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStructMine) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
