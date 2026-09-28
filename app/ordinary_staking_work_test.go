package app_test

import (
	"encoding/binary"
	"testing"
	"time"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	"structs/x/structs/types"
)

// Adds real native staking records to the ordinary workload fixture. The slash
// happens during setup; this measures its deferred repair alongside native
// refunds, maturity cleanup and the existing agreement/generator/cascade load.
type ordinaryStakingWork struct {
	reactor, denom  string
	active, refunds []sdk.AccAddress
	maturityRows    int
	supply          sdk.Coin
	emitted         map[string]types.Infusion
}

func setupOrdinaryStakingWork(t *testing.T, bApp *app.App, ctx sdk.Context) (*ordinaryStakingWork, sdk.Context) {
	t.Helper()
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	return setupOrdinaryStakingWorkForValidator(t, bApp, ctx, validator)
}

func setupOrdinaryStakingWorkForValidator(t *testing.T, bApp *app.App, ctx sdk.Context, validator sdk.ValAddress) (*ordinaryStakingWork, sdk.Context) {
	t.Helper()
	k := bApp.StructsKeeper
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	work := &ordinaryStakingWork{reactor: jailGateReactorId(t, bApp, ctx, validator), denom: denom, emitted: map[string]types.Infusion{}}
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	maturity := ctx.BlockTime()
	for group := 0; group < 2; group++ {
		for i := 0; i < types.InfusionMaturitySweepBudget+2; i++ {
			address := make([]byte, 20)
			address[0], address[1] = 0x64, byte(group)
			binary.BigEndian.PutUint64(address[12:], uint64(i+1))
			player := appendAppPlayer(t, k, ctx, sdk.AccAddress(address))
			vaults, err := k.NewCurrentContext(ctx).EnsureVaults(player.Id)
			require.NoError(t, err)
			vault := vaults[0]
			amount := int64(2000000)
			if group == 1 {
				amount = 1000000
			}
			delegate(t, bApp, ctx, server, denom, vault, validator, math.NewInt(amount))
			if group == 0 {
				work.active = append(work.active, vault)
				continue
			}
			withdrawal, err := server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(vault.String(), validator.String(), sdk.NewInt64Coin(denom, amount)))
			require.NoError(t, err)
			if withdrawal.CompletionTime.After(maturity) {
				maturity = withdrawal.CompletionTime
			}
			work.refunds = append(work.refunds, vault)
			require.True(t, bApp.BankKeeper.GetBalance(ctx, vault, denom).IsZero())
		}
	}
	// A current-height slash only reduces active stake. Already unbonding
	// withdrawals retain their full refund, so the payment oracle is independent
	// of the deferred infusion repair and its ordering.
	val, err := bApp.StakingKeeper.GetValidator(ctx, validator)
	require.NoError(t, err)
	consensus, err := val.GetConsAddr()
	require.NoError(t, err)
	_, err = bApp.StakingKeeper.Slash(ctx, consensus, ctx.BlockHeight(), val.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx)), math.LegacyNewDecWithPrec(1, 1))
	require.NoError(t, err)
	require.NotEmpty(t, k.GetReactorSlashReconcileQueue(ctx))
	for _, address := range work.active {
		row, found := k.GetInfusion(ctx, work.reactor, address.String())
		require.True(t, found)
		require.Equal(t, uint64(2000000), row.Fuel, "slash hook must leave work for the bounded repair")
	}
	work.maturityRows = len(k.GetInfusionMaturitySweepQueueExport(ctx))
	require.Greater(t, work.maturityRows, types.InfusionMaturitySweepBudget)
	work.supply = bApp.BankKeeper.GetSupply(ctx, denom)
	return work, ctx.WithBlockTime(maturity.Add(time.Second))
}

func (work *ordinaryStakingWork) verify(t *testing.T, bApp *app.App, ctx sdk.Context, pass int, events []abci.Event) bool {
	t.Helper()
	k := bApp.StructsKeeper
	credits := map[string][]string{}
	for _, event := range events {
		if event.Type == "structs.structs.EventInfusion" {
			decoded, err := sdk.ParseTypedEvent(event)
			require.NoError(t, err)
			row := decoded.(*types.EventInfusion).Infusion
			if row.DestinationId == work.reactor {
				work.emitted[row.Address] = *row
			}
		}
		if event.Type != banktypes.EventTypeCoinReceived {
			continue
		}
		var receiver, amount string
		for _, attribute := range event.Attributes {
			if attribute.Key == banktypes.AttributeKeyReceiver {
				receiver = attribute.Value
			}
			if attribute.Key == sdk.AttributeKeyAmount {
				amount = attribute.Value
			}
		}
		credits[receiver] = append(credits[receiver], amount)
	}
	remaining := 0
	for _, address := range work.refunds {
		require.True(t, sdk.NewInt64Coin(work.denom, 1000000).IsEqual(bApp.BankKeeper.GetBalance(ctx, address, work.denom)), "native refunds may not wait for Structs bookkeeping")
		if pass == 0 {
			require.Equal(t, []string{sdk.NewInt64Coin(work.denom, 1000000).String()}, credits[address.String()], "one exact public credit per native refund")
		} else {
			require.Empty(t, credits[address.String()], "later bookkeeping must not announce another refund")
		}
		if _, found := k.GetInfusion(ctx, work.reactor, address.String()); found {
			remaining++
		}
	}
	maturity := len(k.GetInfusionMaturitySweepQueueExport(ctx))
	destruction := len(k.GetInfusionDestructionQueueExport(ctx))
	slashes := len(k.GetReactorSlashReconcileQueue(ctx))
	if pass == 0 {
		require.GreaterOrEqual(t, maturity, work.maturityRows-types.InfusionMaturitySweepBudget, "raw maturity work remains bounded")
		require.Greater(t, remaining, 0, "destruction cannot drain more than its budget")
		require.Greater(t, slashes, 0, "slash repair must span blocks")
	}
	require.True(t, work.supply.IsEqual(bApp.BankKeeper.GetSupply(ctx, work.denom)), "refunds and repairs must not mint or burn")
	done := maturity == 0 && destruction == 0 && slashes == 0
	if done {
		require.Zero(t, remaining)
		for _, address := range work.active {
			row, found := k.GetInfusion(ctx, work.reactor, address.String())
			require.True(t, found)
			require.Equal(t, uint64(1800000), row.Fuel)
			require.Zero(t, row.Defusing)
			requireInfusionEventEqual(t, row, work.emitted[address.String()], "last public infusion event must match repaired state")
			owner, found := k.GetVaultOwner(ctx, address)
			require.True(t, found)
			_, _, power := row.GetPowerDistribution()
			require.Equal(t, power, k.NewCurrentContext(ctx).GetPlayer(owner).GetCapacity(), "other block phases must not overwrite slash capacity changes")
		}
		if pass < 5 {
			requireStakingAndDistributionHealthy(t, bApp, ctx)
		}
	}
	if pass < 5 {
		t.Logf("staking pass=%d maturity_rows=%d destruction_rows=%d reactors_pending=%d refund_infusions_remaining=%d", pass, maturity, destruction, slashes, remaining)
	}
	return done
}
