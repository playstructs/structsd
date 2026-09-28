package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgPermissionGrantOnAddress(t *testing.T) {
	for _, name := range []string{"primary signer", "exact secondary signer", "unregistered signer", "unregistered target", "other player target", "restricted secondary signer", "zero grant"} {
		t.Run(name, func(t *testing.T) {
			k, ms, goCtx := setupMsgServer(t)
			ctx := sdk.UnwrapSDKContext(goCtx)
			primary := sdk.AccAddress("permission_primary01")
			player := testAppendPlayer(k, ctx, types.Player{Creator: primary.String(), PrimaryAddress: primary.String()})
			target := sdk.AccAddress("permission_subject01")
			signer := sdk.AccAddress("permission_signer001")
			require.NoError(t, k.SetPlayerIndexForAddress(ctx, target.String(), player.Index))
			require.NoError(t, k.SetPlayerIndexForAddress(ctx, signer.String(), player.Index))
			foreign := sdk.AccAddress("permission_foreign01")
			testAppendPlayer(k, ctx, types.Player{Creator: foreign.String(), PrimaryAddress: foreign.String()})
			targetID := keeperlib.GetAddressPermissionIDBytes(target.String())
			k.SetPermissionsByBytes(ctx, targetID, types.PermUpdate|types.PermDelete)
			k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(signer.String()), types.PermPlay)
			creator, subject, permissions := primary.String(), target.String(), types.PermPlay
			rejected := false
			switch name {
			case "exact secondary signer":
				creator = signer.String()
			case "unregistered signer":
				creator, rejected = sdk.AccAddress("permission_unknown01").String(), true
			case "unregistered target":
				subject, rejected = sdk.AccAddress("permission_unknown01").String(), true
			case "other player target":
				subject, rejected = foreign.String(), true
			case "restricted secondary signer":
				creator, rejected = signer.String(), true
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.Permissionless)
			case "zero grant":
				permissions, rejected = types.Permissionless, true

			}
			before := k.GetAllPermissionExport(ctx)
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			response, err := ms.PermissionGrantOnAddress(ctx, &types.MsgPermissionGrantOnAddress{Creator: creator, Address: subject, Permissions: uint64(permissions)})
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
			if rejected {
				require.Error(t, err)
				if name == "primary cannot be narrowed" {
					require.ErrorContains(t, err, "must keep full access")
				}
				require.Equal(t, before, k.GetAllPermissionExport(ctx))
				require.Zero(t, permissionEvents)
			} else {
				require.NoError(t, err)
				require.Equal(t, types.PermUpdate|types.PermDelete|types.PermPlay, k.GetPermissionsByBytes(ctx, targetID))
				require.Equal(t, 1, permissionEvents)
			}
			require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(primary.String())))
			require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(foreign.String())))
			stored, found := k.GetPlayer(ctx, player.Id)
			require.True(t, found)
			require.Equal(t, player, stored)
		})
	}
}
