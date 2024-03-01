package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgGuildUpdateJoinInfusionMinimumBypassByInvite = "guild_update_join_infusion_minimum_bypass_by_invite"

var _ sdk.Msg = &MsgGuildUpdateJoinInfusionMinimumBypassByInvite{}

func NewMsgGuildUpdateJoinInfusionMinimumBypassByInvite(creator string, guildId string, guildJoinBypassLevel GuildJoinBypassLevel) *MsgGuildUpdateJoinInfusionMinimumBypassByInvite {
	return &MsgGuildUpdateJoinInfusionMinimumBypassByInvite{
		Creator:  creator,
		GuildId: guildId,
		GuildJoinBypassLevel: guildJoinBypassLevel,
	}
}

func (msg *MsgGuildUpdateJoinInfusionMinimumBypassByInvite) Route() string {
	return RouterKey
}

func (msg *MsgGuildUpdateJoinInfusionMinimumBypassByInvite) Type() string {
	return TypeMsgGuildUpdateJoinInfusionMinimumBypassByInvite
}

func (msg *MsgGuildUpdateJoinInfusionMinimumBypassByInvite) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgGuildUpdateJoinInfusionMinimumBypassByInvite) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgGuildUpdateJoinInfusionMinimumBypassByInvite) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	return nil
}
