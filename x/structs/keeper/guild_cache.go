package keeper

import (

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
	CC      *CurrentContext

	Changed bool
	Ready     bool

	GuildLoaded  bool
	Guild        types.Guild

}


func (cache *GuildCache) Commit() {
	if cache.Changed {
    	cache.CC.k.logger.Info("Updating Guild From Cache", "guildId", cache.GuildId)
		cache.CC.k.SetGuild(cache.CC.ctx, cache.Guild)
	}
	cache.Changed = false
}

func (cache *GuildCache) IsChanged() bool {
	return cache.Changed
}

func (cache *GuildCache) ID() string {
	return cache.GuildId
}

/* Separate Loading functions for each of the underlying containers */

// Load the Guild record
func (cache *GuildCache) LoadGuild() bool {
	cache.Guild, cache.GuildLoaded = cache.CC.k.GetGuild(cache.CC.ctx, cache.GuildId)

	// Guild records written before v0.21.0 have no bank fee fields, so the
	// non-nullable LegacyDec members decode as nil. Any arithmetic on them
	// panics, and (worse) re-marshaling a nil non-nullable LegacyDec panics
	// too, which would halt on the next SetGuild. Normalize to zero on load so
	// every downstream read/commit path is safe regardless of migration order.
	if cache.GuildLoaded {
		if cache.Guild.BankConvertInFee.IsNil() {
			cache.Guild.BankConvertInFee = math.LegacyZeroDec()
		}
		if cache.Guild.BankConvertOutFee.IsNil() {
			cache.Guild.BankConvertOutFee = math.LegacyZeroDec()
		}
	}

	return cache.GuildLoaded
}

/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */
func (cache *GuildCache) CheckGuild() (error) {
    if (!cache.GuildLoaded) {
        if !cache.LoadGuild() {
            return types.NewObjectNotFoundError("guild", cache.GuildId)
        }
    }
    return nil
}


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
    player, _ := cache.CC.GetPlayer(cache.GetOwnerId())
	return player
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
func (cache *GuildCache) GetSubstation() (substation *SubstationCache) {
    if cache.GetEntrySubstationId() != "" {
        substation = cache.CC.GetSubstation(cache.GetEntrySubstationId())
    }
	return
}

func (cache *GuildCache) GetEntryRank() uint64 {
	if !cache.GuildLoaded {
		cache.LoadGuild()
	}
	return cache.Guild.EntryRank
}

func (cache *GuildCache) GetPrimaryReactorId() string {
	if !cache.GuildLoaded {
		cache.LoadGuild()
	}
	return cache.Guild.PrimaryReactorId
}

func (cache *GuildCache) GetBankConvertInFee() math.LegacyDec {
	if !cache.GuildLoaded {
		cache.LoadGuild()
	}
	if cache.Guild.BankConvertInFee.IsNil() {
		return math.LegacyZeroDec()
	}
	return cache.Guild.BankConvertInFee
}

