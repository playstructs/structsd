package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildBankRedeem(goCtx context.Context, msg *types.MsgGuildBankRedeem) (*types.MsgGuildBankRedeemResponse, error) {
	emptyResponse := &types.MsgGuildBankRedeemResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	activePlayer, lookupErr := cc.GetPlayerByAddress(msg.Creator)
	if lookupErr != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "guild_bank_redeem")
	}

	if msg.MinAmountAlpha == 0 {
		return emptyResponse, types.NewParameterValidationError("minAmountAlpha", 0, "must_be_positive")
	}

	permissionErr := activePlayer.CanTransferTokensBy(activePlayer)
	if permissionErr != nil {
		return emptyResponse, permissionErr
	}

	guildId, denomErr := types.ParseGuildBankDenom(msg.AmountToken.Denom)
	if denomErr != nil {
		return emptyResponse, denomErr
	}

	guild := cc.GetGuild(guildId)
	if !guild.LoadGuild() {
		return emptyResponse, types.NewObjectNotFoundError("guild", guild.GetGuildId())
	}

	_, err := guild.BankRedeem(msg.AmountToken.Amount, math.NewIntFromUint64(msg.MinAmountAlpha), activePlayer)
	if err != nil {
		return emptyResponse, err
	}

	cc.CommitAll()
	return &types.MsgGuildBankRedeemResponse{}, nil
}
