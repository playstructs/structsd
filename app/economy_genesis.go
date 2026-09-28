package app

import (
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"structs/x/structs/types"
)

// Gentxs must share BaseApp's initialization cache: ExecuteGenesisTx obtains
// its context from BaseApp. Structs restores derived power from staking state
// after the staking and genutil modules have initialized.
func (app *App) initChainerWithEconomyGenesis(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	if req.InitialHeight < 0 {
		return nil, fmt.Errorf("negative genesis initial height")
	}
	var modules map[string]json.RawMessage
	if err := json.Unmarshal(req.AppStateBytes, &modules); err != nil {
		return nil, err
	}
	var genesis types.GenesisState
	if raw := modules[types.ModuleName]; raw != nil {
		if err := app.appCodec.UnmarshalJSON(raw, &genesis); err != nil {
			return nil, err
		}
	}
	var genutilGenesis genutiltypes.GenesisState
	if raw := modules[genutiltypes.ModuleName]; raw != nil {
		if err := app.appCodec.UnmarshalJSON(raw, &genutilGenesis); err != nil {
			return nil, err
		}
	}
	var stakingGenesis stakingtypes.GenesisState
	if raw := modules[stakingtypes.ModuleName]; raw != nil {
		if err := app.appCodec.UnmarshalJSON(raw, &stakingGenesis); err != nil {
			return nil, err
		}
	}
	lastID, err := stakingGenesisLastID(stakingGenesis, genesis.StakingUnbondingId)
	if err != nil {
		return nil, err
	}
	cached, write := ctx, func() {}
	if len(genutilGenesis.GenTxs) == 0 {
		cached, write = ctx.CacheContext()
	}
    genesis.Params = genesis.Params.WithReactorDefaults()
    if err := genesis.Params.ValidateReactorParams(); err != nil { return nil, err }
    if err := app.StructsKeeper.SetParams(cached, genesis.Params); err != nil { return nil, err }
	app.setStakingUnbondingID(cached, lastID)
	// With gentxs, a failure retains the SDK's normal InitChain semantics: the
	// node must abort initialization; nothing is persisted until the first block
	// commits. Retrying InitChain creates a new BaseApp initialization cache.

	response, err := app.App.InitChainer(cached, req)
	if err != nil {
		return nil, err
	}
	if err := app.restoreStakingOperationIndices(cached, stakingGenesis); err != nil {
		return nil, err
	}
	write()
	return response, nil
}
