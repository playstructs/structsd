package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgGuildUpdateJoinInfusionMinimumBypassByRequest = "guild_update_join_infusion_minimum_bypass_by_request"

var _ sdk.Msg = &MsgGuildUpdateJoinInfusionMinimumBypassByRequest{}

func NewMsgGuildUpdateJoinInfusionMinimumBypassByRequest(creator string, guildId string, guildJoinBypassLevel uint64) *MsgGuildUpdateJoinInfusionMinimumBypassByRequest {
	return &MsgGuildUpdateJoinInfusionMinimumBypassByRequest{
		Creator:  creator,
		GuildId: guildId,
		GuildJoinBypassLevel: guildJoinBypassLevel,
	}
}

func (msg *MsgGuildUpdateJoinInfusionMinimumBypassByRequest) Route() string {
	return RouterKey
}

func (msg *MsgGuildUpdateJoinInfusionMinimumBypassByRequest) Type() string {
	return TypeMsgGuildUpdateJoinInfusionMinimumBypassByRequest
}

func (msg *MsgGuildUpdateJoinInfusionMinimumBypassByRequest) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgGuildUpdateJoinInfusionMinimumBypassByRequest) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgGuildUpdateJoinInfusionMinimumBypassByRequest) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

    if (msg.GuildJoinBypassLevel <= GuildJoinBypassLevel_Invalid) {
        return sdkerrors.Wrapf(ErrInvalidGuildJoinBypassLevel, "Invalid guild join bypass level (%d), cannot be equal to or greater than (%d)", msg.GuildJoinBypassLevel, GuildJoinBypassLevel_Invalid )
    }
	return nil
}
