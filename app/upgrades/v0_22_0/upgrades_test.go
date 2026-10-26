package v0_22_0_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"structs/app/upgrades"
	v0_22_0 "structs/app/upgrades/v0_22_0"
	keepertest "structs/testutil/keeper"
	structskeeper "structs/x/structs/keeper"
	"structs/x/structs/types"
)

/* TestMigrateSelfDefenseRegistrations covers the rows an old chain is holding.
 *
 * StructDefenseSet never compared the defender and protected ids, and
 * IsProtecting compares locations - a struct is trivially co-located with
 * itself - so a struct could be registered as its own defender. In
 * ResolveDefenders that put the target's counter in the defender pass, before
 * the volley it was supposed to survive first.
 *
 * The runtime guard makes such a row inert, so this is not what closes the
 * exploit. It matters because the row is also a registration slot: a struct
 * defends one target at a time, so leaving the self-row in place would keep that
 * struct's real assignment blocked.
 */
func TestMigrateSelfDefenseRegistrations(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	selfDefender := types.Struct{Id: "4-1", Index: 1}
	protected := types.Struct{Id: "4-2", Index: 2}
	realDefender := types.Struct{Id: "4-3", Index: 3}

	// The bad row: a struct defending itself.
	k.SetStructDefender(ctx, selfDefender.Id, selfDefender.Index, selfDefender.Id)
	// A legitimate row that must survive untouched.
	k.SetStructDefender(ctx, protected.Id, protected.Index, realDefender.Id)

	require.Contains(t, k.GetAllStructDefender(ctx, selfDefender.Id), selfDefender.Id,
		"fixture sanity: the self-registration is on disk")

	require.NoError(t, v0_22_0.MigrateSelfDefenseRegistrations(ctx, keepers))

	require.Empty(t, k.GetAllStructDefender(ctx, selfDefender.Id),
		"a struct registered as its own defender must be cleared")
	require.Zero(t, k.GetStructAttribute(ctx, structAttributeProtectedIndex(selfDefender.Id)),
		"the defending struct's assignment slot must be freed, not just the index row")

	require.Equal(t, []string{realDefender.Id}, k.GetAllStructDefender(ctx, protected.Id),
		"a legitimate registration must be untouched")

	// Idempotent: a second run finds nothing left to do.
	require.NoError(t, v0_22_0.MigrateSelfDefenseRegistrations(ctx, keepers))
	require.Equal(t, []string{realDefender.Id}, k.GetAllStructDefender(ctx, protected.Id))
}

func structAttributeProtectedIndex(structId string) string {
	return structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, structId)
}
