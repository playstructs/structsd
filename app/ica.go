package app

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	icamodule "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts"
	icagenesistypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/genesis/types"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
)

// Use the same defaults in the application and CLI genesis builder. ICA host
// dispatch does not run inner messages through the Structs ante chain. Hosting
// is unused and stays disabled until supported operations are explicitly reviewed.
type icaAppModule struct {
	icamodule.AppModule
}

func (icaAppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	state := icagenesistypes.DefaultGenesis()
	state.HostGenesisState.Params = icahosttypes.NewParams(false, []string{})
	return cdc.MustMarshalJSON(state)
}
