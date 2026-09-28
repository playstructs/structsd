package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgAllocationDelete(t *testing.T) {
	for _, name := range []string{"dynamic allocation", "missing allocation", "unregistered signer", "static allocation", "unprivileged player", "signing key lacks delete authority"} {
		t.Run(name, func(t *testing.T) {
			k, ms, goCtx := setupMsgServer(t)
			ctx := sdk.UnwrapSDKContext(goCtx)
			ownerAddress := sdk.AccAddress("allocation_owner_001")
			owner := testAppendPlayer(k, ctx, types.Player{Creator: ownerAddress.String(), PrimaryAddress: ownerAddress.String()})
			otherAddress := sdk.AccAddress("allocation_other_001")
			other := testAppendPlayer(k, ctx, types.Player{Creator: otherAddress.String(), PrimaryAddress: otherAddress.String()})
			allocationType := types.AllocationType_dynamic
			if name == "static allocation" {
				allocationType = types.AllocationType_static
			}
			allocation, err := testAppendAllocation(k, ctx, types.Allocation{
				SourceObjectId: owner.Id, Controller: owner.Id, Type: allocationType,
			}, 100)
			require.NoError(t, err)
			// Exercise the allocation's explicit delete permission, independently of
			// permission to modify allocations on its source.
			permissionID := keeperlib.GetObjectPermissionIDBytes(allocation.Id, owner.Id)
			k.SetPermissionsByBytes(ctx, permissionID, types.PermDelete)
			k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(owner.Creator), types.PermDelete)
			msg := &types.MsgAllocationDelete{Creator: owner.Creator, AllocationId: allocation.Id}
			wantError := ""
			switch name {
			case "missing allocation":
				msg.AllocationId = "6-999999"
				wantError = "not found"
			case "unregistered signer":
				msg.Creator = sdk.AccAddress("allocation_unknown01").String()
				wantError = "player"
			case "static allocation":
				wantError = "allocation type (static vs dynamic)"
			case "unprivileged player":
				msg.Creator = other.Creator
				wantError = "permission"
			case "signing key lacks delete authority":
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(owner.Creator), types.Permissionless)
				wantError = "permission"
			}
			powerID := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocation.Id)
			beforePower := k.GetGridAttribute(ctx, powerID)
			beforePermissions := k.GetPermissionsByObject(ctx, allocation.Id)
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			resp, err := ms.AllocationDelete(ctx, msg)
			require.NotNil(t, resp)
			stored, found := k.GetAllocation(ctx, allocation.Id)
			if wantError != "" {
				require.ErrorContains(t, err, wantError)
				require.True(t, found)
				require.Equal(t, allocation, stored)
				require.Equal(t, beforePower, k.GetGridAttribute(ctx, powerID))
				require.Equal(t, beforePermissions, k.GetPermissionsByObject(ctx, allocation.Id))
				for _, event := range ctx.EventManager().ABCIEvents() {
					require.NotContains(t, []string{"structs.structs.EventDelete", "structs.structs.EventAllocation", "structs.structs.EventPermission"}, event.Type)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, allocation.Id, resp.AllocationId)
				require.False(t, found)
				require.Empty(t, k.GetPermissionsByObject(ctx, allocation.Id))
			}
		})
	}
}
