package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// GuildBankConvertToken converts one guild token into another atomically:
// source token -> ualpha (source out-fee) -> target token (target in-fee). It
// composes BankRedeem and BankConvert through the player account, so each leg
// emits its own event and enforces its own guild's fee. The min-output guard
// applies only to the final target-token amount (leg 1 runs unguarded); if
// either leg fails the whole transaction reverts.
func (k msgServer) GuildBankConvertToken(goCtx context.Context, msg *types.MsgGuildBankConvertToken) (*types.MsgGuildBankConvertTokenResponse, error) {
	emptyResponse := &types.MsgGuildBankConvertTokenResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	activePlayer, lookupErr := cc.GetPlayerByAddress(msg.Creator)
	if lookupErr != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "guild_bank_convert_token")
	}

	if msg.MinAmountToken == 0 {
		return emptyResponse, types.NewParameterValidationError("minAmountToken", 0, "must_be_positive")
	}

	permissionErr := activePlayer.CanTransferTokensBy(activePlayer)
	if permissionErr != nil {
		return emptyResponse, permissionErr
	}

	sourceGuildId, denomErr := types.ParseGuildBankDenom(msg.AmountToken.Denom)
	if denomErr != nil {
		return emptyResponse, denomErr
	}

	// A same-guild convert would only donate two fees for nothing.
	if sourceGuildId == msg.GuildId {
		return emptyResponse, types.NewParameterValidationError("guildId", 0, "same_guild")
	}

	sourceGuild := cc.GetGuild(sourceGuildId)
	if !sourceGuild.LoadGuild() {
		return emptyResponse, types.NewObjectNotFoundError("guild", sourceGuildId)
	}

	targetGuild := cc.GetGuild(msg.GuildId)
	if !targetGuild.LoadGuild() {
		return emptyResponse, types.NewObjectNotFoundError("guild", msg.GuildId)
	}

	// Leg 1: source token -> ualpha. No slippage guard here; the guard applies
	// to the final target-token output.
	bridgeAlpha, redeemErr := sourceGuild.BankRedeem(msg.AmountToken.Amount, math.ZeroInt(), activePlayer)
	if redeemErr != nil {
		return emptyResponse, redeemErr
	}

	// Leg 2: ualpha -> target token, guarded by minAmountToken.
	tokensOut, convertErr := targetGuild.BankConvert(bridgeAlpha, math.NewIntFromUint64(msg.MinAmountToken), activePlayer)
	if convertErr != nil {
		return emptyResponse, convertErr
	}

	_ = ctx.EventManager().EmitTypedEvent(&types.EventGuildBankConvertToken{
		EventGuildBankConvertTokenDetail: &types.EventGuildBankConvertTokenDetail{
			SourceGuildId: sourceGuildId, TargetGuildId: msg.GuildId,
			AmountTokenIn: msg.AmountToken.Amount.Uint64(), BridgeAlpha: bridgeAlpha.Uint64(),
			AmountTokenOut: tokensOut.Uint64(), PlayerId: activePlayer.GetPlayerId(),
		},
	})

	cc.CommitAll()
	return &types.MsgGuildBankConvertTokenResponse{}, nil
}
