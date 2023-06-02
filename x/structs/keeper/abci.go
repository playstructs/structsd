package keeper

import (
	"context"
	abci "github.com/tendermint/tendermint/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeginBlocker will persist the current header and validator set as a historical entry
// and prune the oldest entry based on the HistoricalEntries parameter
func (k *Keeper) BeginBlocker(ctx sdk.Context) {

}

// Called every block, update validator set
func (k *Keeper) EndBlocker(ctx context.Context) ([]abci.ValidatorUpdate, error) {
    k.UpdateSubstationStatus(sdk.UnwrapSDKContext(ctx))
	return []abci.ValidatorUpdate{}, nil
}