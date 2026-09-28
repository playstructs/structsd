package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
	"testing"
)

func TestMsgPermissionGrantOnObject(t *testing.T) {
	for _, name := range []string{"owner", "explicit delegate", "missing player", "missing object", "unregistered signer", "unprivileged player", "restricted key", "zero grant"} {
		t.Run(name, func(t *testing.T) {
			k, ms, goCtx := setupMsgServer(t)
			ctx := sdk.UnwrapSDKContext(goCtx)
			gs := testCreateGuild(k, ctx)
			address := sdk.AccAddress("permission_target_01")
			target := testAppendPlayer(k, ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
			delegateAddress := sdk.AccAddress("permission_delegate1")
			delegate := testAppendPlayer(k, ctx, types.Player{Creator: delegateAddress.String(), PrimaryAddress: delegateAddress.String()})
			targetID := keeperlib.GetObjectPermissionIDBytes(gs.Guild.Id, target.Id)
			k.SetPermissionsByBytes(ctx, targetID, types.PermUpdate|types.PermDelete)
			creator, object, player := gs.GuildOwner.Creator, gs.Guild.Id, target.Id
			permissions := types.PermPlay
			needed := types.PermPlay
			k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), needed|types.PermAdmin)
			wantError := ""
			switch name {
			case "explicit delegate":
				creator = delegate.Creator
				grant := needed
				k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(gs.Guild.Id, delegate.Id), grant|types.PermAdmin)
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), needed|types.PermAdmin)
			case "missing player":
				player, wantError = "1-999999", "not found"
			case "missing object":
				object, wantError = "6-999999", "permission"
			case "unregistered signer":
				creator, wantError = sdk.AccAddress("permission_unknown_1").String(), "player"
			case "unprivileged player":
				creator, wantError = delegate.Creator, "permission"
			case "restricted key":
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.Permissionless)
				wantError = "permission"
			case "zero grant":
				permissions, wantError = types.Permissionless, "permissions"
			}
			before := k.GetPermissionsByObject(ctx, gs.Guild.Id)
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			response, err := ms.PermissionGrantOnObject(ctx, &types.MsgPermissionGrantOnObject{
				Creator: creator, ObjectId: object, PlayerId: player, Permissions: uint64(permissions)})
			require.NotNil(t, response)
			permissionEvents := 0
			for _, event := range ctx.EventManager().ABCIEvents() {
				if event.Type == "structs.structs.EventPermission" {
					decoded, decodeErr := sdk.ParseTypedEvent(event)
					require.NoError(t, decodeErr)
					require.Equal(t, &types.PermissionRecord{PermissionId: string(targetID), Value: uint64(types.PermUpdate | types.PermDelete | types.PermPlay)}, decoded.(*types.EventPermission).PermissionRecord)
					permissionEvents++
				}
			}
			if wantError != "" {
				require.ErrorContains(t, err, wantError)
				require.Equal(t, before, k.GetPermissionsByObject(ctx, gs.Guild.Id))
				require.Zero(t, permissionEvents)
				require.Equal(t, types.Permissionless, k.GetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(object, "1-999999")))
			} else {
				require.NoError(t, err)
				require.Equal(t, types.PermUpdate|types.PermDelete|types.PermPlay, k.GetPermissionsByBytes(ctx, targetID))
				require.Equal(t, 1, permissionEvents)
			}
			guild, found := k.GetGuild(ctx, gs.Guild.Id)
			require.True(t, found)
			require.Equal(t, gs.Guild, guild)
			storedTarget, found := k.GetPlayer(ctx, target.Id)
			require.True(t, found)
			require.Equal(t, target, storedTarget)
		})
	}
}