func (cache *GuildCache) GetBankConvertOutFee() math.LegacyDec {
	if !cache.GuildLoaded {
		cache.LoadGuild()
	}
	if cache.Guild.BankConvertOutFee.IsNil() {
		return math.LegacyZeroDec()
	}
	return cache.Guild.BankConvertOutFee
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

/*
    PermGuildAll = PermAdmin | PermUpdate | PermDelete | PermGuildMembership |
                    PermGuildEndpointUpdate | PermGuildJoinConstraintsUpdate | PermGuildSubstationUpdate |
                    PermGuildTokenBurn | PermGuildTokenMint
*/

// Delete Permission
func (cache *GuildCache) CanDeleteBy(activePlayer *PlayerCache) error {
	return cache.CC.PermissionCheck(cache, activePlayer, types.PermDelete)
}

// Update Permission
func (cache *GuildCache) CanUpdateBy(activePlayer *PlayerCache) error {
	return cache.CC.PermissionCheck(cache, activePlayer, types.PermUpdate)
}

// Update Permission
func (cache *GuildCache) CanTransferOwnershipBy(activePlayer *PlayerCache) error {
	return cache.CC.PermissionCheck(cache, activePlayer, types.PermAdmin)
}

// CanUpdatePrimaryReactorBy gates MsgGuildUpdatePrimaryReactor. The primary
// reactor is structurally bound to the guild's economic surface (token mints,
// guild bank, member infusion routing) so we deliberately reuse PermAdmin
// rather than minting a new permission bit (which would force a permission-
// register migration). Practically this means only the guild owner (and any
// player explicitly granted PermAdmin on the guild object) can rotate it.
func (cache *GuildCache) CanUpdatePrimaryReactorBy(activePlayer *PlayerCache) error {
	return cache.CC.PermissionCheck(cache, activePlayer, types.PermAdmin)
}

func (cache *GuildCache) CanUpdateEndpointBy(activePlayer *PlayerCache) error {
	return cache.CC.PermissionCheck(cache, activePlayer, types.PermGuildEndpointUpdate)
}

// CanUpdateBankFeesBy gates MsgGuildUpdateBankConvertInFee and
// MsgGuildUpdateBankConvertOutFee. Bank fees are a treasury lever with the same
// blast radius as token mint/burn (a 100% out-fee freezes every holder's exit),
// so we deliberately reuse PermAdmin rather than minting a new permission bit
// (which would force a permission-register migration), matching the
// CanUpdatePrimaryReactorBy precedent. Only the guild owner (and any player
// explicitly granted PermAdmin on the guild object) can change fees.
func (cache *GuildCache) CanUpdateBankFeesBy(activePlayer *PlayerCache) error {
	return cache.CC.PermissionCheck(cache, activePlayer, types.PermAdmin)
}

func (cache *GuildCache) CanUpdateJoinConstraintsBy(activePlayer *PlayerCache) error {
	return cache.CC.PermissionCheck(cache, activePlayer, types.PermGuildJoinConstraintsUpdate)
}

func (cache *GuildCache) CanUpdateSubstationBy(activePlayer *PlayerCache) error {
	return cache.CC.PermissionCheck(cache, activePlayer, types.PermGuildSubstationUpdate)
}

func (cache *GuildCache) CanBurnTokenBy(activePlayer *PlayerCache) error {
	return cache.CC.PermissionCheck(cache, activePlayer, types.PermGuildTokenBurn)
}

func (cache *GuildCache) CanMintTokenBy(activePlayer *PlayerCache) error {
	return cache.CC.PermissionCheck(cache, activePlayer, types.PermGuildTokenMint)
}

func (cache *GuildCache) CanAllocateAsSourceBy(activePlayer *PlayerCache) error {
    return types.NewAllocationError(cache.ID(), "unacceptable_source")
}

// Associations Permission
func (cache *GuildCache) CanAddMembersByProxy(activePlayer *PlayerCache) error {
	return cache.CC.PermissionCheck(cache, activePlayer, types.PermGuildMembership)
}

func (cache *GuildCache) CanInviteMembers(activePlayer *PlayerCache) (err error) {

	switch cache.GetJoinInfusionMinimumBypassByInvite() {
	// Invites are currently closed
	case types.GuildJoinBypassLevel_closed:
		err = types.NewGuildMembershipError(cache.GetGuildId(), activePlayer.GetPlayerId(), "not_allowed").WithJoinType("invite")

	// Only specific players can invite
	case types.GuildJoinBypassLevel_permissioned:
		err = cache.CC.PermissionCheck(cache, activePlayer, types.PermGuildMembership)

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
            err = cache.CC.PermissionCheck(cache, activePlayer, types.PermGuildMembership)

        // All Guild Members can Invite
        case types.GuildJoinBypassLevel_member:
            if activePlayer.GetGuildId() != cache.GetGuildId() {
                err = types.NewGuildMembershipError(cache.GetGuildId(), activePlayer.GetPlayerId(), "not_member")
            }
        }
	return
}

func (cache *GuildCache) CanKickMembers(activePlayer *PlayerCache) error {
	return cache.CC.PermissionCheck(cache, activePlayer, types.PermGuildMembership)
}

func (cache *GuildCache) CanRequestMembership() (err error) {
	switch cache.GetJoinInfusionMinimumBypassByRequest() {
	// Invites are currently closed
	case types.GuildJoinBypassLevel_closed:
		err = types.NewGuildMembershipError(cache.GetGuildId(), "", "not_allowed").WithJoinType("request")
	}
	return
}


/* Temporary Banking Infrastructure */

func (cache *GuildCache) BankMint(amountAlpha math.Int, amountToken math.Int, player *PlayerCache) error {

	alphaCollateralCoin := sdk.NewCoin("ualpha", amountAlpha)
	alphaCollateralCoins := sdk.NewCoins(alphaCollateralCoin)

	guildTokenCoin := sdk.NewCoin(cache.GetBankDenom(), amountToken)
	guildTokenCoins := sdk.NewCoins(guildTokenCoin)

	// Try to Move Alpha From the Player to the Pool
	if !cache.CC.k.bankKeeper.HasBalance(cache.CC.ctx, player.GetPrimaryAccount(), alphaCollateralCoin) {
		return types.NewPlayerAffordabilityError(player.GetPlayerId(), "mint", amountAlpha.String()+" ualpha")
	}

	errSend := cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, player.GetPrimaryAccount(), cache.GetBankCollateralPool(), alphaCollateralCoins)
	if errSend != nil {
		return errSend
	}

	// Mint new Guild Token
	errMint := cache.CC.k.bankKeeper.MintCoins(cache.CC.ctx, types.ModuleName, guildTokenCoins)
	if errMint != nil {
		return errMint
	}

	// Move the new Guild Token to Player
	errTransfer := cache.CC.k.bankKeeper.SendCoinsFromModuleToAccount(cache.CC.ctx, types.ModuleName, player.GetPrimaryAccount(), guildTokenCoins)
	if errTransfer != nil {
		return errTransfer
	}

	ctxSDK := sdk.UnwrapSDKContext(cache.CC.ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildBankMint{&types.EventGuildBankMintDetail{GuildId: cache.GetGuildId(), AmountAlpha: amountAlpha.Uint64(), AmountToken: amountToken.Uint64(), PlayerId: player.GetPlayerId()}})

	return nil
}

