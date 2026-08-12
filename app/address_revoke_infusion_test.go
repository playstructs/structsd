package app_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	structskeeper "structs/x/structs/keeper"
	structstypes "structs/x/structs/types"
)

// TestAddressRevokeMovesInfusionWithoutLosingCapacity is the release gate for
// the address-move repair, and like the redelegation gate it has to run against
// real staking.
//
// AddressRevoke sweeps the revoked address's delegations onto the player's
// primary address with a RemoveDelegation followed by a SetDelegation, and those
// two calls are asymmetric in a way only real staking shows: RemoveDelegation
// fires BeforeDelegationRemoved, which zeroes the source infusion, while
// SetDelegation is a bare store write that fires nothing. Left to the hooks the
// player's energy is destroyed by an operation that is supposed to relocate it.
// The keeper's mock staking keeper fires no hooks at all, so no unit test can
// see the half that goes wrong.
//
// If this fails after an SDK upgrade, check whether SetDelegation has grown
// hooks of its own, or whether RemoveDelegation has stopped firing them.
func TestAddressRevokeMovesInfusionWithoutLosingCapacity(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)

	bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	stakingMsgServer := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	valAddr := createRedelegationValidator(t, bApp, ctx, stakingMsgServer, bondDenom, "addressrevoke")

	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	reactorId := jailGateReactorId(t, bApp, ctx, valAddr)

	// One player, two addresses. The primary never delegates, so anything the
	// primary ends up holding arrived through the sweep.
	primaryAcc := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	secondaryAcc := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())

	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	player := cc.UpsertPlayer(primaryAcc.String())
	playerId := player.GetPlayerId()
	playerIndex := player.GetIndex()
	cc.CommitAll()

	require.NoError(t, bApp.StructsKeeper.SetPlayerIndexForAddress(ctx, secondaryAcc.String(), playerIndex))
	bApp.StructsKeeper.SetPermissionsByBytes(ctx,
		structskeeper.GetAddressPermissionIDBytes(secondaryAcc.String()), structstypes.PermAll)

	stake := math.NewInt(4_000_000)
	fundRedelegationAccount(t, bApp, ctx, secondaryAcc, sdk.NewCoins(sdk.NewCoin(bondDenom, stake)))

	_, err = stakingMsgServer.Delegate(ctx, stakingtypes.NewMsgDelegate(
		secondaryAcc.String(), valAddr.String(), sdk.NewCoin(bondDenom, stake)))
	require.NoError(t, err)

	before, found := bApp.StructsKeeper.GetInfusion(ctx, reactorId, secondaryAcc.String())
	require.True(t, found, "delegating from the secondary address should have infused the reactor")
	require.Equal(t, stake.Uint64(), before.Fuel)
	require.Equal(t, playerId, before.PlayerId, "the secondary address belongs to this player")

	_, _, playerPower := before.GetPowerDistribution()
	require.NotZero(t, playerPower)

	capacityBefore := redelegationCapacity(t, bApp, ctx, playerId)
	// The reactor's capacity also carries the operator's own self-delegation, so
	// only the player's is this infusion's alone.
	reactorCapacityBefore := redelegationCapacity(t, bApp, ctx, reactorId)
	require.Equal(t, playerPower, capacityBefore)

	structsMsgServer := structskeeper.NewMsgServerImpl(bApp.StructsKeeper)
	_, err = structsMsgServer.AddressRevoke(ctx, &structstypes.MsgAddressRevoke{
		Creator: primaryAcc.String(),
		Address: secondaryAcc.String(),
	})
	require.NoError(t, err)

	require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, secondaryAcc.String()),
		"the address should have been revoked")

	// The stake itself moved address, not validator.
	_, err = bApp.StakingKeeper.GetDelegation(ctx, primaryAcc, valAddr)
	require.NoError(t, err, "the delegation should now be held by the primary address")
	_, err = bApp.StakingKeeper.GetDelegation(ctx, secondaryAcc, valAddr)
	require.Error(t, err, "and no longer by the revoked one")

	after, found := bApp.StructsKeeper.GetInfusion(ctx, reactorId, primaryAcc.String())
	require.True(t, found, "the primary address must hold the infusion the sweep gave it")
	require.Equal(t, stake.Uint64(), after.Fuel, "the stake should have arrived intact")
	require.Equal(t, playerId, after.PlayerId)

	if stale, stillThere := bApp.StructsKeeper.GetInfusion(ctx, reactorId, secondaryAcc.String()); stillThere {
		require.Zero(t, stale.Fuel, "the revoked address must keep no fuel")
		require.Zero(t, stale.Power)
	}

	// The heart of it. Revoking an address relocates the player's energy; it is
	// not a way to burn it.
	require.Equal(t, capacityBefore, redelegationCapacity(t, bApp, ctx, playerId),
		"revoking an address must not cost the player capacity")
	require.Equal(t, reactorCapacityBefore, redelegationCapacity(t, bApp, ctx, reactorId),
		"nor cost the reactor the commission it still earns on the same stake")

	// Net-zero is not the same as never dipping: the source leg subtracts before
	// the destination leg adds it back, and the grid cascade queued in between is
	// only drained in the EndBlocker. By then the player must be whole, or a
	// legitimate revoke sheds their allocations.
	verify := bApp.StructsKeeper.NewCurrentContext(ctx)
	require.True(t, verify.GetPlayer(playerId).IsOnline(),
		"a legitimate address revoke must not knock the player offline")
}
