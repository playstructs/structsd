package keeper_test

import (
	"fmt"
	"sort"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
)

// Committing a cache map in Go map order is only safe while every Commit writes
// to keys derived from its own map key. AddressCache.Commit does not: it
// allocates an auth account number from the account keeper's global sequence
// for any address that has none. These tests pin the sorted order that makes
// that allocation a function of the data instead of the runtime.

// commitOrderAddresses builds n valid bech32 addresses whose sorted order is
// deliberately not their construction order, so a test that passes can only be
// passing because the commit sorted them.
func commitOrderAddresses(t *testing.T, n int) []string {
	t.Helper()

	addresses := make([]string, 0, n)
	for i := 0; i < n; i++ {
		// The leading digit varies the bech32 payload enough that sorted order
		// and insertion order disagree.
		seed := fmt.Sprintf("%d_commit_order_padding_address_1234", (n-i)*7%n)
		addresses = append(addresses, sdk.AccAddress(seed).String())
	}

	require.Len(t, addresses, n)
	return addresses
}

func TestCommitOrder_AccountNumbersFollowSortedAddressOrder(t *testing.T) {
	// One run can hit sorted order by luck, so repeat with fresh contexts. Go
	// randomizes map iteration per range statement, so a handful of rounds
	// makes an unsorted implementation fail essentially every time.
	const rounds = 12

	for round := 0; round < rounds; round++ {
		k, ctx := keepertest.StructsKeeper(t)
		accountKeeper := keepertest.AccountKeeperFrom(t, ctx)

		addresses := commitOrderAddresses(t, 8)

		cc := k.NewCurrentContext(ctx)
		for i, address := range addresses {
			cc.SetPlayerIndexForAddress(address, uint64(i+1))
		}
		cc.CommitAll()

		want := append([]string(nil), addresses...)
		sort.Strings(want)

		require.Equal(t, want, accountKeeper.AccountCreationOrder(),
			"round %d: auth accounts must be created in sorted address order, not Go map order", round)

		// The numbers themselves are what lands in auth state and therefore in
		// the app hash, so assert those and not only the call order.
		for wantNumber, address := range want {
			accAddress, err := sdk.AccAddressFromBech32(address)
			require.NoError(t, err)

			account := accountKeeper.GetAccount(ctx, accAddress)
			require.NotNil(t, account, "round %d: %s should have an auth account", round, address)
			require.Equal(t, uint64(wantNumber), account.GetAccountNumber(),
				"round %d: %s got the wrong account number", round, address)
		}
	}
}

// TestCommitOrder_IsStableAcrossContexts is the property a validator actually
// depends on: two nodes handed the same work must write the same auth state.
// Each CurrentContext ranges its own maps, so this is the closest a unit test
// gets to two independent nodes replaying one block.
func TestCommitOrder_IsStableAcrossContexts(t *testing.T) {
	addresses := commitOrderAddresses(t, 6)

	var reference []string

	for round := 0; round < 10; round++ {
		k, ctx := keepertest.StructsKeeper(t)
		accountKeeper := keepertest.AccountKeeperFrom(t, ctx)

		cc := k.NewCurrentContext(ctx)
		for i, address := range addresses {
			cc.SetPlayerIndexForAddress(address, uint64(i+1))
		}
		cc.CommitAll()

		got := accountKeeper.AccountCreationOrder()
		if reference == nil {
			reference = got
			continue
		}

		require.Equal(t, reference, got,
			"round %d diverged from the first run: two nodes would derive different auth state", round)
	}
}

// An address that already has an auth account must not consume a number, so the
// sequence only ever advances for genuinely new addresses.
func TestCommitOrder_ExistingAccountsAreNotRenumbered(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	accountKeeper := keepertest.AccountKeeperFrom(t, ctx)

	addresses := commitOrderAddresses(t, 4)
	sort.Strings(addresses)

	first := k.NewCurrentContext(ctx)
	for i, address := range addresses {
		first.SetPlayerIndexForAddress(address, uint64(i+1))
	}
	first.CommitAll()

	before := make([]uint64, 0, len(addresses))
	for _, address := range addresses {
		accAddress, err := sdk.AccAddressFromBech32(address)
		require.NoError(t, err)
		before = append(before, accountKeeper.GetAccount(ctx, accAddress).GetAccountNumber())
	}

	// Re-index the same addresses. They all have accounts now, so nothing new
	// should be created.
	second := k.NewCurrentContext(ctx)
	for i, address := range addresses {
		second.SetPlayerIndexForAddress(address, uint64(i+1))
	}
	second.CommitAll()

	require.Len(t, accountKeeper.AccountCreationOrder(), len(addresses),
		"re-indexing known addresses must not create more accounts")

	for i, address := range addresses {
		accAddress, err := sdk.AccAddressFromBech32(address)
		require.NoError(t, err)
		require.Equal(t, before[i], accountKeeper.GetAccount(ctx, accAddress).GetAccountNumber(),
			"%s was renumbered", address)
	}
}

// An unparseable address parses to the empty AccAddress, which has no auth
// account, so the old discarded-error form created and numbered an account for
// the *empty* address and burned a sequence number on a non-address. The index
// row is an ordinary KV write keyed by the string and stays; only the account
// provisioning is refused.
func TestCommitOrder_UnparseableAddressGetsNoAuthAccount(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	accountKeeper := keepertest.AccountKeeperFrom(t, ctx)

	const garbage = "not-a-bech32-address"

	err := k.SetPlayerIndexForAddress(ctx, garbage, 7)
	require.Error(t, err, "the caller must be told this string cannot be a signing address")

	require.Empty(t, accountKeeper.AccountCreationOrder(),
		"no auth account may be created, least of all one for the empty address")
	require.Equal(t, uint64(7), k.GetPlayerIndexFromAddress(ctx, garbage),
		"the index row is order-independent and still belongs in state")

	// The sequence must be untouched, so the next real address still gets 0.
	good := sdk.AccAddress("good_padding_address_1234567890").String()
	require.NoError(t, k.SetPlayerIndexForAddress(ctx, good, 1))

	goodAcc, errParam := sdk.AccAddressFromBech32(good)
	require.NoError(t, errParam)
	require.Equal(t, uint64(0), accountKeeper.GetAccount(ctx, goodAcc).GetAccountNumber(),
		"the refused address must not have advanced the account-number sequence")
}
