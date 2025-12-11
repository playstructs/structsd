package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgSubstationAllocationConnect(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create a player first
	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = k.AppendPlayer(ctx, player)

	// Set up source capacity
	sourceObjectId := "source-object"
	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceObjectId)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(1000))

	// Create an allocation
	allocation := types.Allocation{
		SourceObjectId: sourceObjectId,
		DestinationId:  "", // Must be empty for connection
		Type:           types.AllocationType_dynamic,
		Controller:     player.Creator,
	}
	createdAllocation, _, err := k.AppendAllocation(ctx, allocation, 100)
	require.NoError(t, err)

	// Create a substation
	substationAllocation := types.Allocation{
		SourceObjectId: sourceObjectId,
		DestinationId:  "",
		Type:           types.AllocationType_static,
		Controller:     player.Creator,
	}
	substationAlloc, _, err := k.AppendAllocation(ctx, substationAllocation, 100)
	require.NoError(t, err)

	substation, _, err := k.AppendSubstation(ctx, substationAlloc, player)
	require.NoError(t, err)

	// Grant permissions
	substationPermissionId := keeperlib.GetObjectPermissionIDBytes(substation.Id, player.Id)
	k.PermissionAdd(ctx, substationPermissionId, types.PermissionGrid)

	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(player.Creator)
	k.PermissionAdd(ctx, addressPermissionId, types.PermissionGrid)

	testCases := []struct {
		name      string
		input     *types.MsgSubstationAllocationConnect
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid allocation connection",
			input: &types.MsgSubstationAllocationConnect{
				Creator:       player.Creator,
				AllocationId:  createdAllocation.Id,
				DestinationId: substation.Id,
			},
			expErr: false,
		},
		{
			name: "allocation not found",
			input: &types.MsgSubstationAllocationConnect{
				Creator:       player.Creator,
				AllocationId:  "invalid-allocation",
				DestinationId: substation.Id,
			},
			expErr:    true,
			expErrMsg: "allocation not found",
		},
		{
			name: "substation not found",
			input: &types.MsgSubstationAllocationConnect{
				Creator:       player.Creator,
				AllocationId:  createdAllocation.Id,
				DestinationId: "invalid-substation",
			},
			expErr:    true,
			expErrMsg: "not found",
		},
		{
			name: "source equals destination",
			input: &types.MsgSubstationAllocationConnect{
				Creator:       player.Creator,
				AllocationId:  createdAllocation.Id,
				DestinationId: substation.Id,
			},
			expErr:    true,
			expErrMsg: "cannot match allocation source",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Recreate allocation if needed
			if tc.name == "valid allocation connection" {
				allocation.DestinationId = ""
				createdAllocation, _, _ = k.AppendAllocation(ctx, allocation, 100)
				tc.input.AllocationId = createdAllocation.Id
			} else if tc.name == "source equals destination" {
				// Set source to be the substation
				allocation.SourceObjectId = substation.Id
				createdAllocation, _, _ = k.AppendAllocation(ctx, allocation, 100)
				tc.input.AllocationId = createdAllocation.Id
			}

			resp, err := ms.SubstationAllocationConnect(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify allocation was connected
				updatedAllocation, found := k.GetAllocation(ctx, tc.input.AllocationId)
				require.True(t, found)
				require.Equal(t, tc.input.DestinationId, updatedAllocation.DestinationId)
			}
		})
	}
}
