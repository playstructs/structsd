package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgAllocationTransfer(t *testing.T) {
	for _, name := range []string{"transfer", "connected allocation", "missing allocation", "missing controller", "unregistered signer", "unprivileged player", "key lacks admin"} {
		t.Run(name, func(t *testing.T) {
			f := setupTransferFixture(t, "1-1", "transfer_owner_00001")
			address := sdk.AccAddress("transfer_recipient01")
			recipient := testAppendPlayer(f.k, f.ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
			testPermissionAdd(f.k, f.ctx, f.ownerId, types.PermUpdate|types.PermDelete|types.PermAllocationConnection)
			var stationID string
			if name == "connected allocation" {
				station, err := f.ms.SubstationCreate(f.ctx, &types.MsgSubstationCreate{Creator: f.owner.Creator, AllocationId: f.alloc.Id})
				require.NoError(t, err)
				stationID = station.SubstationId
			}
			msg := &types.MsgAllocationTransfer{Creator: f.owner.Creator, AllocationId: f.alloc.Id, Controller: recipient.Id}
			wantError := ""
			switch name {
			case "missing allocation":
				msg.AllocationId = "6-999999"
				wantError = "not found"
			case "missing controller":
				msg.Controller = "1-999999"
				wantError = "not found"
			case "unregistered signer":
				msg.Creator = sdk.AccAddress("transfer_unknown_01").String()
				wantError = "player"
			case "unprivileged player":
				msg.Creator = recipient.Creator
				wantError = "permission"
			case "key lacks admin":
				f.k.SetPermissionsByBytes(f.ctx, keeperlib.GetAddressPermissionIDBytes(f.owner.Creator), types.PermAllocationConnection)
				wantError = "permission"
			}
			before, found := f.k.GetAllocation(f.ctx, f.alloc.Id)
			require.True(t, found)
			beforePermissions := f.k.GetPermissionsByObject(f.ctx, f.alloc.Id)
			powerID := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, f.alloc.Id)
			beforePower := f.k.GetGridAttribute(f.ctx, powerID)
			beforeSources := f.k.GetAllAllocationIdBySourceIndex(f.ctx, f.alloc.SourceObjectId)
			f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
			resp, err := f.ms.AllocationTransfer(f.ctx, msg)
			require.NotNil(t, resp)
			stored, found := f.k.GetAllocation(f.ctx, f.alloc.Id)
			require.True(t, found)
			if wantError != "" {
				require.ErrorContains(t, err, wantError)
				require.Equal(t, before, stored)
				require.Equal(t, beforePermissions, f.k.GetPermissionsByObject(f.ctx, f.alloc.Id))
				for _, event := range f.ctx.EventManager().ABCIEvents() {
					require.NotContains(t, []string{"structs.structs.EventAllocation", "structs.structs.EventPermission"}, event.Type)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, f.alloc.Id, resp.AllocationId)
				before.Controller = recipient.Id
				require.Equal(t, before, stored, "transfer changes only the controller")
				allocationEvents := 0
				for _, event := range f.ctx.EventManager().ABCIEvents() {
					if event.Type != "structs.structs.EventAllocation" {
						continue
					}
					decoded, decodeErr := sdk.ParseTypedEvent(event)
					require.NoError(t, decodeErr)
					require.Equal(t, &stored, decoded.(*types.EventAllocation).Allocation)
					allocationEvents++
				}
				require.Equal(t, 1, allocationEvents, "indexers receive the final controller exactly once")
				require.Equal(t, types.PermUpdate|types.PermDelete, f.k.GetPermissionsByBytes(f.ctx, f.ownerId))
				require.Equal(t, types.PermAllocationConnection, f.k.GetPermissionsByBytes(f.ctx, keeperlib.GetObjectPermissionIDBytes(f.alloc.Id, recipient.Id)))
			}
			require.Equal(t, beforePower, f.k.GetGridAttribute(f.ctx, powerID))
			require.Equal(t, beforeSources, f.k.GetAllAllocationIdBySourceIndex(f.ctx, f.alloc.SourceObjectId))
			if stationID != "" {
				require.Equal(t, stationID, stored.DestinationId)
				require.Equal(t, uint64(100), f.k.GetGridAttribute(f.ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, stationID)))
			}
		})
	}
}
