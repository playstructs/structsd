package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgReactorDelete = "reactor_delete"

var _ sdk.Msg = &MsgReactorDelete{}

func NewMsgReactorDelete(creator string, reactorId uint64) *MsgReactorDelete {
	return &MsgReactorDelete{
		Creator:   creator,
		ReactorId: reactorId,
	}
}

func (msg *MsgReactorDelete) Route() string {
	return RouterKey
}

func (msg *MsgReactorDelete) Type() string {
	return TypeMsgReactorDelete
}

func (msg *MsgReactorDelete) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgReactorDelete) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgReactorDelete) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
