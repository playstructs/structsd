package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgAddressApproveRegister = "address_approve_register"

var _ sdk.Msg = &MsgAddressApproveRegister{}

func NewMsgAddressApproveRegister(creator string, approve bool, address string, permissions uint64) *MsgAddressApproveRegister {
	return &MsgAddressApproveRegister{
		Creator:     creator,
		Approve:     approve,
		Address:     address,
		Permissions: permissions,
	}
}

func (msg *MsgAddressApproveRegister) Route() string {
	return RouterKey
}

func (msg *MsgAddressApproveRegister) Type() string {
	return TypeMsgAddressApproveRegister
}

func (msg *MsgAddressApproveRegister) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgAddressApproveRegister) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAddressApproveRegister) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
