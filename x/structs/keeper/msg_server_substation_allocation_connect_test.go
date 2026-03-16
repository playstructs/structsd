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
	player = testAppendPlayer(k, ctx, player)

	// Set up source capacity
	sourceObjectId := "source-object"
	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceObjectId)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(1000))

	// Create an allocation
	allocation := types.Allocation{
		SourceObjectId: sourceObjectId,
		DestinationId:  "", // Must be empty for connection
		Type:           types.AllocationType_dynamic,
		Controller: player.Id,
	}
	createdAllocation, err := testAppendAllocation(k, ctx, allocation, 100)
	require.NoError(t, err)

	// Create a substation
	substationAllocation := types.Allocation{
		SourceObjectId: sourceObjectId,
		DestinationId:  "",
		Type:           types.AllocationType_static,
		Controller: player.Id,
	}
	substationAlloc, err := testAppendAllocation(k, ctx, substationAllocation, 100)
	require.NoError(t, err)

	substation, _, err := testAppendSubstation(k, ctx, substationAlloc, player)
	require.NoError(t, err)

	// Grant permissions
	substationPermissionId := keeperlib.GetObjectPermissionIDBytes(substation.Id, player.Id)
	testPermissionAdd(k, ctx, substationPermissionId, types.PermSubstationConnection)

	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(player.Creator)
	testPermissionAdd(k, ctx, addressPermissionId, types.PermSubstationConnection)

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
			expErrMsg: "not found",
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
			expErrMsg: "source_destination_match",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Recreate allocation if needed
			if tc.name == "valid allocation connection" {
				allocation.DestinationId = ""
				createdAllocation, _ = testAppendAllocation(k, ctx, allocation, 100)
				tc.input.AllocationId = createdAllocation.Id
			} else if tc.name == "source equals destination" {
				// Set source to be the substation
				allocation.SourceObjectId = substation.Id
				createdAllocation, _ = testAppendAllocation(k, ctx, allocation, 100)
				tc.input.AllocationId = createdAllocation.Id
			}

			resp, err := ms.SubstationAllocationConnect(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				if tc.expErrMsg != "" {
					require.Contains(t, err.Error(), tc.expErrMsg)
				}
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
