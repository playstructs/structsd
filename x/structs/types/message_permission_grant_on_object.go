package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgPermissionGrantOnObject = "permission_grant_on_object"

var _ sdk.Msg = &MsgPermissionGrantOnObject{}

func NewMsgPermissionGrantOnObject(creator string, objectId string, playerId string, permissions uint64) *MsgPermissionGrantOnObject {
	return &MsgPermissionGrantOnObject{
		Creator:     creator,
		ObjectId:    objectId,
		PlayerId:    playerId,
		Permissions: permissions,
	}
}

func (msg *MsgPermissionGrantOnObject) Route() string {
	return RouterKey
}

func (msg *MsgPermissionGrantOnObject) Type() string {
	return TypeMsgPermissionGrantOnObject
}

func (msg *MsgPermissionGrantOnObject) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgPermissionGrantOnObject) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgPermissionGrantOnObject) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
