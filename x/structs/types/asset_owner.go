package types

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseAssetOwnerID accepts only the canonical identity of a player or guild.
// It validates the namespace, not existence: callers must resolve the entity
// through CurrentContext before using its authority or creating a vault.
func ParseAssetOwnerID(id string) (ObjectType, uint64, error) {
	parts := strings.Split(id, "-")
	if len(parts) != 2 {
		return ObjectType(0), 0, NewObjectNotFoundError("asset_owner", id)
	}
	kind, kindErr := strconv.ParseUint(parts[0], 10, 32)
	index, indexErr := strconv.ParseUint(parts[1], 10, 64)
	if kindErr != nil || indexErr != nil || index == 0 ||
		(kind != uint64(ObjectType_player) && kind != uint64(ObjectType_guild)) ||
		id != fmt.Sprintf("%d-%d", kind, index) {
		return ObjectType(0), 0, NewObjectNotFoundError("asset_owner", id)
	}
	return ObjectType(kind), index, nil
}

func (a Infusion) ValidateAssetOwner() error {
	if a.OwnerId == "" {
		return nil
	}
	kind, _, err := ParseAssetOwnerID(a.OwnerId)
	if err != nil {
		return err
	}
	if kind == ObjectType_guild && a.PlayerId != "" {
		return fmt.Errorf("guild infusion cannot carry a player owner")
	}
	if kind == ObjectType_player && a.PlayerId != "" && a.PlayerId != a.OwnerId {
		return fmt.Errorf("infusion owner fields disagree")
	}
	return nil
}
