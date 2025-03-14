package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
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
        return &types.MsgGuildBankRedeemResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Player not found ")
    }

    // TODO permission check on the address to look for Asset permissions

    guild := k.GetGuildCacheFromId(ctx, strings.Split(msg.AmountToken.Denom,".")[1])
    if !guild.LoadGuild() {
        return &types.MsgGuildBankRedeemResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Guild ID (%s) not found ", guild.GetGuildId())
    }

    err := guild.BankRedeem(msg.AmountToken.Amount, &activePlayer);


	return &types.MsgGuildBankRedeemResponse{}, err
}

