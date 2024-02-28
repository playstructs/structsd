package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgGuildUpdateOwnerId = "guild_update_owner_id"

var _ sdk.Msg = &MsgGuildUpdateOwnerId{}

func NewMsgGuildUpdateOwnerId(creator string, guildId string, ownerId string) *MsgGuildUpdateOwnerId {
	return &MsgGuildUpdateOwnerId{
		Creator:  creator,
		GuildId: guildId,
		Owner: ownerId,
	}
}

func (msg *MsgGuildUpdateOwnerId) Route() string {
	return RouterKey
}

func (msg *MsgGuildUpdateOwnerId) Type() string {
	return TypeMsgGuildUpdateOwnerId
}

func (msg *MsgGuildUpdateOwnerId) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgGuildUpdateOwnerId) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgGuildUpdateOwnerId) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
