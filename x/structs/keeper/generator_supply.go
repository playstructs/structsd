package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// Generator output belongs to its infusion owner, so a generator may supply
// someone other than its operator. Do not recurse into the operator's player
// readiness here: a live generator can supply its own passive draw. Lifecycle
// status comes from the same CurrentContext, including uncommitted changes.
func (cc *CurrentContext) generatorInfusionEligible(row types.Infusion) bool {
	if !ObjectIdHasType(row.DestinationId, types.ObjectType_struct) {
		return false
	}
	structure := cc.GetStruct(row.DestinationId)
	if structure.CheckStruct() != nil || structure.IsDestroyed() || !structure.IsBuilt() || !structure.IsOnline() {
		return false
	}
	kind, found := cc.GetStructType(structure.GetTypeId())
	if !found {
		return false
	}
	switch kind.GetStructType().PowerGeneration {
	case types.TechPowerGeneration_smallGenerator, types.TechPowerGeneration_mediumGenerator, types.TechPowerGeneration_largeGenerator:
	default:
		return false
	}
	address, err := sdk.AccAddressFromBech32(row.Address)
	if err != nil {
		return false
	}
	owner, err := cc.GetAssetOwnerByAddress(address)
	return err == nil && owner.ID() == row.AssetOwnerID()
}
