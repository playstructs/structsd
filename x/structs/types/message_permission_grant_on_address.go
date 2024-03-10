package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgPermissionGrantOnAddress = "permission_grant_on_address"

var _ sdk.Msg = &MsgPermissionGrantOnAddress{}

func NewMsgPermissionGrantOnAddress(creator string, address string, permissions uint64) *MsgPermissionGrantOnAddress {
	return &MsgPermissionGrantOnAddress{
		Creator:     creator,
		Address:     address,
		Permissions: permissions,
	}
}

func (msg *MsgPermissionGrantOnAddress) Route() string {
	return RouterKey
}

func (msg *MsgPermissionGrantOnAddress) Type() string {
	return TypeMsgPermissionGrantOnAddress
}

func (msg *MsgPermissionGrantOnAddress) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgPermissionGrantOnAddress) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgPermissionGrantOnAddress) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
