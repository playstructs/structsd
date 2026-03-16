package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgAllocationCreate(t *testing.T) {
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

	testCases := []struct {
		name      string
		input     *types.MsgAllocationCreate
		expErr    bool
		expErrMsg string
		skip      bool
	}{
		{
			name: "valid allocation creation",
			input: &types.MsgAllocationCreate{
				Creator:        player.Creator,
				AllocationType: types.AllocationType_static,
				SourceObjectId: reactor.Id,
				Power:          100,
			},
			expErr: false,
		},
		{
			name: "invalid source object",
			input: &types.MsgAllocationCreate{
				Creator:        player.Creator,
				AllocationType: types.AllocationType_static,
				SourceObjectId: "bad-source",
				Power:          100,
			},
			expErr:    true,
			expErrMsg: "unacceptable_source",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skip("Skipping test")
			}

			resp, err := ms.AllocationCreate(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotEmpty(t, resp.AllocationId)

				allocation, found := k.GetAllocation(ctx, resp.AllocationId)
				require.True(t, found)
				require.Equal(t, tc.input.SourceObjectId, allocation.SourceObjectId)
				require.Equal(t, tc.input.AllocationType, allocation.Type)
			}
		})
	}
}
