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

    guildObjectPermissionId := GetObjectPermissionIDBytes(msg.GuildId, player.GetPlayerId())
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)

    if (!cc.PermissionHasOneOf(guildObjectPermissionId, types.PermissionUpdate)) {
        return &types.MsgGuildUpdateResponse{}, types.NewPermissionError("player", player.GetPlayerId(), "guild", msg.GuildId, uint64(types.PermissionUpdate), "guild_update")
    }

    // Make sure the address calling this has Associate permissions
    if (!cc.PermissionHasOneOf(addressPermissionId, types.PermissionAssets)) {
        return &types.MsgGuildUpdateResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionAssets), "guild_management")
    }

    if (msg.EntrySubstationId != "") {

        substationObjectPermissionId    := GetObjectPermissionIDBytes(msg.EntrySubstationId, player.GetPlayerId())

        // check that the calling player has substation permissions
        if (!cc.PermissionHasOneOf(substationObjectPermissionId, types.PermissionGrid)) {
            return &types.MsgGuildUpdateResponse{}, types.NewPermissionError("player", player.GetPlayerId(), "substation", msg.EntrySubstationId, uint64(types.PermissionGrid), "substation_connect")
        }

        guild.SetEntrySubstationId(msg.EntrySubstationId)
    }

	cc.CommitAll()
	return &types.MsgGuildUpdateResponse{}, nil
}
