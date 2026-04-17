package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

// emitUGCModerationEventIfActorIsNotOwner emits an audit event when an
// active player updates UGC (name or pfp) for an object whose owner is a
// different player. The event only fires when actor != target owner so
// self-updates do not pollute the moderation log.
//
// `target` is the PermissionedObject being edited (player, guild, planet,
// substation, ...). `field` is one of types.UGCField{Name,Pfp}. `oldValue`
// is captured by the caller BEFORE the mutation so the event can show the
// before/after pair.
func emitUGCModerationEventIfActorIsNotOwner(
	ctx sdk.Context,
	target PermissionedObject,
	actor *PlayerCache,
	field, oldValue, newValue string,
) {
	if target == nil || actor == nil {
		return
	}
	if target.GetOwnerId() == actor.GetPlayerId() {
		return
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUGCModerated,
			sdk.NewAttribute(types.AttributeKeyUGCActorPlayerId, actor.GetPlayerId()),
			sdk.NewAttribute(types.AttributeKeyUGCActorAddress, actor.GetActiveAddress()),
			sdk.NewAttribute(types.AttributeKeyUGCTargetObjectId, target.ID()),
			sdk.NewAttribute(types.AttributeKeyUGCTargetOwnerId, target.GetOwnerId()),
			sdk.NewAttribute(types.AttributeKeyUGCField, field),
			sdk.NewAttribute(types.AttributeKeyUGCOldValue, oldValue),
			sdk.NewAttribute(types.AttributeKeyUGCNewValue, newValue),
		),
	)
}
