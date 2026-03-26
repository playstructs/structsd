package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgSubstationCreate(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	wctx := sdk.WrapSDKContext(sdkCtx)

	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = testAppendPlayer(k, sdkCtx, player)

	reactor := testAppendReactor(k, sdkCtx, types.Reactor{Validator: "cosmosvaloper1test"})
	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, reactor.Id)
	k.SetGridAttribute(sdkCtx, capacityAttrId, uint64(10000))

	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(player.Creator)
	testPermissionAdd(k, sdkCtx, addressPermissionId, types.PermAll)

	allocation := types.Allocation{
		SourceObjectId: reactor.Id,
		DestinationId:  "",
		Type:           types.AllocationType_static,
		Controller:     player.Id,
	}
	createdAllocation, err := testAppendAllocation(k, sdkCtx, allocation, 100)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		input     *types.MsgSubstationCreate
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid substation creation",
			input: &types.MsgSubstationCreate{
				Creator:      player.Creator,
				AllocationId: createdAllocation.Id,
			},
			expErr: false,
		},
		{
			name: "allocation not found",
			input: &types.MsgSubstationCreate{
				Creator:      player.Creator,
				AllocationId: "invalid-allocation",
			},
			expErr:    true,
			expErrMsg: "not found",
		},
		{
			name: "no energy management permissions",
			input: &types.MsgSubstationCreate{
				Creator:      "cosmos1noperms",
				AllocationId: createdAllocation.Id,
			},
			expErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.SubstationCreate(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				if tc.expErrMsg != "" {
					require.Contains(t, err.Error(), tc.expErrMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotEmpty(t, resp.SubstationId)

				substation, found := k.GetSubstation(sdkCtx, resp.SubstationId)
				require.True(t, found)
				require.Equal(t, player.Id, substation.Owner)
			}
		})
	}
}
