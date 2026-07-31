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

// jailEnergyFixture is a reactor with one bonded validator, one delegator, and
// a live infusion built through the normal AfterDelegationModified path so the
// grid attributes are wired exactly as they would be in production.
type jailEnergyFixture struct {
	k         keeperlib.Keeper
	ctx       sdk.Context
	mock      *keepertest.MockStakingKeeper
	player    types.Player
	playerAcc sdk.AccAddress
	valAddr   sdk.ValAddress
	reactor   types.Reactor
}

func (f jailEnergyFixture) reactorCapacity() uint64 {
	return f.k.GetGridAttribute(f.ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, f.reactor.Id))
}

func (f jailEnergyFixture) playerCapacity() uint64 {
	return f.k.GetGridAttribute(f.ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, f.player.Id))
}

func (f jailEnergyFixture) reactorFuel() uint64 {
	return f.k.GetGridAttribute(f.ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, f.reactor.Id))
}

func (f jailEnergyFixture) infusion(t *testing.T) types.Infusion {
	t.Helper()
	infusion, found := f.k.GetInfusion(f.ctx, f.reactor.Id, f.playerAcc.String())
	require.True(t, found, "infusion record should exist")
	return infusion
}

// setupJailEnergy builds the fixture. seed must be unique per test so the
// generated bech32 addresses do not collide.
func setupJailEnergy(t *testing.T, seed string, tokens int64, commission string) jailEnergyFixture {
	t.Helper()

	k, ctx := keepertest.StructsKeeper(t)

	// AccAddress requires a fixed-width byte payload; pad the seed out.
	playerAcc := sdk.AccAddress(fmt.Sprintf("%-36s", seed)[:36])
	player := types.Player{Creator: playerAcc.String(), PrimaryAddress: playerAcc.String()}
	player = testAppendPlayer(k, ctx, player)

	valAddr := sdk.ValAddress(playerAcc.Bytes())
	testAddValidator(k, valAddr, math.NewInt(tokens))

	defaultCommission, err := math.LegacyNewDecFromStr(commission)
	require.NoError(t, err)

	reactor := testAppendReactor(k, ctx, types.Reactor{
		Validator:         valAddr.String(),
		RawAddress:        valAddr.Bytes(),
		DefaultCommission: defaultCommission,
	})

	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)
	require.NoError(t, mock.SetDelegation(ctx, stakingtypes.Delegation{
		DelegatorAddress: playerAcc.String(),
		ValidatorAddress: valAddr.String(),
		Shares:           math.LegacyNewDecFromInt(math.NewInt(tokens)),
	}))

	// Build the infusion the way staking would, so Power and both capacity
	// attributes are populated by the real code path.
	k.ReactorUpdatePlayerInfusion(ctx, playerAcc, valAddr)

	return jailEnergyFixture{
		k:         k,
		ctx:       ctx,
		mock:      mock,
		player:    player,
		playerAcc: playerAcc,
		valAddr:   valAddr,
		reactor:   reactor,
	}
}

// TestReactorGateEnergyPreservesStake is the regression test for the blocking
// bug behind this feature: a zero ratio must not be mistaken for an empty
// infusion, or the EndBlock destruction queue deletes the record and the
// delegator's fuel with it.
func TestReactorGateEnergyPreservesStake(t *testing.T) {
	f := setupJailEnergy(t, "gatepreserve", 1000, "0")

	require.Equal(t, uint64(1000), f.infusion(t).Fuel)
	require.Equal(t, uint64(1000), f.infusion(t).Power)
	require.Equal(t, uint64(1000), f.playerCapacity())

	f.mock.JailValidator(f.valAddr)
	f.k.ReactorGateEnergy(f.ctx, f.valAddr)

	gated := f.infusion(t)
	require.Equal(t, uint64(0), gated.Ratio, "gate should zero the ratio")
	require.Equal(t, uint64(0), gated.Power, "power follows ratio to zero")
	require.Equal(t, uint64(1000), gated.Fuel, "stake must be untouched")
	require.Equal(t, uint64(0), f.playerCapacity(), "energy is withdrawn")
	require.Equal(t, uint64(1000), f.reactorFuel(), "reactor fuel records the stake unchanged")

	// The EndBlocker runs the destruction queue. A gated infusion still holds
	// fuel, so it must survive.
	_, err := f.k.EndBlocker(f.ctx)
	require.NoError(t, err)

	survived, found := f.k.GetInfusion(f.ctx, f.reactor.Id, f.playerAcc.String())
	require.True(t, found, "gated infusion must survive the destruction queue")
	require.Equal(t, uint64(1000), survived.Fuel)
}

