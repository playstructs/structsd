package app_test

import (
	"fmt"
	"os"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

// Account for the pinned upstream SDK behavior, including its known overcharge.
// These checks continue exercising native hooks, shares, balances and invariants.
func TestUpstreamRedelegationSlashPendingPaymentAccounting(t *testing.T) {
	testRedelegationSlashPendingPayment(t, false)
}

// Opt-in failing reproduction for the upstream issue; ordinary suites still run
// the complete matrix above. Remove this distinction when upstream fixes it.
func TestReproduceSDKRedelegationSlashOvercharge(t *testing.T) {
	if os.Getenv("STRUCTS_REPRO_SDK_SLASH_OVERCHARGE") != "1" {
		t.Skip("set STRUCTS_REPRO_SDK_SLASH_OVERCHARGE=1 to assert the desired cap against upstream")
	}
	testRedelegationSlashPendingPayment(t, true)
}

func testRedelegationSlashPendingPayment(t *testing.T, requireCorrectedCap bool) {
	for _, fullSourceSlash := range []bool{false, true} {
		for _, pendingAmount := range []int64{0, 1, 30000, 100000, 150000} {
			for _, priorDestinationSlash := range []bool{false, true} {
				t.Run(fmt.Sprintf("source_infraction=%t/pending=%d/prior_destination_slash=%t", fullSourceSlash, pendingAmount, priorDestinationSlash), func(t *testing.T) {
					bApp, ctx := setupJailGateAppWithValidators(t, 2)
					validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
					require.NoError(t, err)
					source, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
					require.NoError(t, err)
					destination, err := sdk.ValAddressFromBech32(validators[1].OperatorAddress)
					require.NoError(t, err)
					denom, err := bApp.StakingKeeper.BondDenom(ctx)
					require.NoError(t, err)
					primary, _, _ := newTransferPlayer(t, bApp, ctx)
					server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
					delegate(t, bApp, ctx, server, denom, primary, source, math.NewInt(2000000))
					historicalSource, err := bApp.StakingKeeper.GetValidator(ctx, source)
					require.NoError(t, err)
					historicalPower := historicalSource.GetConsensusPower(bApp.StakingKeeper.PowerReduction(ctx))
					_, err = server.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(primary.String(), source.String(), destination.String(), sdk.NewInt64Coin(denom, 1000000)))
					require.NoError(t, err)
					if pendingAmount != 0 {
						_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(primary.String(), destination.String(), sdk.NewInt64Coin(denom, pendingAmount)))
						require.NoError(t, err)
					}

					delegator := primary
					factor := math.LegacyNewDecWithPrec(1, 1)
					if priorDestinationSlash {
						validator, err := bApp.StakingKeeper.GetValidator(ctx, destination)
						require.NoError(t, err)
						consensus, err := validator.GetConsAddr()
						require.NoError(t, err)
						// A current-height infraction reaches active stake, not the
						// pending withdrawals. This changes the real share price.
						_, err = bApp.StakingKeeper.Slash(ctx, consensus, ctx.BlockHeight(), validator.GetConsensusPower(bApp.StakingKeeper.PowerReduction(ctx)), factor)
						require.NoError(t, err)
					}
					row, err := bApp.StakingKeeper.GetRedelegation(ctx, delegator, source, destination)
					require.NoError(t, err)
					require.Len(t, row.Entries, 1)
					before, err := bApp.StakingKeeper.GetDelegation(ctx, delegator, destination)
					require.NoError(t, err)
					dst, err := bApp.StakingKeeper.GetValidator(ctx, destination)
					require.NoError(t, err)
					originalShareCap := factor.Mul(row.Entries[0].SharesDst)
					if originalShareCap.GT(before.Shares) {
						originalShareCap = before.Shares
					}
					_, originalTokenCap := dst.RemoveDelShares(originalShareCap)
					pendingPaid := min(pendingAmount, int64(100000))
					activeLimit := math.MinInt(math.NewInt(100000-pendingPaid), originalTokenCap)
					beforeSupply := bApp.BankKeeper.GetSupply(ctx, denom).Amount
					beforeLiquid := bApp.BankKeeper.GetAllBalances(ctx, delegator)
					src, err := bApp.StakingKeeper.GetValidator(ctx, source)
					require.NoError(t, err)
					sourceBurn := math.ZeroInt()
					if fullSourceSlash {
						// The historical source infraction must discover the native
						// redelegation through its index, collect the pending/active
						// destination penalty, and deduct only the remainder at source.
						consensus, err := src.GetConsAddr()
						require.NoError(t, err)
						ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
						sourceBurn, err = bApp.StakingKeeper.Slash(ctx, consensus, row.Entries[0].CreationHeight, historicalPower, factor)
						require.NoError(t, err)
						wantedTotal := factor.MulInt(bApp.StakingKeeper.TokensFromConsensusPower(ctx, historicalPower)).TruncateInt()
						require.Equal(t, wantedTotal.SubRaw(100000), sourceBurn)
						afterSource, err := bApp.StakingKeeper.GetValidator(ctx, source)
						require.NoError(t, err)
						require.Equal(t, src.Tokens.Sub(sourceBurn), afterSource.Tokens)
					} else {
						nominal, err := bApp.StakingKeeper.SlashRedelegation(ctx, src, row, row.Entries[0].CreationHeight, factor)
						require.NoError(t, err)
						require.Equal(t, int64(100000), nominal.Int64())
					}
					burned := beforeSupply.Sub(bApp.BankKeeper.GetSupply(ctx, denom).Amount).Sub(sourceBurn)
					activeBurn := burned.SubRaw(pendingPaid)
					if requireCorrectedCap {
						require.True(t, activeBurn.LTE(activeLimit), "SDK overcharge: pending=%d total burn=%s active limit=%s actual active burn=%s", pendingPaid, burned, activeLimit, activeBurn)
						// Corrected share truncation may leave one base unit unpaid.
						require.True(t, activeBurn.GTE(math.MaxInt(math.ZeroInt(), activeLimit.SubRaw(1))))
					} else {
						// Upstream skips active burning only when pending withdrawals
						// cover the entire penalty. Partial payment leaves the original
						// active-share burn unchanged; this is the documented defect.
						wantedActive := originalTokenCap
						if pendingPaid == 100000 {
							wantedActive = math.ZeroInt()
						}
						require.True(t, wantedActive.Equal(activeBurn), "upstream behavior changed: expected active burn %s, got %s; review the documented issue", wantedActive, activeBurn)
					}

					after, err := bApp.StakingKeeper.GetDelegation(ctx, delegator, destination)
					require.NoError(t, err)
					require.True(t, before.Shares.Sub(after.Shares).LTE(originalShareCap), "a lower share price must not increase the native share penalty")
					require.Equal(t, beforeLiquid, bApp.BankKeeper.GetAllBalances(ctx, delegator))
					if pendingAmount != 0 {
						ubd, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, delegator, destination)
						require.NoError(t, err)
						require.Equal(t, pendingAmount-pendingPaid, ubd.Entries[0].Balance.Int64())
					}
					requireStakingAndDistributionHealthy(t, bApp, ctx)
				})
			}
		}
	}
}
