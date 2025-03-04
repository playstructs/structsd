package keeper

import (
	"context"

	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"

	// Used in Randomness Orb

	"fmt"
	"cosmossdk.io/math"
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
	K          *Keeper
	Ctx        context.Context

	AnyChange bool
	Ready bool

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
		K:          k,
		Ctx:        ctx,

		AnyChange: false,

		OwnerLoaded: false,

		GuildLoaded:  false,
		GuildChanged: false,

	}
}

func (cache *GuildCache) Commit() {
	cache.AnyChange = false

	fmt.Printf("\n Updating Guild From Cache (%s) \n", cache.GuildId)

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
func (cache *GuildCache) LoadGuild() (bool) {
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
func (cache *GuildCache) GetGuild() types.Guild { if !cache.GuildLoaded { cache.LoadGuild() }; return cache.Guild }
func (cache *GuildCache) GetGuildId() string { return cache.GuildId }

// Get the Owner data
func (cache *GuildCache) GetOwnerId() string { if !cache.GuildLoaded { cache.LoadGuild() }; return cache.Guild.Owner }
func (cache *GuildCache) GetOwner() *PlayerCache { if !cache.OwnerLoaded { cache.LoadOwner() }; return cache.Owner }

func (cache *GuildCache) GetEntrySubstationId() string { if !cache.GuildLoaded { cache.LoadGuild() }; return cache.Guild.EntrySubstationId }
func (cache *GuildCache) GetSubstation() *SubstationCache {if !cache.SubstationLoaded { cache.LoadSubstation() }; return cache.Substation }

func (cache *GuildCache) GetCreator() string { if !cache.GuildLoaded { cache.LoadGuild() }; return cache.Guild.Creator }

func (cache *GuildCache) GetBankCollateralPool() string { return types.GuildBankCollateralPool + cache.GetGuildId() }
func (cache *GuildCache) GetBankDenom() string { return cache.GetGuildId() }


/* Permissions */

// Delete Permission
func (cache *GuildCache) CanDelete(activePlayer *PlayerCache) (error) {
    return cache.PermissionCheck(types.PermissionDelete, activePlayer)
}

// Update Permission
func (cache *GuildCache) CanUpdate(activePlayer *PlayerCache) (error) {
    return cache.PermissionCheck(types.PermissionUpdate, activePlayer)
}

// Assets Permission
func (cache *GuildCache) CanAdministrateBank(activePlayer *PlayerCache) (error) {
    return cache.PermissionCheck(types.PermissionAssets, activePlayer)
}


func (cache *GuildCache) PermissionCheck(permission types.Permission, activePlayer *PlayerCache) (error) {
    // Make sure the address calling this has permissions
    if (!cache.K.PermissionHasOneOf(cache.Ctx, GetAddressPermissionIDBytes(activePlayer.GetActiveAddress()), permission)) {
        return sdkerrors.Wrapf(types.ErrPermission, "Calling address (%s) has no (%d) permissions ", activePlayer.GetActiveAddress(), permission)
    }

    if !activePlayer.HasPlayerAccount() {
        return sdkerrors.Wrapf(types.ErrPermission, "Calling address (%s) has no Account", activePlayer.GetActiveAddress())
    } else {
        if (activePlayer.GetPlayerId() != cache.GetOwnerId()) {
            if (!cache.K.PermissionHasOneOf(cache.Ctx, GetObjectPermissionIDBytes(cache.GetGuildId(), activePlayer.GetPlayerId()), permission)) {
               return sdkerrors.Wrapf(types.ErrPermission, "Calling account (%s) has no (%d) permissions on target guild (%s)", activePlayer.GetPlayerId(), permission, cache.GetGuildId())
            }
        }
    }
    return nil
}


/* Temporary Banking Infrastructure */

func (cache *GuildCache) BankMint(amountAlpha math.Int, amountToken math.Int, player *PlayerCache) (error) {

    alphaCollateralCoin := sdk.NewCoin("alpha", amountAlpha)
    alphaCollateralCoins := sdk.NewCoins(alphaCollateralCoin)


    guildTokenCoin := sdk.NewCoin(cache.GetBankDenom(), amountToken)
    guildTokenCoins := sdk.NewCoins(guildTokenCoin)

    // Try to Move Alpha From the Player to the Pool
    if !cache.K.bankKeeper.HasBalance(cache.Ctx, player.GetPrimaryAccount(), alphaCollateralCoin) {
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Player cannot afford the mint")
    }

    errSend := cache.K.bankKeeper.SendCoinsFromAccountToModule(cache.Ctx, player.GetPrimaryAccount(), cache.GetBankCollateralPool(), alphaCollateralCoins)
    if errSend != nil {
        return errSend
    }

    // Mint new Guild Token
    cache.K.bankKeeper.MintCoins(cache.Ctx, cache.GetBankCollateralPool(), guildTokenCoins)

    // Move the new Guild Token to Player
    cache.K.bankKeeper.SendCoinsFromModuleToAccount(cache.Ctx, cache.GetBankCollateralPool(), player.GetPrimaryAccount(), guildTokenCoins)

    return nil
}


func (cache *GuildCache) BankRedeem(amountToken math.Int, player *PlayerCache) (error) {

    alphaCollateralBalance := cache.K.bankKeeper.SpendableCoin(cache.Ctx, cache.K.accountKeeper.GetModuleAddress(cache.GetBankCollateralPool()), "alpha")
    guildTokenSupply := cache.K.bankKeeper.GetSupply(cache.Ctx, cache.GetBankDenom())

    guildTokenCoin := sdk.NewCoin(cache.GetBankDenom(), amountToken)
    guildTokenCoins := sdk.NewCoins(guildTokenCoin)

    // Try to Move Alpha From the Player to the Pool
    if !cache.K.bankKeeper.HasBalance(cache.Ctx, player.GetPrimaryAccount(), guildTokenCoin) {
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Player cannot afford the mint")
    }

    // alphaAmount = amountToken / guildTokenSupply.Amount
    amountTokenDec := math.LegacyNewDecFromInt(amountToken)
    guildTokenSupplyDec := math.LegacyNewDecFromInt(guildTokenSupply.Amount)
    alphaCollateralBalanceDec := math.LegacyNewDecFromInt(alphaCollateralBalance.Amount)

    alphaAmount := amountTokenDec.Quo(guildTokenSupplyDec).Mul(alphaCollateralBalanceDec).TruncateInt()

    sendError := cache.K.bankKeeper.SendCoinsFromAccountToModule(cache.Ctx, player.GetPrimaryAccount(), cache.GetBankCollateralPool(), guildTokenCoins)
    if (sendError != nil){
       return sendError
    }

    // Burn the Guild Token
    errBurn := cache.K.bankKeeper.BurnCoins(cache.Ctx, player.GetPrimaryAddress(), guildTokenCoins)
    if errBurn != nil {
        return errBurn
    }

    // Move the Alpha to Player
    alphaAmountCoin := sdk.NewCoin("alpha", alphaAmount)
    alphaAmountCoins := sdk.NewCoins(alphaAmountCoin)
    cache.K.bankKeeper.SendCoinsFromModuleToAccount(cache.Ctx, cache.GetBankCollateralPool(), player.GetPrimaryAccount(), alphaAmountCoins)

    return nil
}


func (cache *GuildCache) BankConfiscateAndBurn(amountToken math.Int, address string) (error) {

    guildTokenCoin := sdk.NewCoin(cache.GetBankDenom(), amountToken)
    guildTokenCoins := sdk.NewCoins(guildTokenCoin)

    // Confiscate
    account, _ := sdk.AccAddressFromBech32(address)
    sendError := cache.K.bankKeeper.SendCoinsFromAccountToModule(cache.Ctx, account, cache.GetBankCollateralPool(), guildTokenCoins)
    if (sendError != nil){
       return sendError
    }

    // Burn the Guild Token
    errBurn := cache.K.bankKeeper.BurnCoins(cache.Ctx, cache.GetBankCollateralPool(), guildTokenCoins)
    if errBurn != nil {
        return errBurn
    }

    return nil
}