// TestReactorGateEnergyZeroesCommissionAndPlayerShares confirms the whole-reactor
// blast radius: the operator's commission energy and the delegator's share both
// go to zero.
func TestReactorGateEnergyZeroesCommissionAndPlayerShares(t *testing.T) {
	f := setupJailEnergy(t, "gatecommission", 1000, "0.25")

	require.Equal(t, uint64(250), f.reactorCapacity(), "reactor holds the commission share")
	require.Equal(t, uint64(750), f.playerCapacity(), "delegator holds the remainder")

	f.mock.JailValidator(f.valAddr)
	f.k.ReactorGateEnergy(f.ctx, f.valAddr)

	require.Equal(t, uint64(0), f.reactorCapacity(), "commission energy is gated too")
	require.Equal(t, uint64(0), f.playerCapacity())
	require.Equal(t, uint64(1000), f.infusion(t).Fuel, "stake still untouched")
}

// TestReactorGateEnergyIdempotent asserts a repeat gate writes nothing new, so
// replayed blocks and duplicate triggers do not emit redundant indexer events.
func TestReactorGateEnergyIdempotent(t *testing.T) {
	f := setupJailEnergy(t, "gateidempotent", 500, "0")

	f.mock.JailValidator(f.valAddr)
	f.k.ReactorGateEnergy(f.ctx, f.valAddr)
	first := f.infusion(t)

	f.k.ReactorGateEnergy(f.ctx, f.valAddr)
	second := f.infusion(t)

	require.Equal(t, first, second)
	require.Equal(t, uint64(0), f.playerCapacity())
}

// TestAfterValidatorBeginUnbondingGatesWhenJailed covers the primary trigger:
// staking moves a jailed validator out of the bonded set in the same block as
// the jail, and that transition is what gates the reactor.
func TestAfterValidatorBeginUnbondingGatesWhenJailed(t *testing.T) {
	f := setupJailEnergy(t, "hookgate", 1000, "0")
	require.Equal(t, uint64(1000), f.playerCapacity())

	f.mock.JailValidator(f.valAddr)
	require.NoError(t, f.k.Hooks().AfterValidatorBeginUnbonding(f.ctx, sdk.ConsAddress(f.valAddr), f.valAddr))

	require.Equal(t, uint64(0), f.infusion(t).Power)
	require.Equal(t, uint64(0), f.playerCapacity())
	require.Equal(t, uint64(1000), f.infusion(t).Fuel)
}

// TestAfterValidatorBeginUnbondingIgnoresUnjailed is the counterpart: leaving
// the active set voluntarily is not misconduct, so a validator that is merely
// unbonding keeps producing energy. Gating here would punish ordinary
// undelegation.
func TestAfterValidatorBeginUnbondingIgnoresUnjailed(t *testing.T) {
	f := setupJailEnergy(t, "hookunjailed", 1000, "0")

	require.NoError(t, f.k.Hooks().AfterValidatorBeginUnbonding(f.ctx, sdk.ConsAddress(f.valAddr), f.valAddr))

	require.Equal(t, uint64(1000), f.infusion(t).Power, "unjailed unbonding must not be gated")
	require.Equal(t, uint64(1000), f.playerCapacity())
}

