package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgStructBuildComplete = "struct_build_complete"

var _ sdk.Msg = &MsgStructBuildComplete{}


func NewMsgStructBuildComplete(creator string, structId uint64, proof string, nonce string, activate bool) *MsgStructBuildComplete {
	return &MsgStructBuildComplete{
		Creator:  creator,
		StructId: structId,
		Proof: proof,
		Nonce: nonce,
		Activate: activate,
	}
}

func (msg *MsgStructBuildComplete) Route() string {
	return RouterKey
}

func (msg *MsgStructBuildComplete) Type() string {
	return TypeMsgStructBuildComplete
}

func (msg *MsgStructBuildComplete) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStructBuildComplete) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStructBuildComplete) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