// BankConvert converts ualpha into this guild's token at the current collateral
// ratio. It is ratio-preserving: the full amountAlpha (including the convert-in
// fee) is added to collateral, and tokens are minted against the net alpha, so
// the alpha-backing-per-token for existing holders never decreases.
//
// Math is pure math.Int (multiply THEN floor-divide) to avoid the amplified
// rounding of LegacyDec.Quo; the fee rate is the only LegacyDec, applied with
// the exact MulInt(...).Ceil() form so it always rounds in the pool's favor.
//
// Requires supply > 0 AND collateral > 0: with either at zero the ratio is
// undefined and (collateral == 0) an admin could mint unbounded tokens per
// alpha. Bootstrapping a fresh bank stays on the admin BankMint path.
//
// Returns the amount of guild token minted to the player.
func (cache *GuildCache) BankConvert(amountAlpha math.Int, minAmountToken math.Int, player *PlayerCache) (math.Int, error) {

	if !amountAlpha.IsPositive() {
		return math.ZeroInt(), types.NewParameterValidationError("amountAlpha", 0, "must_be_positive")
	}

	alphaCoin := sdk.NewCoin("ualpha", amountAlpha)
	alphaCoins := sdk.NewCoins(alphaCoin)

	if !cache.CC.k.bankKeeper.HasBalance(cache.CC.ctx, player.GetPrimaryAccount(), alphaCoin) {
		return math.ZeroInt(), types.NewPlayerAffordabilityError(player.GetPlayerId(), "convert", amountAlpha.String()+" ualpha")
	}

	// Snapshot the pool BEFORE moving any alpha in.
	collateral := cache.CC.k.bankKeeper.SpendableCoin(cache.CC.ctx, cache.GetBankCollateralPool(), "ualpha").Amount
	supply := cache.CC.k.bankKeeper.GetSupply(cache.CC.ctx, cache.GetBankDenom()).Amount

	if !supply.IsPositive() || !collateral.IsPositive() {
		return math.ZeroInt(), types.NewParameterValidationError("bank_ratio", 0, "undefined")
	}

	// Fee rounds up (guild-favored); tokensOut = floor(netAlpha * supply / collateral).
	feeAlpha := cache.GetBankConvertInFee().MulInt(amountAlpha).Ceil().TruncateInt()
	netAlpha := amountAlpha.Sub(feeAlpha)
	tokensOut := netAlpha.Mul(supply).Quo(collateral)

	if !tokensOut.IsPositive() {
		return math.ZeroInt(), types.NewParameterValidationError("amountToken", 0, "conversion_too_small")
	}
	if tokensOut.LT(minAmountToken) {
		return math.ZeroInt(), types.NewParameterValidationError("minAmountToken", minAmountToken.Uint64(), "slippage")
	}

	guildTokenCoins := sdk.NewCoins(sdk.NewCoin(cache.GetBankDenom(), tokensOut))

	// Full alpha (fee included) into collateral.
	errSend := cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, player.GetPrimaryAccount(), cache.GetBankCollateralPool(), alphaCoins)
	if errSend != nil {
		return math.ZeroInt(), errSend
	}

	errMint := cache.CC.k.bankKeeper.MintCoins(cache.CC.ctx, types.ModuleName, guildTokenCoins)
	if errMint != nil {
		return math.ZeroInt(), errMint
	}

	errTransfer := cache.CC.k.bankKeeper.SendCoinsFromModuleToAccount(cache.CC.ctx, types.ModuleName, player.GetPrimaryAccount(), guildTokenCoins)
	if errTransfer != nil {
		return math.ZeroInt(), errTransfer
	}

	ctxSDK := sdk.UnwrapSDKContext(cache.CC.ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildBankConvert{&types.EventGuildBankConvertDetail{GuildId: cache.GetGuildId(), AmountAlpha: amountAlpha.Uint64(), Fee: feeAlpha.Uint64(), AmountToken: tokensOut.Uint64(), PlayerId: player.GetPlayerId()}})

	return tokensOut, nil
}

