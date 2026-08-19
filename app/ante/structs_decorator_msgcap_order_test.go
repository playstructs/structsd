package ante_test

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sante "structs/app/ante"
	"structs/x/structs/types"
)

// The per-player message-cap loop writes to the gas-metered transient store and
// can reject mid-loop, so if it iterated the playerMsgCounts map in Go's random
// order the rejected tx's GasUsed would be node-dependent and fork consensus.
// These tests pin the loop to sorted player order: the increment order and the
// rejecting player must be identical on every run regardless of message order.

func msgCapCtx() sdk.Context {
	// DeliverTx (not CheckTx/ReCheckTx) is the only phase that runs the cap loop.
	return freeCtx().WithIsCheckTx(false)
}

func registerPlayer(mk *mockAnteKeeper, addr string, index uint64) string {
	mk.playerIndexes[addr] = index
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, addr)
	mk.permissions[addrPermId] = types.PermPlay
	return fmt.Sprintf("%d-%d", types.ObjectType_player, index)
}

func TestStructsDecorator_MsgCapIteratesInSortedOrder(t *testing.T) {
	// Three players well under the cap; messages supplied in reverse-sorted
	// order. Whatever order they arrive in, they must be counted in sorted
	// playerId order.
	addrs := []struct {
		addr  string
		index uint64
	}{
		{"structs1charlie", 9},
		{"structs1bob", 5},
		{"structs1alice", 2},
	}

	var wantOrder []string
	for _, a := range addrs {
		wantOrder = append(wantOrder, fmt.Sprintf("%d-%d", types.ObjectType_player, a.index))
	}
	sort.Strings(wantOrder)

	// Run many times to defeat Go's per-map-iteration randomization.
	for i := 0; i < 64; i++ {
		mk := newMockAnteKeeper()
		msgs := make([]sdk.Msg, 0, len(addrs))
		for _, a := range addrs {
			registerPlayer(mk, a.addr, a.index)
			// Append in the struct's (reverse-sorted) order.
			msgs = append(msgs, &types.MsgFleetMove{Creator: a.addr, FleetId: "2-1", DestinationLocationId: "7-1"})
		}

		dec := sante.NewStructsDecorator(mk, 40)
		next, called := identityHandler()

		_, err := dec.AnteHandle(msgCapCtx(), mockTx{msgs: msgs}, false, next)
		require.NoError(t, err)
		require.True(t, *called)
		require.Equal(t, wantOrder, mk.incrementOrder, "cap loop must increment in sorted player order")
	}
}

func TestStructsDecorator_MsgCapRejectsDeterministically(t *testing.T) {
	// The lexicographically-last player is already at the cap, the others are
	// fresh. Sorted iteration must increment all three (the over-cap one last)
	// and report that same player on every run — never a different one that
	// happened to sort first under a random map walk.
	alice := struct {
		addr  string
		index uint64
	}{"structs1alice", 2}
	bob := struct {
		addr  string
		index uint64
	}{"structs1bob", 5}
	carol := struct {
		addr  string
		index uint64
	}{"structs1carol", 9}

	wantOrder := []string{
		fmt.Sprintf("%d-%d", types.ObjectType_player, alice.index),
		fmt.Sprintf("%d-%d", types.ObjectType_player, bob.index),
		fmt.Sprintf("%d-%d", types.ObjectType_player, carol.index),
	}
	overCapPlayer := fmt.Sprintf("%d-%d", types.ObjectType_player, carol.index)

	for i := 0; i < 64; i++ {
		mk := newMockAnteKeeper()
		registerPlayer(mk, alice.addr, alice.index)
		registerPlayer(mk, bob.addr, bob.index)
		registerPlayer(mk, carol.addr, carol.index)
		// Carol has already used her whole allowance this block.
		mk.msgCounts[overCapPlayer] = 40

		// Messages in reverse order so tx order cannot be what makes it pass.
		msgs := []sdk.Msg{
			&types.MsgFleetMove{Creator: carol.addr, FleetId: "2-1", DestinationLocationId: "7-1"},
			&types.MsgFleetMove{Creator: bob.addr, FleetId: "2-1", DestinationLocationId: "7-1"},
			&types.MsgFleetMove{Creator: alice.addr, FleetId: "2-1", DestinationLocationId: "7-1"},
		}

		dec := sante.NewStructsDecorator(mk, 40)
		next, _ := identityHandler()

		_, err := dec.AnteHandle(msgCapCtx(), mockTx{msgs: msgs}, false, next)
		require.Error(t, err)
		require.True(t, sante.ErrPlayerMsgCapExceeded.Is(err))
		require.Contains(t, err.Error(), overCapPlayer)
		// All three were counted, the over-cap player last: proof the walk is
		// sorted and not short-circuiting on whichever player sorts first.
		require.Equal(t, wantOrder, mk.incrementOrder)
	}
}
