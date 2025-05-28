package structs

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"structs/testutil/sample"
	structssimulation "structs/x/structs/simulation"
	"structs/x/structs/types"
)

// avoid unused import issue
var (
	_ = structssimulation.FindAccount
	_ = rand.Rand{}
	_ = sample.AccAddress
	_ = sdk.AccAddress{}
	_ = simulation.MsgEntryKind
)

const (
	OpWeightMsgCreateStruct = "op_weight_msg_create_struct"
	OpWeightMsgUpdateStruct = "op_weight_msg_update_struct"
	OpWeightMsgDeleteStruct = "op_weight_msg_delete_struct"
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	structsGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		PortId: types.PortID,
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&structsGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgCreateStruct int
	simState.AppParams.GetOrGenerate(OpWeightMsgCreateStruct, &weightMsgCreateStruct, nil,
		func(_ *rand.Rand) {
			weightMsgCreateStruct = 100
		},
	)

	var weightMsgUpdateStruct int
	simState.AppParams.GetOrGenerate(OpWeightMsgUpdateStruct, &weightMsgUpdateStruct, nil,
		func(_ *rand.Rand) {
			weightMsgUpdateStruct = 50
		},
	)

	var weightMsgDeleteStruct int
	simState.AppParams.GetOrGenerate(OpWeightMsgDeleteStruct, &weightMsgDeleteStruct, nil,
		func(_ *rand.Rand) {
			weightMsgDeleteStruct = 30
		},
	)

	operations = append(operations,
		simulation.NewWeightedOperation(
			weightMsgCreateStruct,
			structssimulation.SimulateMsgCreateStruct(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgUpdateStruct,
			structssimulation.SimulateMsgUpdateStruct(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgDeleteStruct,
			structssimulation.SimulateMsgDeleteStruct(am.keeper, am.accountKeeper, am.bankKeeper),
		),
	)

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
