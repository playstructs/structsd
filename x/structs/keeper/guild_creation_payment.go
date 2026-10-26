package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

const (
	GuildCreationCostAlpha       = int64(2000000)
	GuildCreationBurnAlpha       = int64(1000000)
	GuildCreationCollateralAlpha = int64(1000000)
	GuildCreationInitialTokens   = int64(1000000)
)

// FundCreation settles a newly created guild's fixed founding cost. The caller
// must use the SDK transaction cache: bank failures after the first transfer
// require rollback of the entire creation, including the new guild record.
// This does not authorize or perform membership changes or reactor binding.
func (cache *GuildCache) FundCreation(founder *PlayerCache) error {
	if err := cache.CheckGuild(); err != nil {
		return err
	}
	if founder == nil || cache.GetOwnerId() != founder.ID() {
		return types.NewParameterValidationError("founder", 0, "must_own_new_guild")
	}
	signer, signerErr := cache.CC.GetPlayerByAddress(cache.CC.SignerAddress())
	if signerErr != nil || signer.ID() != founder.ID() {
		return types.NewParameterValidationError("founder", 0, "must_be_signing_player")
	}
	source, err := cache.CC.GetVault(founder.ID())
	if err != nil {
		return err
	}
	if err := source.CanUseBy(founder, types.PermTokenTransfer); err != nil {
		return err
	}
	destination, err := cache.CC.GetVault(cache.ID())
	if err != nil {
		return err
	}
	bank := cache.CC.k.bankKeeper
	ctx := cache.CC.ctx
	// A predictable pool can receive donations before founding. They increase
	// backing; they must never reserve the next guild ID or prevent creation.
	if !bank.GetSupply(ctx, cache.GetBankDenom()).IsZero() {
		return types.NewParameterValidationError("guild_bank", 0, "already_initialized")
	}
	if bank.SpendableCoin(ctx, source.Address(), "ualpha").Amount.LT(sdk.NewInt64Coin("ualpha", GuildCreationCostAlpha).Amount) {
		return types.NewPlayerAffordabilityError(founder.ID(), "guild_create", "2000000ualpha")
	}
	cost := sdk.NewCoins(sdk.NewInt64Coin("ualpha", GuildCreationCostAlpha))
	if err := cache.CC.k.requireSendableAmount(ctx, cost); err != nil {
		return err
	}
	if _, err := cache.CC.EnsureVaults(founder.ID(), cache.ID()); err != nil {
		return err
	}
	burn := sdk.NewCoins(sdk.NewInt64Coin("ualpha", GuildCreationBurnAlpha))
	if err := bank.SendCoinsFromAccountToModule(ctx, source.Address(), types.ModuleName, burn); err != nil {
		return err
	}
	if err := bank.BurnCoins(ctx, types.ModuleName, burn); err != nil {
		return err
	}
	collateral := sdk.NewCoins(sdk.NewInt64Coin("ualpha", GuildCreationCollateralAlpha))
	if err := bank.SendCoins(ctx, source.Address(), cache.GetBankCollateralPool(), collateral); err != nil {
		return err
	}
	initial := sdk.NewCoins(sdk.NewInt64Coin(cache.GetBankDenom(), GuildCreationInitialTokens))
	if err := bank.MintCoins(ctx, types.ModuleName, initial); err != nil {
		return err
	}
	return bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, destination.Address(), initial)
}
