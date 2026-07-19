package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildBankConvert(goCtx context.Context, msg *types.MsgGuildBankConvert) (*types.MsgGuildBankConvertResponse, error) {
	emptyResponse := &types.MsgGuildBankConvertResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	activePlayer, lookupErr := cc.GetPlayerByAddress(msg.Creator)
	if lookupErr != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "guild_bank_convert")
	}

	if msg.MinAmountToken == 0 {
		return emptyResponse, types.NewParameterValidationError("minAmountToken", 0, "must_be_positive")
	}

	// The only authorization gate is the player's own token-transfer
	// permission, matching redeem.
	permissionErr := activePlayer.CanTransferTokensBy(activePlayer)
	if permissionErr != nil {
		return emptyResponse, permissionErr
	}

	guild := cc.GetGuild(msg.GuildId)
	if !guild.LoadGuild() {
		return emptyResponse, types.NewObjectNotFoundError("guild", msg.GuildId)
	}

	_, err := guild.BankConvert(math.NewIntFromUint64(msg.AmountAlpha), math.NewIntFromUint64(msg.MinAmountToken), activePlayer)
	if err != nil {
		return emptyResponse, err
	}

	cc.CommitAll()
	return &types.MsgGuildBankConvertResponse{}, nil
}
