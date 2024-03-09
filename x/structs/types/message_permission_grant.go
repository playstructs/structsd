package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgPermissionGrant = "permission_grant"

var _ sdk.Msg = &MsgPermissionGrant{}

func NewMsgPermissionGrant(creator string, objectId string, playerId string, permissions string) *MsgPermissionGrant {
	return &MsgPermissionGrant{
		Creator:     creator,
		ObjectId:    objectId,
		PlayerId:    playerId,
		Permissions: permissions,
	}
}

func (msg *MsgPermissionGrant) Route() string {
	return RouterKey
}

func (msg *MsgPermissionGrant) Type() string {
	return TypeMsgPermissionGrant
}

func (msg *MsgPermissionGrant) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgPermissionGrant) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgPermissionGrant) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
