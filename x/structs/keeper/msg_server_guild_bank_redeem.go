package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	"strings"

)

func (k msgServer) GuildBankRedeem(goCtx context.Context, msg *types.MsgGuildBankRedeem) (*types.MsgGuildBankRedeemResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    activePlayer, lookupErr := k.GetPlayerCacheFromAddress(ctx, msg.Creator)
    if lookupErr != nil {
        return &types.MsgGuildBankRedeemResponse{}, types.NewPlayerRequiredError(msg.Creator, "guild_bank_redeem")
    }

    // TODO permission check on the address to look for Asset permissions
    denomSlice := strings.Split(msg.AmountToken.Denom,".")
    if len(denomSlice) != 2 {
        return &types.MsgGuildBankRedeemResponse{}, types.NewParameterValidationError("denom", 0, "invalid_format")
    }

    guild := k.GetGuildCacheFromId(ctx, denomSlice[1])
    if !guild.LoadGuild() {
        return &types.MsgGuildBankRedeemResponse{}, types.NewObjectNotFoundError("guild", guild.GetGuildId())
    }

    err := guild.BankRedeem(msg.AmountToken.Amount, &activePlayer);

	return &types.MsgGuildBankRedeemResponse{}, err
}
