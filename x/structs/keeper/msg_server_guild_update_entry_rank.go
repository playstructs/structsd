package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildUpdateEntryRank(goCtx context.Context, msg *types.MsgGuildUpdateEntryRank) (*types.MsgGuildUpdateResponse, error) {
	emptyResponse := &types.MsgGuildUpdateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	k.AddressEmitActivity(ctx, msg.Creator)

	player, err := cc.GetSigningPlayer(msg.Creator)
	if err != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "guild_update_entry_rank")
	}

	guildId := msg.GuildId
	if guildId == "" {
		if player.GetGuildId() == "" {
			return emptyResponse, types.NewGuildMembershipError("", player.GetPlayerId(), "not_member")
		}
		guildId = player.GetGuildId()
	}

	guild := cc.GetGuild(guildId)
	if guild.CheckGuild() != nil {
		return emptyResponse, types.NewObjectNotFoundError("guild", guildId)
	}

	guildPermissionErr := guild.CanUpdateBy(player)
	if guildPermissionErr != nil {
		return emptyResponse, guildPermissionErr
	}

	// A member can only set entry rank equal to or worse (numerically higher)
	// than their own. The ceiling is a member's own rank, so it only applies to a
	// member: a caller's rank in some other guild says nothing about this one,
	// and a guildless owner has no rank to be limited by.
	if player.GetGuildId() == guildId && msg.NewEntryRank < player.GetGuildRank() {
		return emptyResponse, types.NewPermissionError(
			"player", player.GetPlayerId(),
			"guild", guild.GetGuildId(),
			uint64(types.PermAdmin), "guild_update_entry_rank",
		)
	}

	if err := guild.SetEntryRank(msg.NewEntryRank); err != nil {
		return emptyResponse, err
	}

	cc.CommitAll()
	return &types.MsgGuildUpdateResponse{}, nil
}
