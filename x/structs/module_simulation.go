package structs

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
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
	_ = sample.AccAddress
	_ = structssimulation.FindAccount
	_ = rand.Rand{}
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
)

const (
	opWeightMsgReactorAllocationCreate = "op_weight_msg_reactor_allocation_create"
	// TODO: Determine the simulation weight value
	defaultWeightMsgReactorAllocationCreate int = 100

	opWeightMsgSubstationCreate = "op_weight_msg_substation_create"
	// TODO: Determine the simulation weight value
	defaultWeightMsgSubstationCreate int = 100

	opWeightMsgSubstationDelete = "op_weight_msg_substation_delete"
	// TODO: Determine the simulation weight value
	defaultWeightMsgSubstationDelete int = 100

	opWeightMsgSubstationAllocationCreate = "op_weight_msg_substation_allocation_create"
	// TODO: Determine the simulation weight value
	defaultWeightMsgSubstationAllocationCreate int = 100

	opWeightMsgSubstationAllocationConnect = "op_weight_msg_substation_allocation_connect"
	// TODO: Determine the simulation weight value
	defaultWeightMsgSubstationAllocationConnect int = 100

	opWeightMsgSubstationAllocationDisconnect = "op_weight_msg_substation_allocation_disconnect"
	// TODO: Determine the simulation weight value
	defaultWeightMsgSubstationAllocationDisconnect int = 100

	opWeightMsgSubstationPlayerConnect = "op_weight_msg_substation_player_connect"
	// TODO: Determine the simulation weight value
	defaultWeightMsgSubstationPlayerConnect int = 100

	opWeightMsgSubstationPlayerDisconnect = "op_weight_msg_substation_player_disconnect"
	// TODO: Determine the simulation weight value
	defaultWeightMsgSubstationPlayerDisconnect int = 100

	opWeightMsgGuildCreate = "op_weight_msg_guild_create"
	// TODO: Determine the simulation weight value
	defaultWeightMsgGuildCreate int = 100

	opWeightMsgGuildJoinProxy = "op_weight_msg_player_create_proxy"
	// TODO: Determine the simulation weight value
	defaultWeightMsgGuildJoinProxy int = 100

	opWeightMsgGuildJoin = "op_weight_msg_player_create"
	// TODO: Determine the simulation weight value
	defaultWeightMsgGuildJoin int = 100

	opWeightMsgAddressRegister = "op_weight_msg_address_register"
	// TODO: Determine the simulation weight value
	defaultWeightMsgAddressRegister int = 100

	opWeightMsgAddressRevoke = "op_weight_msg_address_revoke"
	// TODO: Determine the simulation weight value
	defaultWeightMsgAddressRevoke int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module
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

// ProposalContents doesn't return any content functions for governance proposals
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgGuildCreate,
			defaultWeightMsgGuildCreate,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				structssimulation.SimulateMsgGuildCreate(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgGuildJoinProxy,
			defaultWeightMsgGuildJoinProxy,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				structssimulation.SimulateMsgGuildJoinProxy(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgGuildJoin,
			defaultWeightMsgGuildJoin,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				structssimulation.SimulateMsgGuildJoin(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgAddressRegister,
			defaultWeightMsgAddressRegister,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				structssimulation.SimulateMsgAddressRegister(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgAddressRevoke,
			defaultWeightMsgAddressRevoke,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				structssimulation.SimulateMsgAddressRevoke(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)



	var weightMsgSubstationCreate int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgSubstationCreate, &weightMsgSubstationCreate, nil,
		func(_ *rand.Rand) {
			weightMsgSubstationCreate = defaultWeightMsgSubstationCreate
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSubstationCreate,
		structssimulation.SimulateMsgSubstationCreate(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgSubstationDelete int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgSubstationDelete, &weightMsgSubstationDelete, nil,
		func(_ *rand.Rand) {
			weightMsgSubstationDelete = defaultWeightMsgSubstationDelete
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSubstationDelete,
		structssimulation.SimulateMsgSubstationDelete(am.accountKeeper, am.bankKeeper, am.keeper),
	))


	var weightMsgSubstationAllocationConnect int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgSubstationAllocationConnect, &weightMsgSubstationAllocationConnect, nil,
		func(_ *rand.Rand) {
			weightMsgSubstationAllocationConnect = defaultWeightMsgSubstationAllocationConnect
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSubstationAllocationConnect,
		structssimulation.SimulateMsgSubstationAllocationConnect(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgSubstationAllocationDisconnect int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgSubstationAllocationDisconnect, &weightMsgSubstationAllocationDisconnect, nil,
		func(_ *rand.Rand) {
			weightMsgSubstationAllocationDisconnect = defaultWeightMsgSubstationAllocationDisconnect
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSubstationAllocationDisconnect,
		structssimulation.SimulateMsgSubstationAllocationDisconnect(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgSubstationPlayerConnect int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgSubstationPlayerConnect, &weightMsgSubstationPlayerConnect, nil,
		func(_ *rand.Rand) {
			weightMsgSubstationPlayerConnect = defaultWeightMsgSubstationPlayerConnect
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSubstationPlayerConnect,
		structssimulation.SimulateMsgSubstationPlayerConnect(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgSubstationPlayerDisconnect int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgSubstationPlayerDisconnect, &weightMsgSubstationPlayerDisconnect, nil,
		func(_ *rand.Rand) {
			weightMsgSubstationPlayerDisconnect = defaultWeightMsgSubstationPlayerDisconnect
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSubstationPlayerDisconnect,
		structssimulation.SimulateMsgSubstationPlayerDisconnect(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgGuildCreate int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgGuildCreate, &weightMsgGuildCreate, nil,
		func(_ *rand.Rand) {
			weightMsgGuildCreate = defaultWeightMsgGuildCreate
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgGuildCreate,
		structssimulation.SimulateMsgGuildCreate(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgGuildJoinProxy int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgGuildJoinProxy, &weightMsgGuildJoinProxy, nil,
		func(_ *rand.Rand) {
			weightMsgGuildJoinProxy = defaultWeightMsgGuildJoinProxy
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgGuildJoinProxy,
		structssimulation.SimulateMsgGuildJoinProxy(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgGuildJoin int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgGuildJoin, &weightMsgGuildJoin, nil,
		func(_ *rand.Rand) {
			weightMsgGuildJoin = defaultWeightMsgGuildJoin
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgGuildJoin,
		structssimulation.SimulateMsgGuildJoin(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgAddressRegister int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgAddressRegister, &weightMsgAddressRegister, nil,
		func(_ *rand.Rand) {
			weightMsgAddressRegister = defaultWeightMsgAddressRegister
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgAddressRegister,
		structssimulation.SimulateMsgAddressRegister(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgAddressRevoke int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgAddressRevoke, &weightMsgAddressRevoke, nil,
		func(_ *rand.Rand) {
			weightMsgAddressRevoke = defaultWeightMsgAddressRevoke
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgAddressRevoke,
		structssimulation.SimulateMsgAddressRevoke(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
