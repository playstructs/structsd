package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgAllocationUpdate(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	playerAcc := sdk.AccAddress("creator123456789012345678901234567890")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	validatorAddress := sdk.ValAddress(playerAcc.Bytes())
	reactor := types.Reactor{
		Validator:  validatorAddress.String(),
		RawAddress: validatorAddress.Bytes(),
	}
	reactor = k.AppendReactor(ctx, reactor)

	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, reactor.Id)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(1000))

	sourceObjectPermissionId := keeperlib.GetObjectPermissionIDBytes(reactor.Id, player.Id)
	testPermissionAdd(k, ctx, sourceObjectPermissionId, types.PermAll)

	allocation := types.Allocation{
		SourceObjectId: reactor.Id,
		DestinationId:  "",
		Type:           types.AllocationType_dynamic,
		Controller:     player.Id,
	}
	createdAllocation, err := testAppendAllocation(k, ctx, allocation, 100)
	require.NoError(t, err)

	allocationPermissionId := keeperlib.GetObjectPermissionIDBytes(createdAllocation.Id, player.Id)
	testPermissionAdd(k, ctx, allocationPermissionId, types.PermAll)

	testCases := []struct {
		name      string
		input     *types.MsgAllocationUpdate
		expErr    bool
		expErrMsg string
		skip      bool
	}{
		{
			name: "valid allocation update",
			input: &types.MsgAllocationUpdate{
				Creator:      player.Creator,
				AllocationId: createdAllocation.Id,
				Power:        200,
			},
			expErr: false,
		},
		{
			name: "allocation not found",
			input: &types.MsgAllocationUpdate{
				Creator:      player.Creator,
				AllocationId: "invalid-allocation",
				Power:        200,
			},
			expErr:    true,
			expErrMsg: "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skip("Skipping test")
			}

			resp, err := ms.AllocationUpdate(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tc.input.AllocationId, resp.AllocationId)
			}
		})
	}
}
