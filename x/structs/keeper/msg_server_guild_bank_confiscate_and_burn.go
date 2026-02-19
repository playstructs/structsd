package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	"cosmossdk.io/math"

)

func (k msgServer) GuildBankConfiscateAndBurn(goCtx context.Context, msg *types.MsgGuildBankConfiscateAndBurn) (*types.MsgGuildBankConfiscateAndBurnResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)


    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    activePlayer, lookupErr := cc.GetPlayerByAddress(msg.Creator)
    if lookupErr != nil {
        return &types.MsgGuildBankConfiscateAndBurnResponse{}, types.NewPlayerRequiredError(msg.Creator, "guild_bank_confiscate")
    }

    guild := cc.GetGuild(activePlayer.GetGuildId())

    permissionError := guild.CanAdministrateBank(activePlayer)
    if (permissionError != nil) {
        return &types.MsgGuildBankConfiscateAndBurnResponse{}, permissionError
    }

    amountTokenInt := math.NewIntFromUint64(msg.AmountToken)
    err := guild.BankConfiscateAndBurn(amountTokenInt, msg.Address)
    if err != nil {
        return &types.MsgGuildBankConfiscateAndBurnResponse{}, err
    }

	cc.CommitAll()
	return &types.MsgGuildBankConfiscateAndBurnResponse{}, nil
}
