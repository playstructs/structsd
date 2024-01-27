package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgGuildUpdateJoinType = "guild_update_join_type"

var _ sdk.Msg = &MsgGuildUpdateJoinType{}

func NewMsgGuildUpdateJoinType(creator string, id uint64, guildJoinType uint64) *MsgGuildUpdateJoinType {
	return &MsgGuildUpdateJoinType{
		Creator:  creator,
		Id: id,
		GuildJoinType: guildJoinType,
	}
}

func (msg *MsgGuildUpdateJoinType) Route() string {
	return RouterKey
}

func (msg *MsgGuildUpdateJoinType) Type() string {
	return TypeMsgGuildUpdateJoinType
}

func (msg *MsgGuildUpdateJoinType) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgGuildUpdateJoinType) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgGuildUpdateJoinType) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
