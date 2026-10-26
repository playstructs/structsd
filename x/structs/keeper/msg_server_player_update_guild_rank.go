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

	// Rank purchases confer configured access, not authority to grant free ranks.
	if err := cc.PermissionAdministrationCheck(guild, callingPlayer); err != nil {
		return emptyResponse, err
	}

	targetPlayer.SetGuildRank(msg.GuildRank)
	cc.CommitAll()
	return &types.MsgPlayerUpdateGuildRankResponse{}, nil
}
