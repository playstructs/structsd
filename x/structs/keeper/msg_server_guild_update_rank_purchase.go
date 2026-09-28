package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

func (k msgServer) GuildUpdateRankPurchase(goCtx context.Context, msg *types.MsgGuildUpdateRankPurchase) (*types.MsgGuildUpdateResponse, error) {
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
	if err := cc.PermissionCheck(guild, player, types.PermAdmin); err != nil {
		return nil, err
	}
	if err := guild.SetRankPurchase(msg.BasePrice, msg.StepIncrease, msg.MinimumRank, msg.JoinByRankPurchaseEnabled); err != nil {
		return nil, err
	}
	k.AddressEmitActivity(ctx, msg.Creator)
	cc.CommitAll()
	return &types.MsgGuildUpdateResponse{}, nil
}
