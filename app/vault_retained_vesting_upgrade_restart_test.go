package app_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

// Run the assembled production migration in x/upgrade's block, then import both
// export modes. Native vesting, self-stake, another validator's stake and pending
// records must remain coupled while another address of the same player migrates.
func TestRetainedNativeVestingUpgradeBlockAndRestart(t *testing.T) {
	for _, kind := range []string{"continuous", "delayed", "periodic", "permanent"} {
		for _, zeroHeight := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/zero=%t", kind, zeroHeight), func(t *testing.T) {
				bApp, ctx := setupJailGateApp(t)
				denom, err := bApp.StakingKeeper.BondDenom(ctx)
				require.NoError(t, err)
				coins := func(n int64) sdk.Coins { return sdk.NewCoins(sdk.NewInt64Coin(denom, n)) }
				validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
				require.NoError(t, err)
				other, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
				require.NoError(t, err)
				operatorKey := secp256k1.GenPrivKey()
				operator := sdk.AccAddress(operatorKey.PubKey().Address())
				self := sdk.ValAddress(operator)
				fundRedelegationAccount(t, bApp, ctx, operator, coins(20000000))
				base := bApp.AccountKeeper.GetAccount(ctx, operator).(*authtypes.BaseAccount)
				start, end := ctx.BlockTime().Unix(), ctx.BlockTime().Add(365*24*time.Hour).Unix()
				var vesting vestingexported.VestingAccount
				switch kind {
				case "continuous":
					vesting, err = vestingtypes.NewContinuousVestingAccount(base, coins(20000000), start, end)
				case "delayed":
					vesting, err = vestingtypes.NewDelayedVestingAccount(base, coins(20000000), end)
				case "periodic":
					vesting, err = vestingtypes.NewPeriodicVestingAccount(base, coins(20000000), start, vestingtypes.Periods{{Length: end - start, Amount: coins(20000000)}})
				case "permanent":
					vesting, err = vestingtypes.NewPermanentLockedAccount(base, coins(20000000))
				}
				require.NoError(t, err)
				bApp.AccountKeeper.SetAccount(ctx, vesting)
				server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
				msg, err := stakingtypes.NewMsgCreateValidator(self.String(), ed25519.GenPrivKey().PubKey(), coins(10000000)[0], stakingtypes.NewDescription("retained-restart", "", "", "", ""), stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(1, 1), math.LegacyNewDecWithPrec(2, 1), math.LegacyNewDecWithPrec(1, 2)), math.OneInt())
				require.NoError(t, err)
				_, err = server.CreateValidator(ctx, msg)
				require.NoError(t, err)
				_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
				require.NoError(t, err)
				delegate(t, bApp, ctx, server, denom, operator, other, math.NewInt(2000000))
				_, err = server.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(operator.String(), other.String(), self.String(), coins(1000000)[0]))
				require.NoError(t, err)
				_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(operator.String(), self.String(), coins(500000)[0]))
				require.NoError(t, err)
				index := bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, operator.String())
				playerID := fmt.Sprintf("1-%d", index)
				secondaryKey := secp256k1.GenPrivKey()
				secondary := sdk.AccAddress(secondaryKey.PubKey().Address())
				require.NoError(t, bApp.StructsKeeper.SetPlayerIndexForAddress(ctx, secondary.String(), index))
				delegate(t, bApp, ctx, server, denom, secondary, other, math.NewInt(3000000))
				beforeSecondary, err := bApp.StakingKeeper.GetDelegation(ctx, secondary, other)
				require.NoError(t, err)
				fundRedelegationAccount(t, bApp, ctx, secondary, coins(123))
				withdraw := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
				require.NoError(t, bApp.DistrKeeper.SetDelegatorWithdrawAddr(ctx, operator, withdraw))
				vault, err := types.VaultAddress(playerID)
				require.NoError(t, err)
				beforeAccount := bApp.AccountKeeper.GetAccount(ctx, operator)
				beforeBalances := bApp.BankKeeper.GetAllBalances(ctx, operator)
				beforeSelf, err := bApp.StakingKeeper.GetDelegation(ctx, operator, self)
				require.NoError(t, err)
				beforeOther, err := bApp.StakingKeeper.GetDelegation(ctx, operator, other)
				require.NoError(t, err)
				ubd, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, operator, self)
				require.NoError(t, err)
				red, err := bApp.StakingKeeper.GetRedelegation(ctx, operator, other, self)
				require.NoError(t, err)
				for _, id := range []uint64{ubd.Entries[0].UnbondingId, red.Entries[0].UnbondingId} {
					require.NoError(t, bApp.StakingKeeper.PutUnbondingOnHold(ctx, id))
				}
				ubd.Entries[0].UnbondingOnHoldRefCount = 1
				red.Entries[0].UnbondingOnHoldRefCount = 1
				const name = "test-retained-vesting-upgrade"
				require.NoError(t, bApp.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{Name: name, Height: 2}))
				_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
				require.NoError(t, err)
				_, err = bApp.Commit()
				require.NoError(t, err)
				bApp.UpgradeKeeper.SetUpgradeHandler(name, func(goCtx context.Context, _ upgradetypes.Plan, versions module.VersionMap) (module.VersionMap, error) {
					restoreLegacyCascadeQueueFixture(t, bApp, sdk.UnwrapSDKContext(goCtx))
					return v22.CreateUpgradeHandler(bApp.ModuleManager, bApp.Configurator(), delegationUpgradeKeepers(bApp))(goCtx, upgradetypes.Plan{Name: v22.UpgradeName}, versions)
				})
				upgradeTime := ctx.BlockTime().Add(time.Second)
				block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: upgradeTime})
				require.NoError(t, err)
				require.Empty(t, block.TxResults)
				current := bApp.NewContext(false).WithBlockTime(upgradeTime)
				require.Equal(t, beforeAccount, bApp.AccountKeeper.GetAccount(current, operator))
				require.Equal(t, beforeBalances, bApp.BankKeeper.GetAllBalances(current, operator))

				association, movement := -1, -1
				for i, event := range block.Events {
					if event.Type == "structs.structs.EventVaultAddress" {
						parsed, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						if parsed.(*types.EventVaultAddress).EventVaultAddressDetail.OwnerId == playerID {
							association = i
						}
					}
					if event.Type == "transfer" {
						attrs := map[string]string{}
						for _, attr := range event.Attributes {
							attrs[attr.Key] = attr.Value
						}
						require.NotEqual(t, operator.String(), attrs["sender"])
						if attrs["sender"] == secondary.String() {
							require.Equal(t, vault.String(), attrs["recipient"])
							require.Equal(t, "123"+denom, attrs["amount"])
							movement = i
						}
					}
				}
				require.NotEqual(t, -1, association)
				require.Greater(t, movement, association)
				_, err = bApp.Commit()
				require.NoError(t, err)
				exported, err := bApp.ExportAppStateAndValidators(zeroHeight, nil, nil)
				require.NoError(t, err)
				restarted, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}, baseapp.SetChainID(jailGateChainID))
				require.NoError(t, err)
				restartTime := upgradeTime.Add(time.Second)
				_, err = restarted.InitChain(&abci.RequestInitChain{AppStateBytes: exported.AppState, ChainId: jailGateChainID, InitialHeight: exported.Height, Time: restartTime, ConsensusParams: &exported.ConsensusParams})
				require.NoError(t, err)
				height := exported.Height
				if height == 0 {
					height = 1
				}
				restored := restarted.NewContextLegacy(false, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: restartTime})
				require.Equal(t, beforeAccount, restarted.AccountKeeper.GetAccount(restored, operator))
				require.Equal(t, beforeBalances, restarted.BankKeeper.GetAllBalances(restored, operator))
				require.Equal(t, index, restarted.StructsKeeper.GetPlayerIndexFromAddress(restored, operator.String()))
				require.Equal(t, beforeAccount.(vestingexported.VestingAccount).LockedCoins(restartTime), restarted.AccountKeeper.GetAccount(restored, operator).(vestingexported.VestingAccount).LockedCoins(restartTime))
				for _, pair := range []struct {
					validator sdk.ValAddress
					expected  stakingtypes.Delegation
				}{{self, beforeSelf}, {other, beforeOther}} {
					actual, err := restarted.StakingKeeper.GetDelegation(restored, operator, pair.validator)
					require.NoError(t, err)
					require.Equal(t, pair.expected, actual)
				}
				if zeroHeight {
					ubd.Entries[0].CreationHeight = 0
					red.Entries[0].CreationHeight = 0
				}
				restoredUBD, err := restarted.StakingKeeper.GetUnbondingDelegationByUnbondingID(restored, ubd.Entries[0].UnbondingId)
				require.NoError(t, err)
				require.Equal(t, ubd, restoredUBD)
				restoredRed, err := restarted.StakingKeeper.GetRedelegationByUnbondingID(restored, red.Entries[0].UnbondingId)
				require.NoError(t, err)
				require.Equal(t, red, restoredRed)
				queue, err := restarted.StakingKeeper.GetUBDQueueTimeSlice(restored, ubd.Entries[0].CompletionTime)
				require.NoError(t, err)
				require.Contains(t, queue, stakingtypes.DVPair{DelegatorAddress: operator.String(), ValidatorAddress: self.String()})
				redQueue, err := restarted.StakingKeeper.GetRedelegationQueueTimeSlice(restored, red.Entries[0].CompletionTime)
				require.NoError(t, err)
				require.Contains(t, redQueue, stakingtypes.DVVTriplet{DelegatorAddress: operator.String(), ValidatorSrcAddress: other.String(), ValidatorDstAddress: self.String()})
				_, err = restarted.StakingKeeper.GetDelegation(restored, vault, other)
				require.ErrorIs(t, err, stakingtypes.ErrNoDelegation)
				moved, err := restarted.StakingKeeper.GetDelegation(restored, secondary, other)
				require.NoError(t, err)
				require.Equal(t, beforeSecondary.Shares, moved.Shares)
				withdrawAfter, err := restarted.DistrKeeper.GetDelegatorWithdrawAddr(restored, operator)
				require.NoError(t, err)
				require.Equal(t, withdraw, withdrawAfter)
				_, err = restarted.DistrKeeper.WithdrawDelegationRewards(restored, operator, other)
				require.NoError(t, err)
				preview, _ := restored.CacheContext()
				validator, err := restarted.StakingKeeper.GetValidator(preview, other)
				require.NoError(t, err)
				supply := restarted.BankKeeper.GetSupply(preview, denom).Amount
				slashed, err := restarted.StakingKeeper.SlashRedelegation(preview, validator, restoredRed, restoredRed.Entries[0].CreationHeight, math.LegacyNewDecWithPrec(1, 1))
				require.NoError(t, err)
				require.Equal(t, math.NewInt(100000), slashed)
				pending, err := restarted.StakingKeeper.GetUnbondingDelegationByUnbondingID(preview, ubd.Entries[0].UnbondingId)
				require.NoError(t, err)
				require.Equal(t, ubd.Entries[0].Balance.Sub(slashed), pending.Entries[0].Balance)
				require.Equal(t, slashed, supply.Sub(restarted.BankKeeper.GetSupply(preview, denom).Amount))
				requireStakingAndDistributionHealthy(t, restarted, preview)
				// The SDK dispatches hold release by the operation type index; a
				// surviving row/time queue alone does not make it reachable.
				for _, id := range []uint64{ubd.Entries[0].UnbondingId, red.Entries[0].UnbondingId} {
					require.NoError(t, restarted.StakingKeeper.UnbondingCanComplete(restored, id))
				}
				releasedUBD, err := restarted.StakingKeeper.GetUnbondingDelegationByUnbondingID(restored, ubd.Entries[0].UnbondingId)
				require.NoError(t, err)
				require.Zero(t, releasedUBD.Entries[0].UnbondingOnHoldRefCount)
				releasedRed, err := restarted.StakingKeeper.GetRedelegationByUnbondingID(restored, red.Entries[0].UnbondingId)
				require.NoError(t, err)
				require.Zero(t, releasedRed.Entries[0].UnbondingOnHoldRefCount)
				// A real new operation must receive a fresh ID and leave both
				// imported lookups pointing at their original records.
				_, err = stakingkeeper.NewMsgServerImpl(restarted.StakingKeeper).Undelegate(restored, stakingtypes.NewMsgUndelegate(operator.String(), self.String(), coins(100000)[0]))
				require.NoError(t, err)
				withNew, err := restarted.StakingKeeper.GetUnbondingDelegationByUnbondingID(restored, ubd.Entries[0].UnbondingId)
				require.NoError(t, err)
				require.Len(t, withNew.Entries, 2)
				require.Greater(t, withNew.Entries[1].UnbondingId, ubd.Entries[0].UnbondingId)
				require.Greater(t, withNew.Entries[1].UnbondingId, red.Entries[0].UnbondingId)
				unchangedRed, err := restarted.StakingKeeper.GetRedelegationByUnbondingID(restored, red.Entries[0].UnbondingId)
				require.NoError(t, err)
				require.Equal(t, releasedRed, unchangedRed)
				_, err = restarted.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: restartTime})
				require.NoError(t, err)
				_, err = restarted.Commit()
				require.NoError(t, err)
				verifySignedRetainedVestingIdentity(t, restarted, operatorKey, secondaryKey, playerID, self, other, height+1, restartTime.Add(time.Second))
			})
		}
	}
}
