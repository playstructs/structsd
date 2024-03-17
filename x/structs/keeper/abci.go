package keeper

import (
	"context"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeginBlocker will persist the current header and validator set as a historical entry
// and prune the oldest entry based on the HistoricalEntries parameter
func (k *Keeper) BeginBlocker(ctx context.Context) {

}

// Called every block, update validator set
func (k *Keeper) EndBlocker(ctx context.Context) ([]abci.ValidatorUpdate, error) {

	/* Cascade all the possible failures across the grid
	 *
	 * This will mean that there will be some cases in which
	 * devices have one last block of power before shutting down
	 * but I think that's ok. We'll see how it goes in practice.
	 */
	k.GridCascade(sdk.UnwrapSDKContext(ctx))

	return []abci.ValidatorUpdate{}, nil
}
