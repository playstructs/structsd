package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Cosmos has no operation for handing a delegation to another account, so
// MoveDelegationsToAddress assembles one. These tests pin the two things that
// assembly can get wrong and that no compiler will catch: the destination's
// existing shares have to be merged rather than replaced, and distribution's
// starting-info lifecycle has to be driven by hand in an order fixed by
// initializeDelegation.
//
// The mock is faithful about exactly that much. It refuses
// BeforeDelegationSharesModified without starting info, as the real hook does
// on ErrEmptyDelegationDistInfo, and refuses AfterDelegationModified when
// staking has no delegation to read, as initializeDelegation does. What it does
// not model is rewards, which is why app/delegation_transfer_test.go drives the
// same paths against real distribution.

type delegationTransferFixture struct {
	t         *testing.T
	k         keeperlib.Keeper
	sdkCtx    sdk.Context
	mock      *keepertest.MockStakingKeeper
	distr     *keepertest.MockDistributionKeeper
	player    types.Player
	valAddr   sdk.ValAddress
	fromAcc   sdk.AccAddress
	toAcc     sdk.AccAddress
	reactorId string
}

// setupDelegationTransfer seeds a player whose primary address is toAcc and a delegation
// of sourceStake held by fromAcc, both addresses registered to that player.
func setupDelegationTransfer(t *testing.T, seed string, sourceStake int64) *delegationTransferFixture {
	t.Helper()

	k, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	fromAcc := sdk.AccAddress(fmt.Sprintf("%-36s", seed+"from")[:36])
	toAcc := sdk.AccAddress(fmt.Sprintf("%-36s", seed+"to")[:36])

	player := testAppendPlayer(k, ctx, types.Player{
		Creator:        toAcc.String(),
		PrimaryAddress: toAcc.String(),
	})
	require.NoError(t, k.SetPlayerIndexForAddress(ctx, fromAcc.String(), player.Index))

	valAddr := sdk.ValAddress(fmt.Sprintf("%-36s", seed+"val")[:36])
	testAddValidator(k, valAddr, math.NewInt(1_000_000))

	reactor := testAppendReactor(k, ctx, types.Reactor{
		Validator:         valAddr.String(),
		RawAddress:        valAddr.Bytes(),
		DefaultCommission: math.LegacyZeroDec(),
	})

	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)
	distr := k.DistributionKeeper().(*keepertest.MockDistributionKeeper)

	require.NoError(t, mock.SetDelegation(ctx, stakingtypes.Delegation{
		DelegatorAddress: fromAcc.String(),
		ValidatorAddress: valAddr.String(),
		Shares:           math.LegacyNewDecFromInt(math.NewInt(sourceStake)),
	}))
	distr.SeedStartingInfo(valAddr, fromAcc)

	return &delegationTransferFixture{
		t:         t,
		k:         k,
		sdkCtx:    sdkCtx,
		mock:      mock,
		distr:     distr,
		player:    player,
		valAddr:   valAddr,
		fromAcc:   fromAcc,
		toAcc:     toAcc,
		reactorId: reactor.Id,
	}
}

// withDestinationStake gives the destination address a delegation to the same
// validator, which is the collision the old rekey silently overwrote.
func (f *delegationTransferFixture) withDestinationStake(t *testing.T, shares int64) {
	t.Helper()
	require.NoError(t, f.mock.SetDelegation(f.sdkCtx, stakingtypes.Delegation{
		DelegatorAddress: f.toAcc.String(),
		ValidatorAddress: f.valAddr.String(),
		Shares:           math.LegacyNewDecFromInt(math.NewInt(shares)),
	}))
	f.distr.SeedStartingInfo(f.valAddr, f.toAcc)
}

func (f *delegationTransferFixture) move(policy keeperlib.DelegationTransferPolicy) error {
	cc := f.k.NewCurrentContext(f.sdkCtx)
	err := f.k.MoveDelegationsToAddress(f.sdkCtx, cc, f.fromAcc, f.toAcc.String(), policy)
	if err == nil {
		cc.CommitAll()
	}
	return err
}

// TestDelegationTransferMergesIntoExistingDestination is the reported bug. The
// destination already delegates to the same validator, and SetDelegation is
// keyed by (delegator, validator), so writing the source's record there
// replaces the destination's rather than combining them -- while
// Validator.DelegatorShares goes on counting both, orphaning the difference.
func TestDelegationTransferMergesIntoExistingDestination(t *testing.T) {
	f := setupDelegationTransfer(t, "mergecollide", 1000)
	f.withDestinationStake(t, 400)

	sharesBefore := f.validatorShares()

	require.NoError(t, f.move(keeperlib.DelegationTransferStrict))

	merged, err := f.mock.GetDelegation(f.sdkCtx, f.toAcc, f.valAddr)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecFromInt(math.NewInt(1400)), merged.Shares,
		"the destination's own 400 shares must survive the arrival of 1000 more")

	_, err = f.mock.GetDelegation(f.sdkCtx, f.fromAcc, f.valAddr)
	require.Error(t, err, "the source delegation is gone")

	require.Equal(t, sharesBefore, f.validatorShares(),
		"shares moved between delegators, so the validator's total is unchanged")
}

