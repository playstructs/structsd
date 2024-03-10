package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgPermissionRevokeOnObject = "permission_revoke_on_object"

var _ sdk.Msg = &MsgPermissionRevokeOnObject{}

func NewMsgPermissionRevokeOnObject(creator string, objectId string, playerId string, permissions uint64) *MsgPermissionRevokeOnObject {
	return &MsgPermissionRevokeOnObject{
		Creator:     creator,
		ObjectId:    objectId,
		PlayerId:    playerId,
		Permissions: permissions,
	}
}

func (msg *MsgPermissionRevokeOnObject) Route() string {
	return RouterKey
}

func (msg *MsgPermissionRevokeOnObject) Type() string {
	return TypeMsgPermissionRevokeOnObject
}

func (msg *MsgPermissionRevokeOnObject) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgPermissionRevokeOnObject) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgPermissionRevokeOnObject) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
