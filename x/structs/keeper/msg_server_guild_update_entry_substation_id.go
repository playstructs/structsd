package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildUpdateEntrySubstationId(goCtx context.Context, msg *types.MsgGuildUpdateEntrySubstationId) (*types.MsgGuildUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)


    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    player, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
     return &types.MsgGuildUpdateResponse{}, types.NewPlayerRequiredError(msg.Creator, "guild_update_entry_substation")
    }

    guild := cc.GetGuild(msg.GuildId)
    if guild.CheckGuild() != nil {
         return &types.MsgGuildUpdateResponse{}, types.NewObjectNotFoundError("guild", msg.GuildId)
    }

    permissionErr := guild.CanUpdateSubstationBy(player)
    if permissionErr != nil {
        return &types.MsgGuildUpdateResponse{}, permissionErr
    }

    if (msg.EntrySubstationId != "") {
        substation := cc.GetSubstation(msg.EntrySubstationId)
        if substation.CheckSubstation() != nil {
            return &types.MsgGuildUpdateResponse{}, types.NewObjectNotFoundError("substation", msg.EntrySubstationId)
        }

        substationPermissionErr := substation.CanManageConnectionsBy(player)
        if substationPermissionErr != nil {
            return &types.MsgGuildUpdateResponse{}, substationPermissionErr
        }

        guild.SetEntrySubstationId(msg.EntrySubstationId)
    }

	cc.CommitAll()
	return &types.MsgGuildUpdateResponse{}, nil
}
