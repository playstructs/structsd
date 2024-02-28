package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgStructInfuse = "struct_infuse"

var _ sdk.Msg = &MsgStructInfuse{}


func NewMsgStructInfuse(creator string, structId string, infuseAmount string) *MsgStructInfuse {
	return &MsgStructInfuse{
		Creator:  creator,
		StructId: structId,
		InfuseAmount: infuseAmount,
	}
}

func (msg *MsgStructInfuse) Route() string {
	return RouterKey
}

func (msg *MsgStructInfuse) Type() string {
	return TypeMsgStructInfuse
}

func (msg *MsgStructInfuse) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStructInfuse) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStructInfuse) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
