package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgStructRefine = "struct_refine"

var _ sdk.Msg = &MsgStructRefine{}


func NewMsgStructRefine(creator string, structId string, proof string, nonce string) *MsgStructRefine {
	return &MsgStructRefine{
		Creator:  creator,
		StructId: structId,
		Proof: proof,
		Nonce: nonce,
	}
}

func (msg *MsgStructRefine) Route() string {
	return RouterKey
}

func (msg *MsgStructRefine) Type() string {
	return TypeMsgStructRefine
}

func (msg *MsgStructRefine) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStructRefine) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStructRefine) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
