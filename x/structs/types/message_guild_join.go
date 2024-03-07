package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgGuildJoin = "guild_join"

var _ sdk.Msg = &MsgGuildJoin{}

func NewMsgGuildJoin(creator string, guildId string) *MsgGuildJoin {
	return &MsgGuildJoin{
		Creator: creator,
		GuildId: guildId,
	}
}

func (msg *MsgGuildJoin) Route() string {
	return RouterKey
}

func (msg *MsgGuildJoin) Type() string {
	return TypeMsgGuildJoin
}

func (msg *MsgGuildJoin) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgGuildJoin) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgGuildJoin) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