// BankRedeem burns amountToken and returns the pro-rata share of collateral,
// less the guild's convert-out fee (which stays in collateral). Returns the net
// alpha paid out to the player. minAmountAlpha guards against ratio movement
// (0 = no guard).
//
// The pro-rata share is computed in pure math.Int (grossAlpha = floor(amountToken
// * collateral / supply)). This intentionally differs from the pre-v0.21.0
// divide-first LegacyDec formula, whose banker's rounding could overpay the
// redeemer at large collateral; the Int form always rounds in the pool's favor.
func (cache *GuildCache) BankRedeem(amountToken math.Int, minAmountAlpha math.Int, player *PlayerCache) (math.Int, error) {

	if !amountToken.IsPositive() {
		return math.ZeroInt(), types.NewParameterValidationError("amountToken", 0, "must_be_positive")
	}

	alphaCollateralBalance := cache.CC.k.bankKeeper.SpendableCoin(cache.CC.ctx, cache.GetBankCollateralPool(), "ualpha")
	guildTokenSupply := cache.CC.k.bankKeeper.GetSupply(cache.CC.ctx, cache.GetBankDenom())

	guildTokenCoin := sdk.NewCoin(cache.GetBankDenom(), amountToken)
	guildTokenCoins := sdk.NewCoins(guildTokenCoin)

	if !cache.CC.k.bankKeeper.HasBalance(cache.CC.ctx, player.GetPrimaryAccount(), guildTokenCoin) {
		return math.ZeroInt(), types.NewPlayerAffordabilityError(player.GetPlayerId(), "redeem", amountToken.String()+" "+cache.GetBankDenom())
	}

	if !guildTokenSupply.Amount.IsPositive() {
		return math.ZeroInt(), types.NewParameterValidationError("bank_ratio", 0, "undefined")
	}

	// grossAlpha = floor(amountToken * collateral / supply); fee rounds up.
	grossAlpha := amountToken.Mul(alphaCollateralBalance.Amount).Quo(guildTokenSupply.Amount)
	feeAlpha := cache.GetBankConvertOutFee().MulInt(grossAlpha).Ceil().TruncateInt()
	netAlpha := grossAlpha.Sub(feeAlpha)

	if netAlpha.LT(minAmountAlpha) {
		return math.ZeroInt(), types.NewParameterValidationError("minAmountAlpha", minAmountAlpha.Uint64(), "slippage")
	}

	// Move the tokens back to the module and burn them.
	errReturn := cache.CC.k.bankKeeper.SendCoinsFromAccountToModule(cache.CC.ctx, player.GetPrimaryAccount(), types.ModuleName, guildTokenCoins)
	if errReturn != nil {
		return math.ZeroInt(), errReturn
	}
	errBurn := cache.CC.k.bankKeeper.BurnCoins(cache.CC.ctx, types.ModuleName, guildTokenCoins)
	if errBurn != nil {
		return math.ZeroInt(), errBurn
	}

	// Move the net alpha to the player; the fee remains in collateral.
	if netAlpha.IsPositive() {
		alphaAmountCoins := sdk.NewCoins(sdk.NewCoin("ualpha", netAlpha))
		errAlpha := cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, cache.GetBankCollateralPool(), player.GetPrimaryAccount(), alphaAmountCoins)
		if errAlpha != nil {
			return math.ZeroInt(), errAlpha
		}
	}

	ctxSDK := sdk.UnwrapSDKContext(cache.CC.ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildBankRedeem{&types.EventGuildBankRedeemDetail{GuildId: cache.GetGuildId(), AmountAlpha: netAlpha.Uint64(), AmountToken: amountToken.Uint64(), PlayerId: player.GetPlayerId(), Fee: feeAlpha.Uint64()}})

	return netAlpha, nil
}

