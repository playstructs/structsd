package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	"strings"

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

    permissionErr := activePlayer.CanTransferTokensBy(activePlayer)
    if permissionErr != nil {
        return emptyResponse, permissionErr
    }

    denomSlice := strings.Split(msg.AmountToken.Denom,".")
    if len(denomSlice) != 2 {
        return emptyResponse, types.NewParameterValidationError("denom", 0, "invalid_format")
    }

    guild := cc.GetGuild(denomSlice[1])
    if !guild.LoadGuild() {
        return emptyResponse, types.NewObjectNotFoundError("guild", guild.GetGuildId())
    }

    err := guild.BankRedeem(msg.AmountToken.Amount, activePlayer);

	cc.CommitAll()
	return emptyResponse, err
}
