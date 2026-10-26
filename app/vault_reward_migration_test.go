package app_test

import (
	"bytes"
	"errors"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

func TestVaultMigrationRewardTransfersAndDustMatchSDKWithdrawal(t *testing.T) {
	for mode, name := range []string{"owned withdrawal address", "unassociated withdrawal address", "earlier player recipient", "later player recipient"} {
		t.Run(name, func(t *testing.T) { testVaultRewardMigration(t, mode) })
	}
}

func testVaultRewardMigration(t *testing.T, mode int) {
	t.Helper()
	bApp, ctx := setupJailGateApp(t)
	externalWithdrawal, associatedWithdrawal := mode != 0, mode >= 2
	var recipientAddress sdk.AccAddress
	var recipientID string
	primary, secondary, playerID := newTransferPlayer(t, bApp, ctx)
	if associatedWithdrawal {
		// The migration sorts source addresses, not player IDs. Choose a real
		// address deterministically on the required side of the primary key.
		for i := 0; i < 256; i++ {
			candidate := sdk.AccAddress(bytes.Repeat([]byte{byte(i)}, 20))
			ordered := (mode == 2 && candidate.String() < primary.String()) || (mode == 3 && candidate.String() > primary.String())
			if ordered && bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, candidate.String()) == 0 {
				recipientAddress = candidate
				recipientID = appendAppPlayer(t, bApp.StructsKeeper, ctx, recipientAddress).Id
				break
			}
		}
		require.NotEmpty(t, recipientID)
	}
	validators, err := bApp.StakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	delegate(t, bApp, ctx, server, denom, primary, validator, math.NewInt(3000000))
	delegate(t, bApp, ctx, server, denom, secondary, validator, math.NewInt(4000000))
	withdrawalAddress := primary
	if externalWithdrawal {
		withdrawalAddress = sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
		if associatedWithdrawal { withdrawalAddress = recipientAddress }
		fundRedelegationAccount(t, bApp, ctx, withdrawalAddress, sdk.NewCoins(sdk.NewInt64Coin("rewardtest", 17)))
		if !associatedWithdrawal { require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, withdrawalAddress.String())) }
	}
	require.NoError(t, bApp.DistrKeeper.SetDelegatorWithdrawAddr(ctx, secondary, withdrawalAddress))
	externalBefore := bApp.BankKeeper.GetAllBalances(ctx, withdrawalAddress)
	rewards := sdk.NewCoins(sdk.NewInt64Coin(denom, 1001), sdk.NewInt64Coin("rewardtest", 701))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, types.ModuleName, rewards))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, distrtypes.ModuleName, rewards))
	val, err := bApp.StakingKeeper.GetValidator(ctx, validator)
	require.NoError(t, err)
	require.NoError(t, bApp.DistrKeeper.AllocateTokensToValidator(ctx, val, sdk.NewDecCoinsFromCoins(rewards...)))
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithEventManager(sdk.NewEventManager())
	poolBefore, err := bApp.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	// Use actual SDK withdrawals on a discarded branch as the accounting oracle,
	// including fractional rewards that are not transferable bank coins.
	preview, _ := ctx.CacheContext()
	keepers := delegationUpgradeKeepers(bApp)
	plan, err := v22.BuildVaultAddressMigrationPlan(ctx, keepers)
	require.NoError(t, err)
	if associatedWithdrawal {
		primaryIndex, recipientIndex := -1, -1
		for i, move := range plan {
			if move.Source == primary.String() { primaryIndex = i }
			if move.Source == recipientAddress.String() { recipientIndex = i }
		}
		require.NotEqual(t, -1, primaryIndex)
		require.NotEqual(t, -1, recipientIndex)
		if mode == 2 { require.Less(t, recipientIndex, primaryIndex) } else { require.Greater(t, recipientIndex, primaryIndex) }
	}
	// The fixture also has a genesis delegation. Include every migrating
	// source when comparing the shared community pool, not just this player.
	for _, move := range plan {
		address, err := sdk.AccAddressFromBech32(move.Source)
		require.NoError(t, err)
		_, err = bApp.StakingKeeper.GetDelegation(preview, address, validator)
		if errors.Is(err, stakingtypes.ErrNoDelegation) {
			continue
		}
		require.NoError(t, err)
		_, err = bApp.DistrKeeper.WithdrawDelegationRewards(preview, address, validator)
		require.NoError(t, err)
	}
	expected := bApp.BankKeeper.GetAllBalances(preview, primary)
	expectedExternal := bApp.BankKeeper.GetAllBalances(preview, withdrawalAddress)
	require.Positive(t, expected.AmountOf("rewardtest").Sign())
	poolExpected, err := bApp.DistrKeeper.FeePool.Get(preview)
	require.NoError(t, err)
	require.False(t, poolExpected.CommunityPool.Equal(poolBefore.CommunityPool), "fixture must exercise fractional dust")
	require.NoError(t, v22.MigrateVaultAssets(ctx, keepers))
	vault, err := types.VaultAddress(playerID)
	require.NoError(t, err)
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, vault).IsZero(), "upgrade must not withdraw rewards")
	poolAfter, err := bApp.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, poolBefore, poolAfter, "upgrade must not change reward dust")
	for _, event := range ctx.EventManager().Events() {
		if event.Type != banktypes.EventTypeTransfer { continue }
		for _, attribute := range event.Attributes {
			if attribute.Key == sdk.AttributeKeySender {
				require.NotEqual(t, authtypes.NewModuleAddress(distrtypes.ModuleName).String(), attribute.Value)
			}
		}
	}
	// The same ordinary SDK withdrawals remain available after upgrade.
	for _, move := range plan {
		address, err := sdk.AccAddressFromBech32(move.Source)
		require.NoError(t, err)
		_, err = bApp.StakingKeeper.GetDelegation(ctx, address, validator)
		if errors.Is(err, stakingtypes.ErrNoDelegation) { continue }
		require.NoError(t, err)
		_, err = bApp.DistrKeeper.WithdrawDelegationRewards(ctx, address, validator)
		require.NoError(t, err)
	}
	require.Equal(t, expected, bApp.BankKeeper.GetAllBalances(ctx, primary))
	if associatedWithdrawal {
		recipientVault, err := types.VaultAddress(recipientID)
		require.NoError(t, err)
		require.Equal(t, externalBefore, bApp.BankKeeper.GetAllBalances(ctx, recipientVault))
		expectedExternal = expectedExternal.Sub(externalBefore...)
	}
	require.Equal(t, expectedExternal, bApp.BankKeeper.GetAllBalances(ctx, withdrawalAddress))
	storedWithdrawal, err := bApp.DistrKeeper.GetDelegatorWithdrawAddr(ctx, secondary)
	require.NoError(t, err)
	require.Equal(t, withdrawalAddress, storedWithdrawal)
	poolAfter, err = bApp.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, poolExpected, poolAfter)
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
