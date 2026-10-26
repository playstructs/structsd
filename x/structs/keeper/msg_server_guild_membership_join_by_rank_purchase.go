package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

func (k msgServer) GuildMembershipJoinByRankPurchase(goCtx context.Context, msg *types.MsgGuildMembershipJoinByRankPurchase) (*types.MsgGuildRankPurchaseResponse, error) {
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
	if err := cc.PermissionCheck(player, player, types.PermGuildMembership|types.PermTokenTransfer); err != nil {
		return nil, err
	}
	if player.GetGuildId() == guild.ID() {
		return nil, types.NewGuildMembershipError(guild.ID(), player.ID(), "already_member")
	}
	if player.GetGuildId() != msg.ExpectedGuildId {
		return nil, types.NewGuildMembershipError(guild.ID(), player.ID(), "membership_changed")
	}
	if _, err := guildCharterLeavable(cc, player); err != nil {
		return nil, err
	}
	price, err := guild.RankPurchasePrice(guild.GetEntryRank(), msg.TargetRank, true)
	if err != nil {
		return nil, err
	}
	// Joining uses the ordinary entry substation, not rank-specific benefits.
	// Validate both sides before payment because MigrateSubstation mutates indices.
	for _, id := range []string{player.GetSubstationId(), guild.GetEntrySubstationId()} {
		if id != "" {
			if _, err := cc.GetExistingSubstation(id); err != nil {
				return nil, err
			}
		}
	}
	if id := guild.GetEntrySubstationId(); id != "" {
		if err := cc.GetSubstation(id).CheckAcceptingConnections(); err != nil { return nil, err }
	}
	if err := cc.payRankPurchase(player, guild, price, msg.MaxPrice); err != nil {
		return nil, err
	}
	previous := player.GetGuildRank()
	player.MigrateGuild(guild)
	player.MigrateSubstation(guild.GetEntrySubstationId())
	player.SetGuildRank(msg.TargetRank)
	// A pending invitation/request is no longer actionable after independent
	// purchase admission. Consume it without inventing evidence of prior consent.
	application := cc.GetGuildMembershipApp(guild.ID(), player.ID())
	if application.LoadGuildMembershipApplication() {
		application.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_approved
		application.Changed = true
	}
	if err := ctx.EventManager().EmitTypedEvent(&types.EventGuildRankPurchase{
		GuildId: guild.ID(), PlayerId: player.ID(), PreviousRank: previous,
		Rank: msg.TargetRank, Price: price, Denom: guild.GetBankDenom(), Joined: true,
	}); err != nil {
		return nil, err
	}
	k.AddressEmitActivity(ctx, msg.Creator)
	cc.CommitAll()
	return &types.MsgGuildRankPurchaseResponse{Price: price, Denom: guild.GetBankDenom(), Rank: msg.TargetRank}, nil
}
