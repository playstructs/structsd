package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgProviderUpdateDurationMaximum(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create a player first
	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = k.AppendPlayer(ctx, player)

	// Create a substation
	sourceObjectId := "source-object"
	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceObjectId)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(1000))

	allocation := types.Allocation{
		SourceObjectId: sourceObjectId,
		DestinationId:  "",
		Type:           types.AllocationType_static,
		Controller:     player.Creator,
	}
	createdAllocation, _, err := k.AppendAllocation(ctx, allocation, 100)
	require.NoError(t, err)

	substation, _, err := k.AppendSubstation(ctx, createdAllocation, player)
	require.NoError(t, err)

	// Create a provider
	provider := types.Provider{
		Owner:                       player.Id,
		Creator:                     player.Creator,
		SubstationId:                substation.Id,
		Rate:                        sdk.NewCoin("token", math.NewInt(100)),
		AccessPolicy:                types.ProviderAccessPolicy_openMarket,
		CapacityMinimum:             100,
		CapacityMaximum:             1000,
		DurationMinimum:             1,
		DurationMaximum:             10,
		ProviderCancellationPenalty: math.LegacyNewDec(1),
		ConsumerCancellationPenalty: math.LegacyNewDec(1),
	}
	provider, _ = k.AppendProvider(ctx, provider)

	testCases := []struct {
		name      string
		input     *types.MsgProviderUpdateDurationMaximum
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid duration maximum update",
			input: &types.MsgProviderUpdateDurationMaximum{
				Creator:            player.Creator,
				ProviderId:         provider.Id,
				NewMaximumDuration: 20,
			},
			expErr: false,
		},
		{
			name: "provider not found",
			input: &types.MsgProviderUpdateDurationMaximum{
				Creator:            player.Creator,
				ProviderId:         "invalid-provider",
				NewMaximumDuration: 20,
			},
			expErr:    true,
			expErrMsg: "not found",
		},
		{
			name: "no update permissions",
			input: &types.MsgProviderUpdateDurationMaximum{
				Creator:            "cosmos1noperms",
				ProviderId:         provider.Id,
				NewMaximumDuration: 20,
			},
			expErr:    true,
			expErrMsg: "has no",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Recreate provider if needed
			if tc.name == "valid duration maximum update" {
				provider, _ = k.AppendProvider(ctx, provider)
				tc.input.ProviderId = provider.Id
			}

			resp, err := ms.ProviderUpdateDurationMaximum(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify duration maximum was updated
				updatedProvider, found := k.GetProvider(ctx, provider.Id)
				require.True(t, found)
				require.Equal(t, tc.input.NewMaximumDuration, updatedProvider.DurationMaximum)
			}
		})
	}
}