// TestAfterValidatorBeginUnbondingMissingValidatorFailsClosed asserts the
// fail-closed direction: a validator that cannot be read is treated as
// unhealthy rather than assumed fine.
func TestAfterValidatorBeginUnbondingMissingValidatorFailsClosed(t *testing.T) {
	f := setupJailEnergy(t, "hookmissing", 1000, "0")

	f.mock.RemoveValidator(f.valAddr)
	require.NoError(t, f.k.Hooks().AfterValidatorBeginUnbonding(f.ctx, sdk.ConsAddress(f.valAddr), f.valAddr))

	require.Equal(t, uint64(0), f.infusion(t).Power, "missing validator must be gated")
	require.Equal(t, uint64(1000), f.infusion(t).Fuel)
}

// TestAfterValidatorBondedRestores covers the routine unjail: MsgUnjail clears
// the flag, staking rebonds the validator at the end of that block, and the
// bonded hook brings energy back.
func TestAfterValidatorBondedRestores(t *testing.T) {
	f := setupJailEnergy(t, "hookrestore", 1000, "0")

	f.mock.JailValidator(f.valAddr)
	f.k.ReactorGateEnergy(f.ctx, f.valAddr)
	require.Equal(t, uint64(0), f.playerCapacity())

	f.mock.UnjailValidator(f.valAddr)
	f.mock.BondValidator(f.valAddr)
	require.NoError(t, f.k.Hooks().AfterValidatorBonded(f.ctx, sdk.ConsAddress(f.valAddr), f.valAddr))

	require.Equal(t, uint64(1000), f.infusion(t).Power, "energy returns on rebond")
	require.Equal(t, uint64(1000), f.playerCapacity())
}

// TestReactorRestoreEnergyReflectsSlashDuringJail asserts restoration reconciles
// against live staking rather than replaying the pre-jail number. A validator
// slashed while jailed must come back proportionally weaker.
func TestReactorRestoreEnergyReflectsSlashDuringJail(t *testing.T) {
	f := setupJailEnergy(t, "restoreslash", 1000, "0")

	f.mock.JailValidator(f.valAddr)
	f.k.ReactorGateEnergy(f.ctx, f.valAddr)

	// Slashing halves the token pool while delegator shares stay put, so each
	// share is worth half as much.
	f.mock.SlashValidatorTokens(f.valAddr, math.NewInt(500))
	f.mock.UnjailValidator(f.valAddr)

	f.k.ReactorRestoreEnergy(f.ctx, f.valAddr)

	require.Equal(t, uint64(500), f.infusion(t).Fuel, "fuel must be recomputed from live stake")
	require.Equal(t, uint64(500), f.infusion(t).Power)
	require.Equal(t, uint64(500), f.playerCapacity(), "energy returns at the slashed level")
}

// TestReactorRestoreEnergyHealsStaleFuel is the reason restoration reconciles
// instead of simply writing a non-zero ratio back. A row whose delegation is
// gone but whose fuel lingers must be zeroed, not relit into real capacity.
func TestReactorRestoreEnergyHealsStaleFuel(t *testing.T) {
	f := setupJailEnergy(t, "restorestale", 1000, "0")

	f.mock.JailValidator(f.valAddr)
	f.k.ReactorGateEnergy(f.ctx, f.valAddr)
	require.Equal(t, uint64(1000), f.infusion(t).Fuel)

	// The delegation disappears while the reactor is gated, leaving stale fuel.
	require.NoError(t, f.mock.RemoveDelegation(f.ctx, stakingtypes.Delegation{
		DelegatorAddress: f.playerAcc.String(),
		ValidatorAddress: f.valAddr.String(),
	}))
	f.mock.UnjailValidator(f.valAddr)

	f.k.ReactorRestoreEnergy(f.ctx, f.valAddr)

	require.Equal(t, uint64(0), f.infusion(t).Fuel, "stale fuel must be cleared")
	require.Equal(t, uint64(0), f.infusion(t).Power)
	require.Equal(t, uint64(0), f.playerCapacity(), "no capacity for stake that no longer exists")
}

