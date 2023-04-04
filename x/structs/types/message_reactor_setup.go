package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgReactorSetup = "reactor_setup"

var _ sdk.Msg = &MsgReactorSetup{}

func NewMsgReactorSetup(creator string, validator string) *MsgReactorSetup {
	return &MsgReactorSetup{
		Creator:   creator,
		Validator: validator,
	}
}

func (msg *MsgReactorSetup) Route() string {
	return RouterKey
}

func (msg *MsgReactorSetup) Type() string {
	return TypeMsgReactorSetup
}

func (msg *MsgReactorSetup) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgReactorSetup) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgReactorSetup) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
