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
        return &types.MsgGuildCreateResponse{}, sdkerrors.Wrapf(types.ErrReactorRequired, "Guild creation requires Reactor but none associated with %s", msg.Creator)
    }

    // Currently, no real reason to do permission checks that the player can
    // add the reactor, since the player account IS the reactor
        // TODO
        // Although, this should be changed so that if an account
        // that includes a validator as an associated address, it's able to perform this step


    playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (playerIndex == 0) {
        // should really never get here as player creation is triggered
        // during reactor initialization
        return &types.MsgGuildCreateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Guild creation requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayerFromIndex(ctx, playerIndex, false)


    if (msg.EntrySubstationId != "") {
        // check that the calling player has substation permissions
        substationObjectPermissionId := GetObjectPermissionIDBytes(msg.EntrySubstationId, player.Id)
        if (!k.PermissionHasOneOf(ctx,substationObjectPermissionId, types.Permission(types.SubstationPermissionConnectPlayer))) {
            return &types.MsgGuildCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationPlayerConnect, "Calling player (%s) has no Substation Connect Player permissions ", player.Id)
        }
    }

    guild := k.AppendGuild(ctx, msg.Endpoint, msg.EntrySubstationId, reactor, player)

    player.GuildId = guild.Id
    k.SetPlayer(ctx, player)

    reactor.GuildId = guild.Id
    k.SetReactor(ctx, reactor)

	return &types.MsgGuildCreateResponse{GuildId: guild.Id}, nil
}