func (cache *GuildCache) BankConfiscateAndBurn(amountToken math.Int, address string) error {

	guildTokenCoin := sdk.NewCoin(cache.GetBankDenom(), amountToken)
	guildTokenCoins := sdk.NewCoins(guildTokenCoin)

	// Confiscate
	playerAcc, errAddr := sdk.AccAddressFromBech32(address)
	if errAddr != nil {
		return errAddr
	}
	errConfiscate := cache.CC.k.bankKeeper.SendCoinsFromAccountToModule(cache.CC.ctx, playerAcc, types.ModuleName, guildTokenCoins)
	if errConfiscate != nil {
		return errConfiscate
	}

	// Burn the Guild Token
	errBurn := cache.CC.k.bankKeeper.BurnCoins(cache.CC.ctx, types.ModuleName, guildTokenCoins)
	if errBurn != nil {
		return errBurn
	}

	ctxSDK := sdk.UnwrapSDKContext(cache.CC.ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildBankConfiscateAndBurn{&types.EventGuildBankConfiscateAndBurnDetail{GuildId: cache.GetGuildId(), AmountToken: amountToken.Uint64(), Address: address}})

	return nil
}




func (cache *GuildCache) SetEndpoint(endpoint string) {
    if (!cache.GuildLoaded) {
        cache.LoadGuild()
    }
    cache.Guild.Endpoint = endpoint
    cache.Changed = true
}

func (cache *GuildCache) SetOwner(owner string) {
    if (!cache.GuildLoaded) {
        cache.LoadGuild()
    }

    cache.CC.PermissionAdd(GetObjectPermissionIDBytes(cache.ID(), owner), types.PermGuildAll)
    cache.Guild.Owner = owner
    cache.Changed = true
}


