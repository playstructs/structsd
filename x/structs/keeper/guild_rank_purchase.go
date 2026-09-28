package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

func (cc *CurrentContext) GetExistingGuild(id string) (*GuildCache, error) {
	kind, _, err := types.ParseAssetOwnerID(id)
	if err != nil || kind != types.ObjectType_guild {
		return nil, types.NewObjectNotFoundError("guild", id)
	}
	guild := cc.GetGuild(id)
	if err := guild.CheckGuild(); err != nil {
		return nil, err
	}
	return guild, nil
}

func (cache *GuildCache) SetRankPurchase(base, increase, minimum uint64, joinEnabled bool) error {
	if err := cache.CheckGuild(); err != nil {
		return err
	}
	if err := types.ValidateRankPurchaseSettings(cache.GetEntryRank(), minimum, joinEnabled); err != nil {
		return err
	}
	cache.Guild.RankPurchaseBasePrice = base
	cache.Guild.RankPurchaseStepIncrease = increase
	cache.Guild.RankPurchaseMinimumRank = minimum
	cache.Guild.JoinByRankPurchaseEnabled = joinEnabled
	cache.Changed = true
	return nil
}

// RankPurchasePrice does not authorize a player or move coins. Both transaction
// paths use it so admission cannot acquire a cheaper interpretation of the curve.
func (cache *GuildCache) RankPurchasePrice(current, target uint64, joining bool) (uint64, error) {
	if err := cache.CheckGuild(); err != nil {
		return 0, err
	}
	guild := cache.GetGuild()
	if guild.RankPurchaseMinimumRank == 0 {
		return 0, types.NewGuildMembershipError(guild.Id, "", "rank_purchase_disabled")
	}
	if target < guild.RankPurchaseMinimumRank || target > guild.EntryRank {
		return 0, types.NewParameterValidationError("target_rank", target, "outside_purchase_range")
	}
	if joining {
		if !guild.JoinByRankPurchaseEnabled {
			return 0, types.NewGuildMembershipError(guild.Id, "", "rank_purchase_admission_disabled")
		}
		current = guild.EntryRank
		if target == current {
			return 0, nil
		}
	}
	return types.CalculateGuildRankPurchasePrice(guild.EntryRank, current, target,
		guild.RankPurchaseBasePrice, guild.RankPurchaseStepIncrease)
}

// payRankPurchase is called only after all membership and rank checks succeed.
// The signing key must authorize spending even for a free configured rank.
func (cc *CurrentContext) payRankPurchase(player *PlayerCache, guild *GuildCache, price, maxPrice uint64) error {
	if price > maxPrice {
		return types.NewParameterValidationError("max_price", maxPrice, "quoted_price_exceeded")
	}
	vault, err := cc.GetVault(player.ID())
	if err != nil {
		return err
	}
	if err := vault.CanUseBy(player, types.PermTokenTransfer); err != nil {
		return err
	}
	if price == 0 {
		return nil
	}
	coins := sdk.NewCoins(sdk.NewCoin(guild.GetBankDenom(), math.NewIntFromUint64(price)))
	if err := cc.k.requireSendableAmount(cc.ctx, coins); err != nil {
		return err
	}
	if !cc.k.bankKeeper.HasBalance(cc.ctx, vault.Address(), coins[0]) {
		return types.NewPlayerAffordabilityError(player.ID(), "rank_purchase", coins.String())
	}
	addresses, err := cc.EnsureVaults(player.ID(), guild.ID())
	if err != nil {
		return err
	}
	return cc.k.bankKeeper.SendCoins(cc.ctx, addresses[0], addresses[1], coins)
}
