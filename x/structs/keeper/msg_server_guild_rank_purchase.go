package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

func (k msgServer) GuildRankPurchase(goCtx context.Context, msg *types.MsgGuildRankPurchase) (*types.MsgGuildRankPurchaseResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	player, err := cc.GetSigningPlayer(msg.Creator)
	if err != nil {
		return nil, err
	}
	guild, err := cc.GetExistingGuild(msg.GuildId)
	if err != nil {
		return nil, err
	}
	if player.GetGuildId() != guild.ID() {
		return nil, types.NewGuildMembershipError(guild.ID(), player.ID(), "not_member")
	}
	if err := cc.PermissionCheck(player, player, types.PermGuildMembership|types.PermTokenTransfer); err != nil {
		return nil, err
	}
	previous := player.GetGuildRank()
	if previous != msg.ExpectedRank {
		return nil, types.NewParameterValidationError("expected_rank", msg.ExpectedRank, "rank_changed")
	}
	price, err := guild.RankPurchasePrice(previous, msg.TargetRank, false)
	if err != nil {
		return nil, err
	}
	if err := cc.payRankPurchase(player, guild, price, msg.MaxPrice); err != nil {
		return nil, err
	}
	player.SetGuildRank(msg.TargetRank)
	if err := ctx.EventManager().EmitTypedEvent(&types.EventGuildRankPurchase{
		GuildId: guild.ID(), PlayerId: player.ID(), PreviousRank: previous,
		Rank: msg.TargetRank, Price: price, Denom: guild.GetBankDenom(),
	}); err != nil {
		return nil, err
	}
	k.AddressEmitActivity(ctx, msg.Creator)
	cc.CommitAll()
	return &types.MsgGuildRankPurchaseResponse{Price: price, Denom: guild.GetBankDenom(), Rank: msg.TargetRank}, nil
}