// TestReactorRestoreEnergyStillJailedRegates asserts restoration derives the
// ratio from validator health rather than trusting its caller, so calling it on
// a still-jailed reactor is a re-gate and not a bypass.
func TestReactorRestoreEnergyStillJailedRegates(t *testing.T) {
	f := setupJailEnergy(t, "restorejailed", 1000, "0")

	f.mock.JailValidator(f.valAddr)
	f.k.ReactorRestoreEnergy(f.ctx, f.valAddr)

	require.Equal(t, uint64(0), f.infusion(t).Power, "a jailed reactor cannot be restored")
	require.Equal(t, uint64(1000), f.infusion(t).Fuel)
	require.Equal(t, uint64(0), f.playerCapacity())
}

// TestReactorEnergyHooksNoReactorIsNoop asserts both transitions are safe when
// the validator has no reactor at all, which happens for validators created
// before the structs module tracked them.
func TestReactorEnergyHooksNoReactorIsNoop(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	orphanAcc := sdk.AccAddress("orphanvalidator_padding_____________")
	orphanVal := sdk.ValAddress(orphanAcc.Bytes())
	testAddValidator(k, orphanVal, math.NewInt(1000))

	require.NotPanics(t, func() {
		k.ReactorGateEnergy(ctx, orphanVal)
		k.ReactorRestoreEnergy(ctx, orphanVal)
	})

	require.NoError(t, k.Hooks().AfterValidatorBeginUnbonding(ctx, sdk.ConsAddress(orphanVal), orphanVal))
	require.NoError(t, k.Hooks().AfterValidatorBonded(ctx, sdk.ConsAddress(orphanVal), orphanVal))
}

// TestReactorRatioDriftAcrossHookPaths is the drift regression. Every path that
// writes an infusion ratio must derive it, or a jailed reactor would quietly
// have its energy restored by unrelated staking activity.
func TestReactorRatioDriftAcrossHookPaths(t *testing.T) {
	t.Run("delegation modified does not restore a jailed reactor", func(t *testing.T) {
		f := setupJailEnergy(t, "driftdelegation", 1000, "0")

		f.mock.JailValidator(f.valAddr)
		f.k.ReactorGateEnergy(f.ctx, f.valAddr)

		f.k.ReactorUpdatePlayerInfusion(f.ctx, f.playerAcc, f.valAddr)

		require.Equal(t, uint64(0), f.infusion(t).Ratio, "AfterDelegationModified must not undo the gate")
		require.Equal(t, uint64(0), f.playerCapacity())
	})

	t.Run("unbonding initiated does not restore a jailed reactor", func(t *testing.T) {
		f := setupJailEnergy(t, "driftunbonding", 1000, "0")

		f.mock.JailValidator(f.valAddr)
		f.k.ReactorGateEnergy(f.ctx, f.valAddr)

		require.NoError(t, f.mock.SetUnbondingDelegation(f.ctx, stakingtypes.UnbondingDelegation{
			DelegatorAddress: f.playerAcc.String(),
			ValidatorAddress: f.valAddr.String(),
			Entries: []stakingtypes.UnbondingDelegationEntry{
				{Balance: math.NewInt(100), InitialBalance: math.NewInt(100), UnbondingId: 1},
			},
		}))
		f.k.ReactorInfusionUnbonding(f.ctx, 1)

		require.Equal(t, uint64(0), f.infusion(t).Ratio, "AfterUnbondingInitiated must not undo the gate")
		require.Equal(t, uint64(0), f.playerCapacity())
	})

	t.Run("maturity sweep does not restore a jailed reactor", func(t *testing.T) {
		f := setupJailEnergy(t, "driftsweep", 1000, "0")

		f.mock.JailValidator(f.valAddr)
		f.k.ReactorGateEnergy(f.ctx, f.valAddr)

		f.k.ReconcileInfusionForDelegation(f.ctx, f.playerAcc, f.valAddr)

		require.Equal(t, uint64(0), f.infusion(t).Ratio, "the maturity sweep must not undo the gate")
		require.Equal(t, uint64(0), f.playerCapacity())
	})
}

