package keeper

import (
	"context"
	abci "github.com/tendermint/tendermint/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeginBlocker will persist the current header and validator set as a historical entry
// and prune the oldest entry based on the HistoricalEntries parameter
func (k *Keeper) BeginBlocker(ctx sdk.Context) {
    k.UpdateSubstationStatus(sdk.UnwrapSDKContext(ctx))
}

// Called every block, update validator set
func (k *Keeper) EndBlocker(ctx context.Context) ([]abci.ValidatorUpdate, error) {

	return []abci.ValidatorUpdate{}, nil
}