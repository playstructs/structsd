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

	player, err := cc.GetPlayerByAddress(msg.Creator)
	if err != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "guild_update_entry_rank")
	}

	if player.GetGuildId() == "" {
		return emptyResponse, types.NewGuildMembershipError("", player.GetPlayerId(), "not_member")
	}

	guild := cc.GetGuild(player.GetGuildId())
	if guild.CheckGuild() != nil {
		return emptyResponse, types.NewObjectNotFoundError("guild", player.GetGuildId())
	}

    guildPermissionErr := guild.CanUpdateBy(player)
    if guildPermissionErr != nil {
        return emptyResponse, guildPermissionErr
    }

	// Player can only set entry rank equal to or worse (numerically higher) than their own
	if msg.NewEntryRank < player.GetGuildRank() {
		return emptyResponse, types.NewPermissionError(
			"player", player.GetPlayerId(),
			"guild", guild.GetGuildId(),
			uint64(types.PermAdmin), "guild_update_entry_rank",
		)
	}

	guild.SetEntryRank(msg.NewEntryRank)

	cc.CommitAll()
	return &types.MsgGuildUpdateResponse{}, nil
}
