package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildCreate(goCtx context.Context, msg *types.MsgGuildCreate) (*types.MsgGuildCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    player, playerErr := cc.GetPlayerByAddress(msg.Creator)
    if playerErr != nil {
        return &types.MsgGuildCreateResponse{}, types.NewPlayerRequiredError(msg.Creator, "guild_create")
    }

    // Need to change this to be more open.
    // TODO should probably accept a reactor ID instead of using this method for figuring it out
    var playerAddress sdk.AccAddress
    playerAddress, _ = sdk.AccAddressFromBech32(msg.Creator)

    var validatorAddress sdk.ValAddress
    validatorAddress = playerAddress.Bytes()

    reactorBytes, _ := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
    reactor := cc.GetReactor(string(reactorBytes))

    if reactor.CheckReactor() != nil {
        return &types.MsgGuildCreateResponse{}, types.NewReactorError("guild_create", "required").WithAddress(msg.Creator, "validator")
    }

    reactorPermissionCheck := reactor.CanCreateGuildBy(player)
    if reactorPermissionCheck != nil {
        return &types.MsgGuildCreateResponse{}, reactorPermissionCheck
    }

    if (msg.EntrySubstationId != "") {
        // Check that the Substation exists
        substation := cc.GetSubstation(msg.EntrySubstationId)
        if substation.CheckSubstation() != nil {
            return &types.MsgGuildCreateResponse{}, types.NewObjectNotFoundError("substation", msg.EntrySubstationId)
        }

        substationPermissionErr := substation.CanManageConnectionsBy(player)
        if substationPermissionErr != nil {
            return &types.MsgGuildCreateResponse{}, substationPermissionErr
        }
    }

    // TODO Fix Guild Creation
    guild := k.AppendGuild(ctx, msg.Endpoint, msg.EntrySubstationId, reactor.GetReactor(), player.GetPlayer())

    player.SetGuild(guild.Id)
    reactor.SetGuild(guild.Id)

	cc.CommitAll()
	return &types.MsgGuildCreateResponse{GuildId: guild.Id}, nil
}
