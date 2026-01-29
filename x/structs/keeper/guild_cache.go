package keeper

import (
	"context"

	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	// Used in Randomness Orb

	"cosmossdk.io/math"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

/*

message Guild {
  string id                 = 1;
  uint64 index              = 2;

  string endpoint           = 3;

  string creator            = 4;
  string owner              = 5;

  uint64 joinInfusionMinimum = 6;
  guildJoinBypassLevel joinInfusionMinimumBypassByRequest   = 7 [(amino.dont_omitempty) = true];
  guildJoinBypassLevel joinInfusionMinimumBypassByInvite    = 8 [(amino.dont_omitempty) = true];

  string primaryReactorId    = 9;
  string entrySubstationId   = 10;
*/

type GuildCache struct {
	GuildId string
	K       *Keeper
	Ctx     context.Context

	AnyChange bool
	Ready     bool

	GuildLoaded  bool
	GuildChanged bool
	Guild        types.Guild

	OwnerLoaded bool
	Owner       *PlayerCache

	SubstationLoaded bool
	Substation       *SubstationCache
}

// Build this initial Guild Cache object
func (k *Keeper) GetGuildCacheFromId(ctx context.Context, guildId string) GuildCache {
	return GuildCache{
		GuildId: guildId,
		K:       k,
		Ctx:     ctx,

		AnyChange: false,

		OwnerLoaded: false,

		GuildLoaded:  false,
		GuildChanged: false,
	}
}

func (cache *GuildCache) Commit() {
	cache.AnyChange = false

	cache.K.logger.Info("Updating Guild From Cache", "guildId", cache.GuildId)

	if cache.GuildChanged {
		cache.K.SetGuild(cache.Ctx, cache.Guild)
		cache.GuildChanged = false
	}

	if cache.Substation != nil && cache.GetSubstation().IsChanged() {
		cache.GetSubstation().Commit()
	}

	if cache.Owner != nil && cache.GetOwner().IsChanged() {
		cache.GetOwner().Commit()
	}

}

func (cache *GuildCache) IsChanged() bool {
	return cache.AnyChange
}

func (cache *GuildCache) ID() string {
	return cache.GuildId
}

func (cache *GuildCache) Changed() {
	cache.AnyChange = true
}

/* Separate Loading functions for each of the underlying containers */

// Load the Player data
func (cache *GuildCache) LoadOwner() bool {
	newOwner, _ := cache.K.GetPlayerCacheFromId(cache.Ctx, cache.GetOwnerId())
	cache.Owner = &newOwner
	cache.OwnerLoaded = true
	return cache.OwnerLoaded
}

func (cache *GuildCache) ManualLoadOwner(owner *PlayerCache) {
	cache.Owner = owner
	cache.OwnerLoaded = true
}

// Load the Guild record
func (cache *GuildCache) LoadGuild() bool {
	guild, guildFound := cache.K.GetGuild(cache.Ctx, cache.GuildId)

	if guildFound {
		cache.Guild = guild
		cache.GuildLoaded = true
	}

	return cache.GuildLoaded
}

// Load the Substation data
func (cache *GuildCache) LoadSubstation() bool {
	newSubstation := cache.K.GetSubstationCacheFromId(cache.Ctx, cache.GetEntrySubstationId())
	cache.Substation = &newSubstation
	cache.SubstationLoaded = true
	return cache.SubstationLoaded
}

/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */
func (cache *GuildCache) GetGuild() types.Guild {
	if !cache.GuildLoaded {
		cache.LoadGuild()
	}
	return cache.Guild
}
func (cache *GuildCache) GetGuildId() string { return cache.GuildId }

// Get the Owner data
func (cache *GuildCache) GetOwnerId() string {
	if !cache.GuildLoaded {
		cache.LoadGuild()
	}
	return cache.Guild.Owner
}
func (cache *GuildCache) GetOwner() *PlayerCache {
	if !cache.OwnerLoaded {
		cache.LoadOwner()
	}
	return cache.Owner
}

func (cache *GuildCache) GetJoinInfusionMinimum() uint64 {
	if !cache.GuildLoaded {
		cache.LoadGuild()
	}
	return cache.Guild.JoinInfusionMinimum
}
func (cache *GuildCache) GetJoinInfusionMinimumBypassByInvite() types.GuildJoinBypassLevel {
	if !cache.GuildLoaded {
		cache.LoadGuild()
	}
	return cache.Guild.JoinInfusionMinimumBypassByInvite
}
func (cache *GuildCache) GetJoinInfusionMinimumBypassByRequest() types.GuildJoinBypassLevel {
	if !cache.GuildLoaded {
		cache.LoadGuild()
	}
	return cache.Guild.JoinInfusionMinimumBypassByRequest
}
func (cache *GuildCache) GetEntrySubstationId() string {
	if !cache.GuildLoaded {
		cache.LoadGuild()
	}
	return cache.Guild.EntrySubstationId
}
func (cache *GuildCache) GetSubstation() *SubstationCache {
	if !cache.SubstationLoaded {
		cache.LoadSubstation()
	}
	return cache.Substation
}
func (cache *GuildCache) GetPrimaryReactorId() string {
	if !cache.GuildLoaded {
		cache.LoadGuild()
	}
	return cache.Guild.PrimaryReactorId
}

func (cache *GuildCache) GetCreator() string {
	if !cache.GuildLoaded {
		cache.LoadGuild()
	}
	return cache.Guild.Creator
}

func (cache *GuildCache) GetBankCollateralPool() sdk.AccAddress {
	return authtypes.NewModuleAddress(types.GuildBankCollateralPool + cache.GetGuildId())
}
func (cache *GuildCache) GetBankDenom() string { return "uguild." + cache.GetGuildId() }

/* Permissions */

// Delete Permission
func (cache *GuildCache) CanDelete(activePlayer *PlayerCache) error {
	return cache.PermissionCheck(types.PermissionDelete, activePlayer)
}

// Update Permission
func (cache *GuildCache) CanUpdate(activePlayer *PlayerCache) error {
	return cache.PermissionCheck(types.PermissionUpdate, activePlayer)
}

// Assets Permission
func (cache *GuildCache) CanAdministrateBank(activePlayer *PlayerCache) error {
	return cache.PermissionCheck(types.PermissionAssets, activePlayer)
}

// Associations Permission
func (cache *GuildCache) CanAddMembersByProxy(activePlayer *PlayerCache) error {
	return cache.PermissionCheck(types.PermissionAssociations, activePlayer)
}

func (cache *GuildCache) CanInviteMembers(activePlayer *PlayerCache) (err error) {

	switch cache.GetJoinInfusionMinimumBypassByInvite() {
	// Invites are currently closed
	case types.GuildJoinBypassLevel_closed:
		err = types.NewGuildMembershipError(cache.GetGuildId(), activePlayer.GetPlayerId(), "not_allowed").WithJoinType("invite")

	// Only specific players can invite
	case types.GuildJoinBypassLevel_permissioned:
		err = cache.PermissionCheck(types.PermissionAssociations, activePlayer)

	// All Guild Members can Invite
	case types.GuildJoinBypassLevel_member:
		if activePlayer.GetGuildId() != cache.GetGuildId() {
			err = types.NewGuildMembershipError(cache.GetGuildId(), activePlayer.GetPlayerId(), "not_member")
		}
	}
	return
}

func (cache *GuildCache) CanApproveMembershipRequest(activePlayer *PlayerCache) (err error) {
	switch cache.GetJoinInfusionMinimumBypassByRequest() {
	// Invites are currently closed
	case types.GuildJoinBypassLevel_closed:
		err = types.NewGuildMembershipError(cache.GetGuildId(), activePlayer.GetPlayerId(), "not_allowed").WithJoinType("request")

	// Only specific players can request
	case types.GuildJoinBypassLevel_permissioned:
		err = cache.PermissionCheck(types.PermissionAssociations, activePlayer)

	// All Guild Members can Invite
	case types.GuildJoinBypassLevel_member:
		if activePlayer.GetGuildId() != cache.GetGuildId() {
			err = types.NewGuildMembershipError(cache.GetGuildId(), activePlayer.GetPlayerId(), "not_member")
		}
	}
	return
}

func (cache *GuildCache) CanKickMembers(activePlayer *PlayerCache) error {
	return cache.PermissionCheck(types.PermissionAssociations, activePlayer)
}

func (cache *GuildCache) CanRequestMembership() (err error) {
	switch cache.GetJoinInfusionMinimumBypassByRequest() {
	// Invites are currently closed
	case types.GuildJoinBypassLevel_closed:
		err = types.NewGuildMembershipError(cache.GetGuildId(), "", "not_allowed").WithJoinType("request")
	}
	return
}

func (cache *GuildCache) PermissionCheck(permission types.Permission, activePlayer *PlayerCache) error {
	// Make sure the address calling this has permissions
	if !cache.K.PermissionHasOneOf(cache.Ctx, GetAddressPermissionIDBytes(activePlayer.GetActiveAddress()), permission) {
		return types.NewPermissionError("address", activePlayer.GetActiveAddress(), "", "", uint64(permission), "guild_action")
	}

	if !activePlayer.HasPlayerAccount() {
		return types.NewPlayerRequiredError(activePlayer.GetActiveAddress(), "guild_action")
	} else {
		if activePlayer.GetPlayerId() != cache.GetOwnerId() {
			if !cache.K.PermissionHasOneOf(cache.Ctx, GetObjectPermissionIDBytes(cache.GetGuildId(), activePlayer.GetPlayerId()), permission) {
				return types.NewPermissionError("player", activePlayer.GetPlayerId(), "guild", cache.GetGuildId(), uint64(permission), "guild_action")
			}
		}
	}
	return nil
}

/* Temporary Banking Infrastructure */

func (cache *GuildCache) BankMint(amountAlpha math.Int, amountToken math.Int, player *PlayerCache) error {

	alphaCollateralCoin := sdk.NewCoin("ualpha", amountAlpha)
	alphaCollateralCoins := sdk.NewCoins(alphaCollateralCoin)

	guildTokenCoin := sdk.NewCoin(cache.GetBankDenom(), amountToken)
	guildTokenCoins := sdk.NewCoins(guildTokenCoin)

	// Try to Move Alpha From the Player to the Pool
	if !cache.K.bankKeeper.HasBalance(cache.Ctx, player.GetPrimaryAccount(), alphaCollateralCoin) {
		return types.NewPlayerAffordabilityError(player.GetPlayerId(), "mint", amountAlpha.String()+" ualpha")
	}

	errSend := cache.K.bankKeeper.SendCoins(cache.Ctx, player.GetPrimaryAccount(), cache.GetBankCollateralPool(), alphaCollateralCoins)
	if errSend != nil {
		return errSend
	}

	// Mint new Guild Token
	cache.K.bankKeeper.MintCoins(cache.Ctx, types.ModuleName, guildTokenCoins)

	// Move the new Guild Token to Player
	cache.K.bankKeeper.SendCoinsFromModuleToAccount(cache.Ctx, types.ModuleName, player.GetPrimaryAccount(), guildTokenCoins)

	ctxSDK := sdk.UnwrapSDKContext(cache.Ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildBankMint{&types.EventGuildBankMintDetail{GuildId: cache.GetGuildId(), AmountAlpha: amountAlpha.Uint64(), AmountToken: amountToken.Uint64(), PlayerId: player.GetPlayerId()}})

	return nil
}

func (cache *GuildCache) BankRedeem(amountToken math.Int, player *PlayerCache) error {

	alphaCollateralBalance := cache.K.bankKeeper.SpendableCoin(cache.Ctx, cache.GetBankCollateralPool(), "ualpha")
	guildTokenSupply := cache.K.bankKeeper.GetSupply(cache.Ctx, cache.GetBankDenom())

	guildTokenCoin := sdk.NewCoin(cache.GetBankDenom(), amountToken)
	guildTokenCoins := sdk.NewCoins(guildTokenCoin)

	// Try to Move Alpha From the Player to the Pool
	if !cache.K.bankKeeper.HasBalance(cache.Ctx, player.GetPrimaryAccount(), guildTokenCoin) {
		return types.NewPlayerAffordabilityError(player.GetPlayerId(), "redeem", amountToken.String()+" "+cache.GetBankDenom())
	}

	// alphaAmount = amountToken / guildTokenSupply.Amount
	amountTokenDec := math.LegacyNewDecFromInt(amountToken)
	guildTokenSupplyDec := math.LegacyNewDecFromInt(guildTokenSupply.Amount)
	alphaCollateralBalanceDec := math.LegacyNewDecFromInt(alphaCollateralBalance.Amount)

	alphaAmount := amountTokenDec.Quo(guildTokenSupplyDec).Mul(alphaCollateralBalanceDec).TruncateInt()

	// Move the new coins back to the module
	cache.K.bankKeeper.SendCoinsFromAccountToModule(cache.Ctx, player.GetPrimaryAccount(), types.ModuleName, guildTokenCoins)
	// Burn the Guild Token
	errBurn := cache.K.bankKeeper.BurnCoins(cache.Ctx, types.ModuleName, guildTokenCoins)
	if errBurn != nil {
		return errBurn
	}

	// Move the Alpha to Player
	alphaAmountCoin := sdk.NewCoin("ualpha", alphaAmount)
	alphaAmountCoins := sdk.NewCoins(alphaAmountCoin)
	cache.K.bankKeeper.SendCoins(cache.Ctx, cache.GetBankCollateralPool(), player.GetPrimaryAccount(), alphaAmountCoins)

	ctxSDK := sdk.UnwrapSDKContext(cache.Ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildBankRedeem{&types.EventGuildBankRedeemDetail{GuildId: cache.GetGuildId(), AmountAlpha: alphaAmount.Uint64(), AmountToken: amountToken.Uint64(), PlayerId: player.GetPlayerId()}})

	return nil
}

func (cache *GuildCache) BankConfiscateAndBurn(amountToken math.Int, address string) error {

	guildTokenCoin := sdk.NewCoin(cache.GetBankDenom(), amountToken)
	guildTokenCoins := sdk.NewCoins(guildTokenCoin)

	// Confiscate
	playerAcc, _ := sdk.AccAddressFromBech32(address)
	cache.K.bankKeeper.SendCoinsFromAccountToModule(cache.Ctx, playerAcc, types.ModuleName, guildTokenCoins)

	// Burn the Guild Token
	errBurn := cache.K.bankKeeper.BurnCoins(cache.Ctx, types.ModuleName, guildTokenCoins)
	if errBurn != nil {
		return errBurn
	}

	ctxSDK := sdk.UnwrapSDKContext(cache.Ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildBankConfiscateAndBurn{&types.EventGuildBankConfiscateAndBurnDetail{GuildId: cache.GetGuildId(), AmountToken: amountToken.Uint64(), Address: address}})

	return nil
}
