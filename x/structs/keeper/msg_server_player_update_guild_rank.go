package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PlayerUpdateGuildRank(goCtx context.Context, msg *types.MsgPlayerUpdateGuildRank) (*types.MsgPlayerUpdateGuildRankResponse, error) {
	emptyResponse := &types.MsgPlayerUpdateGuildRankResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	k.AddressEmitActivity(ctx, msg.Creator)

	callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
	if err != nil {
		return emptyResponse, err
	}

	if msg.PlayerId == "" {
		return emptyResponse, types.NewParameterValidationError("player_id", 0, "required")
	}
	if len(msg.PlayerId) > maxIDLength {
		return emptyResponse, types.NewParameterValidationError("player_id", 0, "exceeds_max_length")
	}

	if msg.GuildRank == 0 {
		return emptyResponse, types.NewParameterValidationError("rank", 0, "zero")
	}

	targetPlayer, targetErr := cc.GetPlayer(msg.PlayerId)
	if targetErr != nil {
		return emptyResponse, targetErr
	}

	if callingPlayer.GetGuildId() == "" {
		return emptyResponse, types.NewGuildMembershipError("", callingPlayer.GetPlayerId(), "not_member")
	}
	if targetPlayer.GetGuildId() == "" || targetPlayer.GetGuildId() != callingPlayer.GetGuildId() {
		return emptyResponse, types.NewGuildMembershipError(callingPlayer.GetGuildId(), targetPlayer.GetPlayerId(), "not_member")
	}

	guild := cc.GetGuild(callingPlayer.GetGuildId())
	if guild.CheckGuild() != nil {
		return emptyResponse, types.NewObjectNotFoundError("guild", callingPlayer.GetGuildId())
	}

	// Authorization: either PermAdmin on the guild (bypasses rank check)
	// or rank-based authority (actor rank must be strictly better than target's current rank,
	// and new rank must be >= actor's rank).
	permErr := cc.PermissionCheck(guild, callingPlayer, types.PermAdmin)
	if permErr != nil {
		actorRank := callingPlayer.GetGuildRank()
		targetRank := targetPlayer.GetGuildRank()

		if actorRank >= targetRank {
			return emptyResponse, types.NewPermissionError("player", callingPlayer.GetPlayerId(), "guild", guild.GetGuildId(), uint64(types.PermAdmin), "player_update_guild_rank")
		}
		if msg.GuildRank < actorRank {
			return emptyResponse, types.NewPermissionError("player", callingPlayer.GetPlayerId(), "guild", guild.GetGuildId(), uint64(types.PermAdmin), "player_update_guild_rank")
		}
	}

	targetPlayer.SetGuildRank(msg.GuildRank)
	cc.CommitAll()
	return &types.MsgPlayerUpdateGuildRankResponse{}, nil
}
