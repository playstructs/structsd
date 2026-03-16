package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgSubstationDelete(t *testing.T) {
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

	allocation := types.Allocation{
		SourceObjectId: reactor.Id,
		DestinationId:  "",
		Type:           types.AllocationType_static,
		Controller:     player.Id,
	}
	createdAllocation, err := testAppendAllocation(k, sdkCtx, allocation, 100)
	require.NoError(t, err)

	substation, _, err := testAppendSubstation(k, sdkCtx, createdAllocation, player)
	require.NoError(t, err)

	substationPermissionId := keeperlib.GetObjectPermissionIDBytes(substation.Id, player.Id)
	testPermissionAdd(k, sdkCtx, substationPermissionId, types.PermAll)

	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(player.Creator)
	testPermissionAdd(k, sdkCtx, addressPermissionId, types.PermAll)

	testCases := []struct {
		name      string
		input     *types.MsgSubstationDelete
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid substation deletion",
			input: &types.MsgSubstationDelete{
				Creator:               player.Creator,
				SubstationId:          substation.Id,
				MigrationSubstationId: "",
			},
			expErr: false,
		},
		{
			name: "no delete permissions",
			input: &types.MsgSubstationDelete{
				Creator:               "cosmos1noperms",
				SubstationId:          substation.Id,
				MigrationSubstationId: "",
			},
			expErr: true,
		},
		{
			name: "no energy management permissions",
			input: &types.MsgSubstationDelete{
				Creator:               player.Creator,
				SubstationId:          substation.Id,
				MigrationSubstationId: "",
			},
			expErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "valid substation deletion" {
				substation, _, _ = testAppendSubstation(k, sdkCtx, createdAllocation, player)
				testPermissionAdd(k, sdkCtx, keeperlib.GetObjectPermissionIDBytes(substation.Id, player.Id), types.PermAll)
				tc.input.SubstationId = substation.Id
			} else if tc.name == "no energy management permissions" {
				k.PermissionClearAll(sdkCtx, addressPermissionId)
			}

			resp, err := ms.SubstationDelete(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				if tc.expErrMsg != "" {
					require.Contains(t, err.Error(), tc.expErrMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}
		})
	}
}
