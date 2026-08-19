package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// transferFixture is an unconnected allocation controlled by its creator, which is
// the shape AllocationCreate produces when msg.Controller is left empty.
type transferFixture struct {
	k       keeperlib.Keeper
	ms      types.MsgServer
	ctx     sdk.Context
	owner   types.Player
	alloc   types.Allocation
	ownerId []byte
}

func setupTransferFixture(t *testing.T, sourceObjectId string, ownerSeed string) transferFixture {
	t.Helper()

	k, ms, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	ownerAcc := sdk.AccAddress(ownerSeed)
	owner := types.Player{
		Creator:        ownerAcc.String(),
		PrimaryAddress: ownerAcc.String(),
	}
	owner = testAppendPlayer(k, ctx, owner)

	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceObjectId)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(1000))

	alloc, err := testAppendAllocation(k, ctx, types.Allocation{
		SourceObjectId: sourceObjectId,
		DestinationId:  "",
		Type:           types.AllocationType_static,
		Controller:     owner.Id,
	}, 100)
	require.NoError(t, err)

	return transferFixture{
		k:       k,
		ms:      ms,
		ctx:     ctx,
		owner:   owner,
		alloc:   alloc,
		ownerId: keeperlib.GetObjectPermissionIDBytes(alloc.Id, owner.Id),
	}
}

// TestAllocationTransferRejectsControllerThatIsNotAPlayer pins the fix for a
// transfer to a player id that names nobody.
//
// The handler validated msg.Controller with a guard on CurrentContext.GetPlayer,
// which never returns an error, so the branch was unreachable and the id was
// effectively unchecked. Nothing downstream covered it either: CanBeTransferBy
// asks only whether the caller may transfer. Unlike the guild membership paths,
// which reach a player write and panic on the empty key, this one commits — it
// writes the allocation, keyed by allocation id — so the damage persisted.
func TestAllocationTransferRejectsControllerThatIsNotAPlayer(t *testing.T) {
	f := setupTransferFixture(t, "transfer-phantom-source", "alloc_phantom_owner_01")

	phantom := "1-999"
	_, err := f.ms.AllocationTransfer(f.ctx, &types.MsgAllocationTransfer{
		Creator:      f.owner.Creator,
		AllocationId: f.alloc.Id,
		Controller:   phantom,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrObjectNotFound)
	require.Contains(t, err.Error(), phantom)

	stored, found := f.k.GetAllocation(f.ctx, f.alloc.Id)
	require.True(t, found)
	require.Equal(t, f.owner.Id, stored.Controller,
		"a refused transfer must leave the allocation where it was")

	require.Equal(t, types.Permissionless,
		f.k.GetPermissionsByBytes(f.ctx, keeperlib.GetObjectPermissionIDBytes(f.alloc.Id, phantom)),
		"a refused transfer must not leave a permission row keyed to a player that does not exist")
}

// TestAllocationTransferPreservesCreatorDeleteFallback characterises the
// permission handoff so a future reader does not tidy it into a bug.
//
// AllocationCreate splits the bits on purpose: a third-party controller gets
// PermAllocationConnection | PermAdmin, while the creator keeps PermDelete, plus
// PermUpdate when the allocation is dynamic. Because msg.Controller defaults to
// the creator, the common allocation has one row holding all of it, and
// AllocationTransfer removing exactly one bit is what hands over the right to
// connect the allocation without taking the creator's control of it away.
// PermDelete in particular is the fallback AllocationDelete tries when the
// source-side PermSourceAllocation check fails, so clearing the row would remove a
// documented path.
func TestAllocationTransferPreservesCreatorDeleteFallback(t *testing.T) {
	f := setupTransferFixture(t, "transfer-handoff-source", "alloc_handoff_ownr_1")

	recipientAcc := sdk.AccAddress("alloc_handoff_recip1")
	recipient := types.Player{
		Creator:        recipientAcc.String(),
		PrimaryAddress: recipientAcc.String(),
	}
	recipient = testAppendPlayer(f.k, f.ctx, recipient)

	// What AllocationCreate grants when the creator is also the controller.
	testPermissionAdd(f.k, f.ctx, f.ownerId,
		types.PermUpdate|types.PermDelete|types.PermAllocationConnection|types.PermAdmin)

	_, err := f.ms.AllocationTransfer(f.ctx, &types.MsgAllocationTransfer{
		Creator:      f.owner.Creator,
		AllocationId: f.alloc.Id,
		Controller:   recipient.Id,
	})
	require.NoError(t, err)

	stored, found := f.k.GetAllocation(f.ctx, f.alloc.Id)
	require.True(t, found)
	require.Equal(t, recipient.Id, stored.Controller)

	creatorPerms := f.k.GetPermissionsByBytes(f.ctx, f.ownerId)
	require.NotZero(t, creatorPerms&types.PermDelete,
		"the creator keeps PermDelete, which is the AllocationDelete fallback")
	require.NotZero(t, creatorPerms&types.PermUpdate,
		"the creator keeps PermUpdate")
	require.Zero(t, creatorPerms&types.PermAllocationConnection,
		"the right to connect the allocation moves to the recipient")

	require.NotZero(t,
		f.k.GetPermissionsByBytes(f.ctx, keeperlib.GetObjectPermissionIDBytes(f.alloc.Id, recipient.Id))&types.PermAllocationConnection,
		"the recipient can connect the allocation they now control")
}
