package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	"cosmossdk.io/math"
)

func (k msgServer) GuildBankMint(goCtx context.Context, msg *types.MsgGuildBankMint) (*types.MsgGuildBankMintResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    activePlayer, _ := k.GetPlayerCacheFromAddress(ctx, msg.Creator)

    guild := k.GetGuildCacheFromId(ctx, activePlayer.GetGuildId())

    permissionError := guild.CanAdministrateBank(&activePlayer)
    if (permissionError != nil) {
        return &types.MsgGuildBankMintResponse{}, permissionError
    }

    amountAlphaInt := math.NewIntFromUint64(msg.AmountAlpha)
    amountTokenInt := math.NewIntFromUint64(msg.AmountToken)

    err := guild.BankMint(amountAlphaInt, amountTokenInt, &activePlayer);

	return &types.MsgGuildBankMintResponse{}, err
}
