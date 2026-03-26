package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	"cosmossdk.io/math"
)

func (k msgServer) GuildBankMint(goCtx context.Context, msg *types.MsgGuildBankMint) (*types.MsgGuildBankMintResponse, error) {
    emptyResponse := &types.MsgGuildBankMintResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)


    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    activePlayer, lookupErr := cc.GetPlayerByAddress(msg.Creator)
    if lookupErr != nil {
        return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "guild_bank_mint")
    }

    guild := cc.GetGuild(activePlayer.GetGuildId())

    permissionError := guild.CanMintTokenBy(activePlayer)
    if (permissionError != nil) {
        return emptyResponse, permissionError
    }

    amountAlphaInt := math.NewIntFromUint64(msg.AmountAlpha)
    amountTokenInt := math.NewIntFromUint64(msg.AmountToken)

    err := guild.BankMint(amountAlphaInt, amountTokenInt, activePlayer)
    if err != nil {
        return emptyResponse, err
    }

	cc.CommitAll()
	return &types.MsgGuildBankMintResponse{}, nil
}
