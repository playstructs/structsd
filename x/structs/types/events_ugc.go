package types

// UGC moderation event constants. These are emitted as standard untyped
// Cosmos events (sdk.NewEvent) whenever an actor updates the name or pfp
// of an object they do not directly own -- i.e. when the change went
// through the guild UGC moderation permission path rather than the
// owner's own update permission. Indexers can subscribe to these to
// surface a moderation/audit log.
const (
	EventTypeUGCModerated = "ugc_moderated"

	AttributeKeyUGCActorPlayerId  = "actor_player_id"
	AttributeKeyUGCActorAddress   = "actor_address"
	AttributeKeyUGCTargetObjectId = "target_object_id"
	AttributeKeyUGCTargetOwnerId  = "target_owner_player_id"
	AttributeKeyUGCField          = "field"
	AttributeKeyUGCOldValue       = "old_value"
	AttributeKeyUGCNewValue       = "new_value"

	UGCFieldName = "name"
	UGCFieldPfp  = "pfp"
)
