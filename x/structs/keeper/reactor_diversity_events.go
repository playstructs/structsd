package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// Genesis block replay replaces derived owner state, including owners whose
// last infusion vanished before export. The existing genesis replay obligation
// covers imports starting above block one. No runtime per-block scan is added.
func (k Keeper) EmitOwnerReactorDiversitySnapshot(ctx context.Context) error {
	owners, err := k.allOwnerReactorDiversity(ctx)
	if err != nil {
		return err
	}
	events := sdk.UnwrapSDKContext(ctx).EventManager()
	if err := events.EmitTypedEvent(&types.EventOwnerReactorDiversityReset{}); err != nil {
		return err
	}
	for i := range owners {
		if err := events.EmitTypedEvent(&types.EventOwnerReactorDiversity{Diversity: &owners[i]}); err != nil {
			return err
		}
	}
	return nil
}
