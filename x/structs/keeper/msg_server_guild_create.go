package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildCreate(goCtx context.Context, msg *types.MsgGuildCreate) (*types.MsgGuildCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}


    var playerAddress sdk.AccAddress
    playerAddress, _ = sdk.AccAddressFromBech32(msg.Creator)
    var validatorAddress sdk.ValAddress
    validatorAddress = playerAddress.Bytes()
    reactorBytes, _ := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
    reactor, reactorFound := k.GetReactorByBytes(ctx, reactorBytes, true)

    if (!reactorFound) {
        return nil, sdkerrors.Wrapf(types.ErrReactorRequired, "Guild creation requires Reactor but none associated with %s", msg.Creator)
    }

    playerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (playerId == 0) {
        // should really never get here as player creation is triggered
        // during reactor initialization
        return nil, sdkerrors.Wrapf(types.ErrPlayerRequired, "Guild creation requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayer(ctx, playerId)


    guild := k.AppendGuild(ctx, msg.Endpoint, reactor, player)

	return &types.MsgGuildCreateResponse{GuildId: guild.Id}, nil
}
