package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// capacityOf reads a player's grid capacity attribute directly, which is where
// an infusion's delegator share lands and therefore the only thing ownership
// actually moves.
func capacityOf(f jailEnergyFixture, playerId string) uint64 {
	return f.k.GetGridAttribute(f.ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, playerId))
}

// newRivalPlayer registers a second player on their own address, for use as the
// party an infusion is re-homed to.
func newRivalPlayer(t *testing.T, f jailEnergyFixture, seed string) types.Player {
	t.Helper()

	rivalAcc := sdk.AccAddress(fmt.Sprintf("%-36s", seed)[:36])
	return testAppendPlayer(f.k, f.ctx, types.Player{
		Creator:        rivalAcc.String(),
		PrimaryAddress: rivalAcc.String(),
	})
}

// TestSetPlayerIdMovesCapacityBetweenPlayers is the core of the ownership fix:
// the delegator's share of an infusion's power has to follow the record's owner,
// not stay with whoever held the address when the record was first written.
func TestSetPlayerIdMovesCapacityBetweenPlayers(t *testing.T) {
	f := setupJailEnergy(t, "ownmovecap", 1000, "0.04")
	rival := newRivalPlayer(t, f, "ownmovecaprival")

	require.Equal(t, uint64(960), capacityOf(f, f.player.Id), "the original owner holds the 96% delegator share")
	require.Equal(t, uint64(0), capacityOf(f, rival.Id))

	cc := f.k.NewCurrentContext(f.ctx)
	cc.GetInfusion(f.reactor.Id, f.playerAcc.String()).SetPlayerId(rival.Id)
	cc.CommitAll()

	require.Equal(t, rival.Id, f.infusion(t).PlayerId)
	require.Equal(t, uint64(0), capacityOf(f, f.player.Id), "the former owner keeps nothing")
	require.Equal(t, uint64(960), capacityOf(f, rival.Id), "the new owner is credited the same share")

	// Ownership is a claim on the delegator's share alone. The reactor's
	// commission capacity and its fuel are keyed by destination, so a re-home
	// must leave both exactly where they were.
	require.Equal(t, uint64(40), f.reactorCapacity(), "the reactor's commission is not the delegator's to move")
	require.Equal(t, uint64(1000), f.reactorFuel())
	require.Equal(t, uint64(1000), f.infusion(t).Fuel, "no stake moves with the ownership")
	require.Equal(t, uint64(1000), f.infusion(t).Power)
}

// TestSetPlayerIdIsIdempotent guards the re-run. SetInfusion emits an
// EventInfusion on every write and those events are a public API, so re-homing
// to the id already stored must write nothing at all.
func TestSetPlayerIdIsIdempotent(t *testing.T) {
	f := setupJailEnergy(t, "ownidempotent", 1000, "0.04")
	rival := newRivalPlayer(t, f, "ownidempotentrival")

	cc := f.k.NewCurrentContext(f.ctx)
	cc.GetInfusion(f.reactor.Id, f.playerAcc.String()).SetPlayerId(rival.Id)
	cc.CommitAll()

	first := f.infusion(t)
	eventsAfterFirst := len(f.ctx.EventManager().Events())

	replay := f.k.NewCurrentContext(f.ctx)
	replay.GetInfusion(f.reactor.Id, f.playerAcc.String()).SetPlayerId(rival.Id)
	replay.CommitAll()

	require.Equal(t, first, f.infusion(t), "a repeat re-home must change nothing")
	require.Equal(t, eventsAfterFirst, len(f.ctx.EventManager().Events()),
		"a repeat re-home must emit no further events")
	require.Equal(t, uint64(960), capacityOf(f, rival.Id), "and must not double-credit the new owner")
}

// TestSetPlayerIdOnPowerlessInfusion covers a gated reactor, where the record
// carries the delegator's stake but contributes no capacity. There is nothing to
// move, but the ownership still has to change, or the capacity lands on the
// wrong player the moment the validator is unjailed.
func TestSetPlayerIdOnPowerlessInfusion(t *testing.T) {
	f := setupJailEnergy(t, "ownpowerless", 1000, "0.04")
	rival := newRivalPlayer(t, f, "ownpowerlessrival")

	f.mock.JailValidator(f.valAddr)
	f.k.ReactorGateEnergy(f.ctx, f.valAddr)
	require.Equal(t, uint64(0), capacityOf(f, f.player.Id))

	cc := f.k.NewCurrentContext(f.ctx)
	cc.GetInfusion(f.reactor.Id, f.playerAcc.String()).SetPlayerId(rival.Id)
	cc.CommitAll()

	require.Equal(t, rival.Id, f.infusion(t).PlayerId)
	require.Equal(t, uint64(0), capacityOf(f, f.player.Id))
	require.Equal(t, uint64(0), capacityOf(f, rival.Id), "there was no capacity to hand over")
	require.Equal(t, uint64(1000), f.infusion(t).Fuel, "the stake is still owed to whoever holds the address")
}

// TestUpsertInfusionRehomesReassignedAddress is the end of the reported attack.
// An address is revoked and re-registered to somebody else; the next time
// staking touches the delegation behind it, the infusion must catch up with the
// address index rather than keep crediting the player who let the address go.
func TestUpsertInfusionRehomesReassignedAddress(t *testing.T) {
	f := setupJailEnergy(t, "ownreassign", 1000, "0.04")
	rival := newRivalPlayer(t, f, "ownreassignrival")

	require.Equal(t, f.player.Id, f.infusion(t).PlayerId)
	require.Equal(t, uint64(960), capacityOf(f, f.player.Id))

	// The address changes hands, exactly as AddressRevoke followed by
	// AddressRegister would leave the index.
	f.k.RevokePlayerIndexForAddress(f.ctx, f.playerAcc.String(), f.player.Index)
	require.NoError(t, f.k.SetPlayerIndexForAddress(f.ctx, f.playerAcc.String(), rival.Index))

	f.k.ReactorUpdatePlayerInfusion(f.ctx, f.playerAcc, f.valAddr)

	require.Equal(t, rival.Id, f.infusion(t).PlayerId,
		"the record follows the address, because whoever holds the address holds the stake")
	require.Equal(t, uint64(0), capacityOf(f, f.player.Id), "the former owner stops being paid for stake they cannot reach")
	require.Equal(t, uint64(960), capacityOf(f, rival.Id))
	require.Equal(t, uint64(1000), f.infusion(t).Fuel, "a re-home is not a withdrawal")
}

// TestUpsertInfusionLeavesMatchingOwnerAlone is the negative of the above: the
// overwhelmingly common case, where the address never moved, must take the same
// silent path it always did.
func TestUpsertInfusionLeavesMatchingOwnerAlone(t *testing.T) {
	f := setupJailEnergy(t, "ownunchanged", 1000, "0.04")

	before := f.infusion(t)
	eventsBefore := len(f.ctx.EventManager().Events())

	f.k.ReactorUpdatePlayerInfusion(f.ctx, f.playerAcc, f.valAddr)

	require.Equal(t, before, f.infusion(t))
	require.Equal(t, eventsBefore, len(f.ctx.EventManager().Events()),
		"a reconcile that changes nothing must emit nothing")
	require.Equal(t, uint64(960), capacityOf(f, f.player.Id))
}
