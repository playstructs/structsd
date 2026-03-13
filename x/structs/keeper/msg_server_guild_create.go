package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildCreate(goCtx context.Context, msg *types.MsgGuildCreate) (*types.MsgGuildCreateResponse, error) {
    emptyResponse := &types.MsgGuildCreateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    player, playerErr := cc.GetPlayerByAddress(msg.Creator)
    if playerErr != nil {
        return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "guild_create")
    }

    reactor := cc.GetReactor(msg.ReactorId)
    if reactor.CheckReactor() != nil {
        return emptyResponse, types.NewReactorError("guild_create", "required").WithAddress(msg.Creator, "validator")
    }

    reactorPermissionCheck := reactor.CanCreateGuildBy(player)
    if reactorPermissionCheck != nil {
        return emptyResponse, reactorPermissionCheck
    }

    if (msg.EntrySubstationId != "") {
        // Check that the Substation exists
        substation := cc.GetSubstation(msg.EntrySubstationId)
        if substation.CheckSubstation() != nil {
            return emptyResponse, types.NewObjectNotFoundError("substation", msg.EntrySubstationId)
        }

        substationPermissionErr := substation.CanManageConnectionsBy(player)
        if substationPermissionErr != nil {
            return emptyResponse, substationPermissionErr
        }
    }

    // TODO Fix Guild Creation
    guild := k.AppendGuild(ctx, msg.Endpoint, msg.EntrySubstationId, reactor.GetReactor(), player.GetPlayer())

    player.SetGuild(guild.Id)
    player.SetGuildRank(1)

    if reactor.GetReactor().GuildId == "" {
        reactor.SetGuild(guild.Id)
    }

	cc.CommitAll()
	return &types.MsgGuildCreateResponse{GuildId: guild.Id}, nil
}
