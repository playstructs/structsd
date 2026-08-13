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

	callingPlayer, err := cc.GetSigningPlayer(msg.Creator)
	if err != nil {
		return emptyResponse, err
	}

	if msg.PlayerId == "" {
		return emptyResponse, types.NewParameterValidationError("player_id", 0, "required")
	}

	if msg.GuildRank == 0 {
		return emptyResponse, types.NewParameterValidationError("rank", 0, "zero")
	}

	targetPlayer, targetErr := cc.GetExistingPlayer(msg.PlayerId)
	if targetErr != nil {
		return emptyResponse, targetErr
	}

	guildId := msg.GuildId
	if guildId == "" {
		if callingPlayer.GetGuildId() == "" {
			return emptyResponse, types.NewGuildMembershipError("", callingPlayer.GetPlayerId(), "not_member")
		}
		guildId = callingPlayer.GetGuildId()
	}

	// The rule is that the target belongs to the named guild. For a member caller
	// naming nothing that is the same test as before; for an owner who is not a
	// member it is the only test that can be made.
	if targetPlayer.GetGuildId() != guildId {
		return emptyResponse, types.NewGuildMembershipError(guildId, targetPlayer.GetPlayerId(), "not_member")
	}

	guild := cc.GetGuild(guildId)
	if guild.CheckGuild() != nil {
		return emptyResponse, types.NewObjectNotFoundError("guild", guildId)
	}

	// Authorization: either PermAdmin on the guild (bypasses rank check)
	// or rank-based authority (actor rank must be strictly better than target's current rank,
	// and new rank must be >= actor's rank).
	permErr := cc.PermissionCheck(guild, callingPlayer, types.PermAdmin)
	if permErr != nil {
		// Rank authority is authority within a guild, so it belongs to members of
		// the one being edited. Anyone else has only PermAdmin to fall back on,
		// which they have already failed.
		if callingPlayer.GetGuildId() != guildId {
			return emptyResponse, permErr
		}

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
