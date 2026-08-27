package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

/* Regression suite for permission records written against players who do not
 * exist.
 *
 * The three permission-on-object handlers put msg.PlayerId straight into a KV
 * key. Nothing resolved it: the object check speaks to the object, and
 * PermissionCheck's owner shortcut passes an owner acting on their own object
 * whatever target they name. So any string at all could become half of a
 * permanent key.
 *
 * This is the phantom-cache rule reached by a road that has no getter on it. The
 * other handlers hand a message id to cc.GetPlayer, which allocates a cache that
 * cannot report existence; these never resolved the id at all, which is why the
 * arch test - looking for a cc.GetPlayer call to object to - saw nothing.
 *
 * Revoke was the sharpest of the three because it needs no real permission to
 * write: a bit that was never held is removed to zero, and zero used to be
 * committed as eight bytes rather than as an absence.
 */

type permissionTargetFixture struct {
	k        keeperlib.Keeper
	ms       types.MsgServer
	ctx      sdk.Context
	owner    types.Player
	target   types.Player
	objectId string
}

func setupPermissionTargetFixture(t *testing.T) *permissionTargetFixture {
	t.Helper()

	k, ms, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	ownerAcc := sdk.AccAddress("permowner1234567890123456789012345")
	owner := testAppendPlayer(k, ctx, types.Player{Creator: ownerAcc.String(), PrimaryAddress: ownerAcc.String()})

	targetAcc := sdk.AccAddress("permtarget123456789012345678901234")
	target := testAppendPlayer(k, ctx, types.Player{Creator: targetAcc.String(), PrimaryAddress: targetAcc.String()})

	structObj := testAppendStruct(k, ctx, types.Struct{Creator: owner.Creator, Owner: owner.Id, Type: 1})
	testPermissionAdd(k, ctx, keeperlib.GetObjectPermissionIDBytes(structObj.Id, owner.Id), types.PermAll)

	return &permissionTargetFixture{k: k, ms: ms, ctx: ctx, owner: owner, target: target, objectId: structObj.Id}
}

// permissionRowCount counts what is actually on disk under the permission
// prefix, which is the quantity an attacker was growing.
func permissionRowCount(t *testing.T, f *permissionTargetFixture) int {
	t.Helper()
	return len(f.k.GetAllPermissionExport(f.ctx))
}

/* TestPermissionRevokeOnObject_RejectsPhantomTarget is the direct regression:
 * a revoke naming a player that does not exist must be refused, and must leave
 * nothing behind.
 */
func TestPermissionRevokeOnObject_RejectsPhantomTarget(t *testing.T) {
	f := setupPermissionTargetFixture(t)
	before := permissionRowCount(t, f)

	_, err := f.ms.PermissionRevokeOnObject(f.ctx, &types.MsgPermissionRevokeOnObject{
		Creator:     f.owner.Creator,
		ObjectId:    f.objectId,
		PlayerId:    "not-a-player",
		Permissions: uint64(types.PermPlay),
	})
	require.Error(t, err, "a permission target that does not exist must be refused")
	require.Equal(t, before, permissionRowCount(t, f),
		"a refused revoke must not have written a permission row")
}

// The same target validation on the other two handlers of the family. The report
// named only revoke; all three build the key the same way.
func TestPermissionGrantAndSetOnObject_RejectPhantomTarget(t *testing.T) {
	f := setupPermissionTargetFixture(t)
	before := permissionRowCount(t, f)

	_, err := f.ms.PermissionGrantOnObject(f.ctx, &types.MsgPermissionGrantOnObject{
		Creator:     f.owner.Creator,
		ObjectId:    f.objectId,
		PlayerId:    "not-a-player",
		Permissions: uint64(types.PermPlay),
	})
	require.Error(t, err, "granting to a player that does not exist must be refused")

	_, err = f.ms.PermissionSetOnObject(f.ctx, &types.MsgPermissionSetOnObject{
		Creator:     f.owner.Creator,
		ObjectId:    f.objectId,
		PlayerId:    "not-a-player",
		Permissions: uint64(types.PermPlay),
	})
	require.Error(t, err, "setting on a player that does not exist must be refused")

	require.Equal(t, before, permissionRowCount(t, f))
}

/* TestPermissionRevokeOnObject_RepeatedNoOpsWriteNothing is the state-growth
 * property, and it holds independently of the target check: even a real target
 * must not accumulate rows for revokes that revoke nothing.
 */
func TestPermissionRevokeOnObject_RepeatedNoOpsWriteNothing(t *testing.T) {
	f := setupPermissionTargetFixture(t)
	before := permissionRowCount(t, f)

	for i := 0; i < 25; i++ {
		_, err := f.ms.PermissionRevokeOnObject(f.ctx, &types.MsgPermissionRevokeOnObject{
			Creator:     f.owner.Creator,
			ObjectId:    f.objectId,
			PlayerId:    f.target.Id,
			Permissions: uint64(types.PermPlay),
		})
		require.NoError(t, err, "revoking a bit the target never had is not an error")
	}

	require.Equal(t, before, permissionRowCount(t, f),
		"revoking nothing, repeatedly, must leave no rows behind")
}

/* TestPermissionRevoke_ClearsRatherThanStoringZero pins the other half. A row
 * whose bits all go away must be deleted: a stored zero says exactly what an
 * absent row says, and costs eight bytes of consensus state forever.
 */
func TestPermissionRevoke_ClearsRatherThanStoringZero(t *testing.T) {
	f := setupPermissionTargetFixture(t)

	targetPermissionId := keeperlib.GetObjectPermissionIDBytes(f.objectId, f.target.Id)
	testPermissionAdd(f.k, f.ctx, targetPermissionId, types.PermPlay)

	withGrant := permissionRowCount(t, f)

	_, err := f.ms.PermissionRevokeOnObject(f.ctx, &types.MsgPermissionRevokeOnObject{
		Creator:     f.owner.Creator,
		ObjectId:    f.objectId,
		PlayerId:    f.target.Id,
		Permissions: uint64(types.PermPlay),
	})
	require.NoError(t, err)

	require.Equal(t, withGrant-1, permissionRowCount(t, f),
		"a permission record emptied by a revoke must be removed, not stored as zero")
	require.Equal(t, types.Permissionless, f.k.GetPermissionsByBytes(f.ctx, targetPermissionId))
}

// A real revoke that leaves some bits behind must still write the record.
func TestPermissionRevoke_PartialRevokeStillWrites(t *testing.T) {
	f := setupPermissionTargetFixture(t)

	targetPermissionId := keeperlib.GetObjectPermissionIDBytes(f.objectId, f.target.Id)
	testPermissionAdd(f.k, f.ctx, targetPermissionId, types.PermPlay|types.PermUpdate)

	_, err := f.ms.PermissionRevokeOnObject(f.ctx, &types.MsgPermissionRevokeOnObject{
		Creator:     f.owner.Creator,
		ObjectId:    f.objectId,
		PlayerId:    f.target.Id,
		Permissions: uint64(types.PermPlay),
	})
	require.NoError(t, err)

	require.Equal(t, types.PermUpdate, f.k.GetPermissionsByBytes(f.ctx, targetPermissionId),
		fmt.Sprintf("the surviving bit must remain on %s", targetPermissionId))
}
