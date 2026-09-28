package app_test

import (
	"encoding/binary"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestDiversityReadinessGasAndCrossOwnerIsolation(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 4)
	k := bApp.StructsKeeper
	_, _, largeOwner := newTransferPlayer(t, bApp, ctx)
	smallAddress, _, smallOwner := newTransferPlayer(t, bApp, ctx)
	large, found := k.GetPlayer(ctx, largeOwner)
	require.True(t, found)
	for _, owner := range []string{largeOwner, smallOwner} {
		k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, owner), 0)
	}
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	for i := 0; i <= keeperlib.OwnerSupplyReadBudget/2; i++ {
		address := smallAddress
		if i < keeperlib.OwnerSupplyReadBudget/2 {
			bytes := make([]byte, 20)
			bytes[0] = 0x7d
			binary.BigEndian.PutUint64(bytes[12:], uint64(i+1))
			address = sdk.AccAddress(bytes)
			require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), large.Index))
		}
		for _, validator := range validators[:2] {
			val, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
			require.NoError(t, err)
			delegate(t, bApp, ctx, server, denom, address, val, math.NewInt(100))
		}
	}
	require.NoError(t, k.CheckReactorDiversityAccounting(ctx))
	largeCapacity := k.GetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, largeOwner))
	smallCapacity := k.GetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, smallOwner))
	metered := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	cc := k.NewCurrentContext(metered)
	for i := 0; i < 2; i++ {
		require.Equal(t, largeCapacity, cc.GetPlayer(largeOwner).GetAvailableCapacity())
		require.Equal(t, smallCapacity, cc.GetPlayer(smallOwner).GetAvailableCapacity(), "the large owner's reads cannot exhaust another owner's allowance")
	}
	gas := metered.GasMeter().GasConsumed()
	require.Less(t, gas, uint64(20_000_000), "two repeated attacker/defender readiness pairs must fit the free gas ceiling")
	t.Logf("two_readiness_pairs_2048_infusions_gas=%d", gas)
}