func (f *delegationTransferFixture) validatorShares() math.LegacyDec {
	f.t.Helper()
	validator, err := f.mock.GetValidator(f.sdkCtx, f.valAddr)
	require.NoError(f.t, err)
	return validator.DelegatorShares
}

// TestDelegationTransferInitializesDestinationDistributionState covers the
// half of the bug with no visible symptom at the time: raw SetDelegation fires
// no hooks, so the destination pair never gets its starting info and every
// later withdraw, undelegate or redelegate fails on ErrEmptyDelegationDistInfo.
func TestDelegationTransferInitializesDestinationDistributionState(t *testing.T) {
	f := setupDelegationTransfer(t, "distinit", 1000)

	require.NoError(t, f.move(keeperlib.DelegationTransferStrict))

	has, err := f.distr.HasDelegatorStartingInfo(f.sdkCtx, f.valAddr, f.toAcc)
	require.NoError(t, err)
	require.True(t, has, "the destination must be able to price its rewards")

	has, err = f.distr.HasDelegatorStartingInfo(f.sdkCtx, f.valAddr, f.fromAcc)
	require.NoError(t, err)
	require.False(t, has, "the source settled and released its starting info")
}

// TestDelegationTransferHookOrder pins the sequence, because every step of it
// is load-bearing and none of it is enforced by types.
//
// The source is settled at its old share count before it loses them. The
// destination is settled too when it already had shares, or has its validator
// period incremented when it did not, because initializeDelegation prices from
// Period-1 and needs something to have moved it. AfterDelegationModified comes
// last because it reads the merged delegation back out of staking.
func TestDelegationTransferHookOrder(t *testing.T) {
	t.Run("destination already delegating", func(t *testing.T) {
		f := setupDelegationTransfer(t, "hookordermerge", 1000)
		f.withDestinationStake(t, 400)
		f.distr.Calls = nil

		require.NoError(t, f.move(keeperlib.DelegationTransferStrict))

		require.Equal(t, []string{
			"BeforeDelegationSharesModified:" + f.fromAcc.String() + ":" + f.valAddr.String(),
			"BeforeDelegationSharesModified:" + f.toAcc.String() + ":" + f.valAddr.String(),
			"AfterDelegationModified:" + f.toAcc.String() + ":" + f.valAddr.String(),
		}, f.distr.Calls)
	})

	t.Run("destination new to this validator", func(t *testing.T) {
		f := setupDelegationTransfer(t, "hookordernew", 1000)
		f.distr.Calls = nil

		require.NoError(t, f.move(keeperlib.DelegationTransferStrict))

		require.Equal(t, []string{
			"BeforeDelegationSharesModified:" + f.fromAcc.String() + ":" + f.valAddr.String(),
			"BeforeDelegationCreated:" + f.toAcc.String() + ":" + f.valAddr.String(),
			"AfterDelegationModified:" + f.toAcc.String() + ":" + f.valAddr.String(),
		}, f.distr.Calls)
	})
}

// TestDelegationTransferStrictRefusals covers the three states a delegation
// cannot be moved out of. Each asserts that nothing moved, which is the part
// that matters: a partial move is worse than a refused one.
func TestDelegationTransferStrictRefusals(t *testing.T) {
	tests := []struct {
		name   string
		seed   string
		reason string
		block  func(f *delegationTransferFixture)
	}{
		{
			// SlashRedelegation resolves the delegation it slashes through the
			// redelegation record's delegator address and continues on a miss,
			// so moving out from under one makes the stake unslashable.
			name:   "redelegation in flight",
			seed:   "refuseredel",
			reason: "redelegation_in_flight",
			block: func(f *delegationTransferFixture) {
				f.mock.AddReceivingRedelegation(f.fromAcc, f.valAddr)
			},
		},
		{
			// The UBD queue is keyed by (delegator, validator) and is not
			// reachable through any public keeper API, so the unbonding cannot
			// follow the delegation.
			name:   "unbonding delegation in flight",
			seed:   "refuseubd",
			reason: "defusing_in_flight",
			block: func(f *delegationTransferFixture) {
				require.NoError(f.t, f.mock.SetUnbondingDelegation(f.sdkCtx, stakingtypes.UnbondingDelegation{
					DelegatorAddress: f.fromAcc.String(),
					ValidatorAddress: f.valAddr.String(),
					Entries: []stakingtypes.UnbondingDelegationEntry{{
						Balance:        math.NewInt(100),
						InitialBalance: math.NewInt(100),
					}},
				}))
			},
		},
		{
			// What the pre-v0.21.0 rekey left behind. Such a delegation cannot
			// settle, so it cannot be moved either.
			name:   "no distribution starting info",
			seed:   "refusenodist",
			reason: "missing_distribution_state",
			block: func(f *delegationTransferFixture) {
				f.distr.ClearStartingInfo(f.valAddr, f.fromAcc)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := setupDelegationTransfer(t, test.seed, 1000)
			test.block(f)

			err := f.move(keeperlib.DelegationTransferStrict)
			require.Error(t, err)

			var transferErr *types.DelegationTransferError
			require.ErrorAs(t, err, &transferErr)
			require.Equal(t, test.reason, transferErr.Reason)
			require.Equal(t, uint32(1760), transferErr.Code())

			_, err = f.mock.GetDelegation(f.sdkCtx, f.fromAcc, f.valAddr)
			require.NoError(t, err, "a refused move must leave the source delegation exactly where it was")
			_, err = f.mock.GetDelegation(f.sdkCtx, f.toAcc, f.valAddr)
			require.Error(t, err, "and must not have written anything at the destination")
			require.Empty(t, f.distr.Calls, "nor touched distribution")
		})
	}
}

