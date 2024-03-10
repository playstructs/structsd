package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgPermissionRevokeOnAddress = "permission_revoke_on_address"

var _ sdk.Msg = &MsgPermissionRevokeOnAddress{}

func NewMsgPermissionRevokeOnAddress(creator string, address string, permissions uint64) *MsgPermissionRevokeOnAddress {
	return &MsgPermissionRevokeOnAddress{
		Creator:     creator,
		Address:     address,
		Permissions: permissions,
	}
}

func (msg *MsgPermissionRevokeOnAddress) Route() string {
	return RouterKey
}

func (msg *MsgPermissionRevokeOnAddress) Type() string {
	return TypeMsgPermissionRevokeOnAddress
}

func (msg *MsgPermissionRevokeOnAddress) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgPermissionRevokeOnAddress) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgPermissionRevokeOnAddress) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