// TestGatedPlayerComputesOfflineWithStructsStillFlaggedOnline documents the
// behaviour that surprises operators: gating never clears a struct's online
// flag. Struct draw lives in structsLoad, which the grid cascade does not read,
// so what actually changes is that the player computes as offline and every
// power-gated action starts failing.
func TestGatedPlayerComputesOfflineWithStructsStillFlaggedOnline(t *testing.T) {
	f := setupJailEnergy(t, "gatedoffline", 1000, "0")

	// Put a struct's worth of draw on the player.
	structsLoadId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, f.player.Id)
	f.k.SetGridAttribute(f.ctx, structsLoadId, uint64(400))

	cc := f.k.NewCurrentContext(f.ctx)
	player, err := cc.GetPlayer(f.player.Id)
	require.NoError(t, err)
	require.True(t, player.IsOnline(), "1000 capacity covers 400 of draw")

	f.mock.JailValidator(f.valAddr)
	f.k.ReactorGateEnergy(f.ctx, f.valAddr)

	cc2 := f.k.NewCurrentContext(f.ctx)
	gatedPlayer, err := cc2.GetPlayer(f.player.Id)
	require.NoError(t, err)
	require.True(t, gatedPlayer.IsOffline(), "capacity no longer covers the struct draw")
	require.Equal(t, uint64(400), f.k.GetGridAttribute(f.ctx, structsLoadId), "struct draw is left in place")
}

// TestGateEnergyCascadesOverCapacityAllocations asserts the accepted lossy
// behaviour. Capacity dropping to zero drives the cascade, which destroys the
// player's outgoing allocation rather than suspending it. This is deliberate;
// see the v0.21.0 upgrade notes before changing it.
func TestGateEnergyCascadesOverCapacityAllocations(t *testing.T) {
	f := setupJailEnergy(t, "gatecascade", 1000, "0")

	substation, _, err := testAppendSubstation(f.k, f.ctx, types.Allocation{}, f.player)
	require.NoError(t, err)

	allocation, err := testAppendAllocation(f.k, f.ctx, types.Allocation{
		Creator:        f.player.Creator,
		Controller:     f.player.Id,
		SourceObjectId: f.player.Id,
		DestinationId:  substation.Id,
		Type:           types.AllocationType_static,
	}, 600)
	require.NoError(t, err)

	// Outgoing allocations show up as load on their source.
	f.k.SetGridAttribute(f.ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, f.player.Id), uint64(600))

	_, found := f.k.GetAllocation(f.ctx, allocation.Id)
	require.True(t, found)

	f.mock.JailValidator(f.valAddr)
	f.k.ReactorGateEnergy(f.ctx, f.valAddr)

	// The cascade runs in the EndBlocker, after the gate has already dropped
	// capacity to zero.
	_, err = f.k.EndBlocker(f.ctx)
	require.NoError(t, err)

	_, stillThere := f.k.GetAllocation(f.ctx, allocation.Id)
	require.False(t, stillThere, "an allocation the player can no longer power is destroyed, not suspended")
}

// TestMsgReactorRestartRestoresWithoutRebond covers the case the hooks cannot
// reach: an operator who unjails but stays below the active-set cutoff is never
// rebonded, so AfterValidatorBonded never fires and this message is their only
// route back.
func TestMsgReactorRestartRestoresWithoutRebond(t *testing.T) {
	f := setupJailEnergy(t, "restartrestore", 1000, "0")
	ms := keeperlib.NewMsgServerImpl(f.k)

	f.mock.JailValidator(f.valAddr)
	f.k.ReactorGateEnergy(f.ctx, f.valAddr)
	require.Equal(t, uint64(0), f.playerCapacity())

	// Unjailed but deliberately left out of the bonded set, so no rebond ever
	// happens and AfterValidatorBonded never fires.
	f.mock.UnjailValidator(f.valAddr)

	_, err := ms.ReactorRestart(f.ctx, &types.MsgReactorRestart{
		Creator:          f.player.Creator,
		ValidatorAddress: f.valAddr.String(),
	})
	require.NoError(t, err)

	require.Equal(t, uint64(1000), f.infusion(t).Power)
	require.Equal(t, uint64(1000), f.playerCapacity())
}

