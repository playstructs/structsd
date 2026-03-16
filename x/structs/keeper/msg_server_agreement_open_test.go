package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgAgreementOpen(t *testing.T) {
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

	reactorCapacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, reactor.Id)
	k.SetGridAttribute(ctx, reactorCapacityAttrId, uint64(10000))

	reactorPermissionId := keeperlib.GetObjectPermissionIDBytes(reactor.Id, player.Id)
	testPermissionAdd(k, ctx, reactorPermissionId, types.PermAll)

	allocation := types.Allocation{
		SourceObjectId: reactor.Id,
		DestinationId:  "",
		Type:           types.AllocationType_static,
		Controller:     player.Id,
	}
	createdAllocation, err := testAppendAllocation(k, ctx, allocation, 5000)
	require.NoError(t, err)

	substation, _, err := testAppendSubstation(k, ctx, createdAllocation, player)
	require.NoError(t, err)

	createdAllocation.DestinationId = substation.Id
	k.ImportAllocation(ctx, createdAllocation)

	substationCapacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, substation.Id)
	k.SetGridAttribute(ctx, substationCapacityAttrId, uint64(5000))

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
	provider = testAppendProvider(k, ctx, provider)

	providerPermissionId := keeperlib.GetObjectPermissionIDBytes(provider.Id, player.Id)
	testPermissionAdd(k, ctx, providerPermissionId, types.PermAll)

	collateralAmount := math.NewInt(100).Mul(math.NewInt(5)).Mul(math.NewInt(100))
	collateralCoin := sdk.NewCoin("token", collateralAmount)
	k.BankKeeper().MintCoins(ctx, types.ModuleName, sdk.NewCoins(collateralCoin))
	k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, playerAcc, sdk.NewCoins(collateralCoin))

	testCases := []struct {
		name      string
		input     *types.MsgAgreementOpen
		expErr    bool
		expErrMsg string
		skip      bool
	}{
		{
			name: "valid agreement open",
			input: &types.MsgAgreementOpen{
				Creator:    player.Creator,
				ProviderId: provider.Id,
				Duration:   5,
				Capacity:   100,
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skip("Skipping test")
			}

			resp, err := ms.AgreementOpen(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}
		})
	}
}
