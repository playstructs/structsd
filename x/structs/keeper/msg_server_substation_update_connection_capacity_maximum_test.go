package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgSubstationUpdateConnectionCapacityMaximum(t *testing.T) {
	for _, name := range []string{"owner with exact key bit", "explicit delegate", "clear maximum", "maximum above share", "unchanged maximum", "missing substation", "unregistered signer", "unprivileged player", "key lacks update", "closing substation"} {
		t.Run(name, func(t *testing.T) {
			k, ms, goCtx := setupMsgServer(t)
			ctx := sdk.UnwrapSDKContext(goCtx)
			ownerAddress := sdk.AccAddress("substation_cap_owner")
			owner := testAppendPlayer(k, ctx, types.Player{Creator: ownerAddress.String(), PrimaryAddress: ownerAddress.String()})
			otherAddress := sdk.AccAddress("substation_cap_actor")
			other := testAppendPlayer(k, ctx, types.Player{Creator: otherAddress.String(), PrimaryAddress: otherAddress.String()})
			substation, _, err := testAppendSubstation(k, ctx, types.Allocation{}, owner)
			require.NoError(t, err)

			// 1000 raw capacity leaves 900 usable, shared three ways: 300 uncapped.
			maximumID := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacityMaximum, substation.Id)
			shareID := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, substation.Id)
			k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, substation.Id), 1000)
			k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, substation.Id), 3)
			k.SetGridAttribute(ctx, maximumID, 250)
			k.SetGridAttribute(ctx, shareID, 250)

			creator, substationID, wantError := owner.Creator, substation.Id, ""
			value, wantShare, wantWrites := uint64(200), uint64(200), 2
			k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermUpdate)
			switch name {
			case "explicit delegate":
				creator = other.Creator
				k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(substation.Id, other.Id), types.PermUpdate)
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermUpdate)
			case "clear maximum":
				value, wantShare = 0, 300
			case "maximum above share":
				value, wantShare = 5000, 300
			case "unchanged maximum":
				value, wantShare, wantWrites = 250, 250, 0
			case "missing substation":
				substationID, wantError = "4-999", "not found"
			case "unregistered signer":
				creator, wantError = sdk.AccAddress("substation_cap_nokey").String(), "player"
			case "unprivileged player":
				creator, wantError = other.Creator, "permission"
			case "key lacks update":
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermAll&^types.PermUpdate)
				wantError = "permission"
			case "closing substation":
				closing := substation
				closing.DeletionPending = true
				k.SetSubstation(ctx, closing)
				wantError = "substation_deleting"
			}

			grid := k.GetAllGridExport(ctx)
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			response, err := ms.SubstationUpdateConnectionCapacityMaximum(ctx, &types.MsgSubstationUpdateConnectionCapacityMaximum{
				Creator: creator, SubstationId: substationID, ConnectionCapacityMaximum: value,
			})
			require.NotNil(t, response)

			gridEvents := 0
			for _, event := range ctx.EventManager().ABCIEvents() {
				if event.Type == "structs.structs.EventGrid" {
					gridEvents++
				}
			}
			if wantError != "" {
				require.ErrorContains(t, err, wantError)
				require.Zero(t, gridEvents)
				require.Equal(t, grid, k.GetAllGridExport(ctx))
				return
			}
			require.NoError(t, err)
			require.Equal(t, value, k.GetGridAttribute(ctx, maximumID))
			require.Equal(t, wantShare, k.GetGridAttribute(ctx, shareID))
			require.Equal(t, wantWrites, gridEvents, "one grid event per changed attribute")
		})
	}
}
