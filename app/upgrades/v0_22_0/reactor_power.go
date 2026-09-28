package v0_22_0

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/app/upgrades"
	"structs/x/structs/types"
)

// MigrateReactorPower runs after asset relocation and rebuilds the final flat
// backing under bonded/jailed eligibility. Any capacity loss owes normal cascade.
func MigrateReactorPower(ctx context.Context, keepers *upgrades.Keepers) error {
	cacheCtx, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	ctx = cacheCtx
	k := keepers.StructsKeeper
	for _, row := range k.GetAllInfusion(ctx) {
		if row.DestinationType != types.ObjectType_reactor {
			continue
		}
		reactor, found := k.GetReactor(ctx, row.DestinationId)
		if !found {
			return fmt.Errorf("infusion references missing reactor %s", row.DestinationId)
		}
		validator, err := sdk.ValAddressFromBech32(reactor.Validator)
		if err != nil {
			return err
		}
		address, err := sdk.AccAddressFromBech32(row.Address)
		if err != nil {
			return err
		}
		if err := k.ReconcileExistingInfusionForDelegation(ctx, address, validator); err != nil {
			return err
		}
	}
	k.RebuildGeneratorPowerQueue(ctx)
	write()
	return nil
}

// Install replacement parameters before dependent hooks and power reconstruction.
func MigrateReactorParameters(ctx context.Context, keepers *upgrades.Keepers) error {
	p := keepers.StructsKeeper.GetParams(ctx).WithReactorDefaults()
	if err := p.ValidateReactorParams(); err != nil {
		return err
	}
	return keepers.StructsKeeper.SetParams(ctx, p)
}