// TestMsgReactorRestartForceGatesJailedReactor asserts the message cannot be
// used to escape the gate, and doubles as a manual remediation path if an
// automatic gate is ever missed.
func TestMsgReactorRestartForceGatesJailedReactor(t *testing.T) {
	f := setupJailEnergy(t, "restartforcegate", 1000, "0")
	ms := keeperlib.NewMsgServerImpl(f.k)

	f.mock.JailValidator(f.valAddr)
	require.Equal(t, uint64(1000), f.playerCapacity(), "not yet gated")

	_, err := ms.ReactorRestart(f.ctx, &types.MsgReactorRestart{
		Creator:          f.player.Creator,
		ValidatorAddress: f.valAddr.String(),
	})
	require.NoError(t, err)

	require.Equal(t, uint64(0), f.playerCapacity(), "restart gates a jailed reactor")
	require.Equal(t, uint64(1000), f.infusion(t).Fuel)
}

// TestMsgReactorRestartIsPermissionless asserts a caller who neither owns the
// reactor nor holds any permission on it can still trigger reconciliation. The
// message only writes state derived from staking, so there is nothing to guard.
func TestMsgReactorRestartIsPermissionless(t *testing.T) {
	f := setupJailEnergy(t, "restartperms", 1000, "0")
	ms := keeperlib.NewMsgServerImpl(f.k)

	strangerAcc := sdk.AccAddress("strangerplayer_padding______________")
	stranger := types.Player{Creator: strangerAcc.String(), PrimaryAddress: strangerAcc.String()}
	stranger = testAppendPlayer(f.k, f.ctx, stranger)

	f.mock.JailValidator(f.valAddr)

	_, err := ms.ReactorRestart(f.ctx, &types.MsgReactorRestart{
		Creator:          stranger.Creator,
		ValidatorAddress: f.valAddr.String(),
	})
	require.NoError(t, err)

	require.Equal(t, uint64(0), f.playerCapacity(), "any player may reconcile any reactor")
}

// TestMsgReactorRestartRejectsUnknownInputs covers the two failure modes.
func TestMsgReactorRestartRejectsUnknownInputs(t *testing.T) {
	f := setupJailEnergy(t, "restartreject", 1000, "0")
	ms := keeperlib.NewMsgServerImpl(f.k)

	_, err := ms.ReactorRestart(f.ctx, &types.MsgReactorRestart{
		Creator:          f.player.Creator,
		ValidatorAddress: "not-a-validator-address",
	})
	require.Error(t, err)

	strayAcc := sdk.AccAddress("strayvalidator_padding______________")
	_, err = ms.ReactorRestart(f.ctx, &types.MsgReactorRestart{
		Creator:          f.player.Creator,
		ValidatorAddress: sdk.ValAddress(strayAcc.Bytes()).String(),
	})
	require.Error(t, err, "a validator with no reactor has nothing to restart")
}

// TestGenesisImportReactorInfusionsGatesJailedValidator asserts a chain exported
// and reimported while a validator sits in jail does not hand back the energy
// the gate removed. The gated ratio is derived state, so it is rebuilt on import
// rather than carried in genesis.
func TestGenesisImportReactorInfusionsGatesJailedValidator(t *testing.T) {
	f := setupJailEnergy(t, "genesisjailed", 1000, "0")

	f.mock.JailValidator(f.valAddr)

	cc := f.k.NewCurrentContext(f.ctx)
	cc.GenesisImportReactorInfusions(f.reactor)
	cc.CommitAll()

	require.Equal(t, uint64(0), f.infusion(t).Ratio, "a jailed validator imports gated")
	require.Equal(t, uint64(0), f.infusion(t).Power)
}