// TestDelegationTransferDisownLeavesBlockedStakeBehind is why AddressRevoke does
// not use the strict policy. Revoking is what a player does about a key they no
// longer trust, and whoever holds that key can keep a redelegation in flight
// indefinitely -- so a refusal would let an attacker block their own eviction,
// while gaining the player nothing, since that key could always have
// undelegated the stake anyway.
//
// So the blocked delegation stays bonded where it is and the player simply
// stops being credited for it, while every other validator still moves.
func TestDelegationTransferDisownLeavesBlockedStakeBehind(t *testing.T) {
	f := setupDelegationTransfer(t, "disownblocked", 1000)

	// A second validator that nothing blocks, to prove one stuck delegation
	// does not strand the rest.
	movableVal := sdk.ValAddress(fmt.Sprintf("%-36s", "disownblockedmovableval")[:36])
	testAddValidator(f.k, movableVal, math.NewInt(1_000_000))
	testAppendReactor(f.k, f.sdkCtx, types.Reactor{
		Validator:         movableVal.String(),
		RawAddress:        movableVal.Bytes(),
		DefaultCommission: math.LegacyZeroDec(),
	})
	require.NoError(t, f.mock.SetDelegation(f.sdkCtx, stakingtypes.Delegation{
		DelegatorAddress: f.fromAcc.String(),
		ValidatorAddress: movableVal.String(),
		Shares:           math.LegacyNewDecFromInt(math.NewInt(500)),
	}))
	f.distr.SeedStartingInfo(movableVal, f.fromAcc)

	// Build the blocked infusion through the normal path so it carries real
	// capacity for the player, then block it.
	f.k.ReactorUpdatePlayerInfusion(f.sdkCtx, f.fromAcc, f.valAddr)
	require.Equal(t, uint64(1000), f.playerCapacity(),
		"precondition: the player is credited for the stake at the blocked validator")
	f.mock.AddReceivingRedelegation(f.fromAcc, f.valAddr)

	require.NoError(t, f.move(keeperlib.DelegationTransferDisown))

	blocked, err := f.mock.GetDelegation(f.sdkCtx, f.fromAcc, f.valAddr)
	require.NoError(t, err, "the blocked delegation is untouched and still bonded at the SDK layer")
	require.Equal(t, math.LegacyNewDecFromInt(math.NewInt(1000)), blocked.Shares)

	_, found := f.k.GetInfusion(f.sdkCtx, f.reactorId, f.fromAcc.String())
	require.False(t, found, "but the game no longer represents it")

	moved, err := f.mock.GetDelegation(f.sdkCtx, f.toAcc, movableVal)
	require.NoError(t, err, "the unblocked validator still moves")
	require.Equal(t, math.LegacyNewDecFromInt(math.NewInt(500)), moved.Shares)

	require.Equal(t, uint64(500), f.playerCapacity(),
		"the player keeps the capacity that moved and loses only the 1000 left behind")
}

func (f *delegationTransferFixture) playerCapacity() uint64 {
	f.t.Helper()
	return f.k.GetGridAttribute(f.sdkCtx,
		keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, f.player.Id))
}

// TestDelegationTransferToSelfIsANoop guards the noop primary-address update
// that tests/test_chain.sh performs. Without the guard the address would be
// merged into itself and its shares would double.
func TestDelegationTransferToSelfIsANoop(t *testing.T) {
	f := setupDelegationTransfer(t, "selfmove", 1000)

	cc := f.k.NewCurrentContext(f.sdkCtx)
	require.NoError(t, f.k.MoveDelegationsToAddress(f.sdkCtx, cc, f.fromAcc, f.fromAcc.String(), keeperlib.DelegationTransferStrict))
	cc.CommitAll()

	unchanged, err := f.mock.GetDelegation(f.sdkCtx, f.fromAcc, f.valAddr)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecFromInt(math.NewInt(1000)), unchanged.Shares,
		"moving an address onto itself must not double its shares")
	require.Empty(t, f.distr.Calls)
}