func (cache *GuildCache) SetJoinInfusionMinimumBypassByRequest(level types.GuildJoinBypassLevel) {
    if (!cache.GuildLoaded) {
        cache.LoadGuild()
    }
    cache.Guild.JoinInfusionMinimumBypassByRequest = level
    cache.Changed = true
}


func (cache *GuildCache) SetJoinInfusionMinimumBypassByInvite(level types.GuildJoinBypassLevel) {
    if (!cache.GuildLoaded) {
        cache.LoadGuild()
    }
    cache.Guild.JoinInfusionMinimumBypassByInvite = level
    cache.Changed = true
}

func (cache *GuildCache) SetJoinInfusionMinimum(minimum uint64) {
    if (!cache.GuildLoaded) {
        cache.LoadGuild()
    }
    cache.Guild.JoinInfusionMinimum = minimum
    cache.Changed = true
}


func (cache *GuildCache) SetEntrySubstationId(substationId string) {
    if (!cache.GuildLoaded) {
        cache.LoadGuild()
    }
    cache.Guild.EntrySubstationId = substationId
    cache.Changed = true
}

func (cache *GuildCache) SetPrimaryReactorId(reactorId string) {
    if (!cache.GuildLoaded) {
        cache.LoadGuild()
    }
    cache.Guild.PrimaryReactorId = reactorId
    cache.Changed = true
}

func (cache *GuildCache) SetBankConvertInFee(fee math.LegacyDec) error {
    if (!cache.GuildLoaded) {
        cache.LoadGuild()
    }

    // A nil dec panics on the range comparison below (and on marshal), so reject
    // it before any arithmetic. A msg-handler panic is a chain-halt DoS vector.
    if fee.IsNil() {
        return types.NewParameterValidationError("bank_convert_in_fee", 0, "nil")
    }

    // 0 <= fee <= 1
    if (!fee.GTE(math.LegacyZeroDec())) || (!fee.LTE(math.LegacyOneDec())) {
        return types.NewParameterValidationError("bank_convert_in_fee", 0, "out_of_range")
    }

    cache.Guild.BankConvertInFee = fee
    cache.Changed = true
    return nil
}

func (cache *GuildCache) SetBankConvertOutFee(fee math.LegacyDec) error {
    if (!cache.GuildLoaded) {
        cache.LoadGuild()
    }

    if fee.IsNil() {
        return types.NewParameterValidationError("bank_convert_out_fee", 0, "nil")
    }

    // 0 <= fee <= 1
    if (!fee.GTE(math.LegacyZeroDec())) || (!fee.LTE(math.LegacyOneDec())) {
        return types.NewParameterValidationError("bank_convert_out_fee", 0, "out_of_range")
    }

    cache.Guild.BankConvertOutFee = fee
    cache.Changed = true
    return nil
}

func (cache *GuildCache) SetEntryRank(entryRank uint64) {
    if !cache.GuildLoaded {
        cache.LoadGuild()
    }
    cache.Guild.EntryRank = entryRank
    cache.Changed = true
}

func (cache *GuildCache) GetName() string {
    if !cache.GuildLoaded {
        cache.LoadGuild()
    }
    return cache.Guild.Name
}

func (cache *GuildCache) GetPfp() string {
    if !cache.GuildLoaded {
        cache.LoadGuild()
    }
    return cache.Guild.Pfp
}

func (cache *GuildCache) SetName(name string) {
    if !cache.GuildLoaded {
        cache.LoadGuild()
    }
    cache.Guild.Name = name
    cache.Changed = true
}

func (cache *GuildCache) SetPfp(pfp string) {
    if !cache.GuildLoaded {
        cache.LoadGuild()
    }
    cache.Guild.Pfp = pfp
    cache.Changed = true
}

func (cache *GuildCache) CanUpdateUGCBy(activePlayer *PlayerCache) error {
    return cache.CC.UGCPermissionCheck(cache, activePlayer)
}

