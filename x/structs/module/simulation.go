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
	OpWeightMsgStructBuildInitiate        = "op_weight_msg_struct_build_initiate"
	OpWeightMsgStructMove                 = "op_weight_msg_struct_move"
	OpWeightMsgGuildCreate                = "op_weight_msg_guild_create"
	OpWeightMsgGuildBankMint              = "op_weight_msg_guild_bank_mint"
	OpWeightMsgGuildBankRedeem            = "op_weight_msg_guild_bank_redeem"
	OpWeightMsgGuildBankConfiscateAndBurn = "op_weight_msg_guild_bank_confiscate_and_burn"
	OpWeightMsgAddressRegister            = "op_weight_msg_address_register"
	OpWeightMsgPlayerSend                 = "op_weight_msg_player_send"
	OpWeightMsgGuildMembershipRequest     = "op_weight_msg_guild_membership_request"
	OpWeightMsgGuildMembershipJoin        = "op_weight_msg_guild_membership_join"
	OpWeightMsgPlanetExplore              = "op_weight_msg_planet_explore"
	OpWeightMsgReactorInfuse              = "op_weight_msg_reactor_infuse"
	OpWeightCommandShipBuildInitiate      = "op_weight_command_ship_build_initiate"
	OpWeightCommandShipBuildComplete      = "op_weight_command_ship_build_complete"
	OpWeightGiftUalpha                    = "op_weight_gift_ualpha"
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

// WeightedOperations returns all the structs module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgStructBuildInitiate int
	simState.AppParams.GetOrGenerate(OpWeightMsgStructBuildInitiate, &weightMsgStructBuildInitiate, nil,
		func(_ *rand.Rand) {
			weightMsgStructBuildInitiate = 100
		},
	)

	var weightMsgStructMove int
	simState.AppParams.GetOrGenerate(OpWeightMsgStructMove, &weightMsgStructMove, nil,
		func(_ *rand.Rand) {
			weightMsgStructMove = 50
		},
	)

	var weightMsgGuildCreate int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildCreate, &weightMsgGuildCreate, nil,
		func(_ *rand.Rand) {
			weightMsgGuildCreate = 20
		},
	)

	var weightMsgGuildBankMint int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildBankMint, &weightMsgGuildBankMint, nil,
		func(_ *rand.Rand) {
			weightMsgGuildBankMint = 30
		},
	)

	var weightMsgGuildBankRedeem int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildBankRedeem, &weightMsgGuildBankRedeem, nil,
		func(_ *rand.Rand) {
			weightMsgGuildBankRedeem = 25
		},
	)

	var weightMsgGuildBankConfiscateAndBurn int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildBankConfiscateAndBurn, &weightMsgGuildBankConfiscateAndBurn, nil,
		func(_ *rand.Rand) {
			weightMsgGuildBankConfiscateAndBurn = 10
		},
	)

	var weightMsgAddressRegister int
	simState.AppParams.GetOrGenerate(OpWeightMsgAddressRegister, &weightMsgAddressRegister, nil,
		func(_ *rand.Rand) {
			weightMsgAddressRegister = 15
		},
	)

	var weightMsgPlayerSend int
	simState.AppParams.GetOrGenerate(OpWeightMsgPlayerSend, &weightMsgPlayerSend, nil,
		func(_ *rand.Rand) {
			weightMsgPlayerSend = 40
		},
	)

	var weightMsgGuildMembershipRequest int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildMembershipRequest, &weightMsgGuildMembershipRequest, nil,
		func(_ *rand.Rand) {
			weightMsgGuildMembershipRequest = 15
		},
	)

	var weightMsgGuildMembershipJoin int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildMembershipJoin, &weightMsgGuildMembershipJoin, nil,
		func(_ *rand.Rand) {
			weightMsgGuildMembershipJoin = 15
		},
	)

	var weightMsgPlanetExplore int
	simState.AppParams.GetOrGenerate(OpWeightMsgPlanetExplore, &weightMsgPlanetExplore, nil,
		func(_ *rand.Rand) {
			weightMsgPlanetExplore = 30
		},
	)

	var weightMsgReactorInfuse int
	simState.AppParams.GetOrGenerate(OpWeightMsgReactorInfuse, &weightMsgReactorInfuse, nil,
		func(_ *rand.Rand) {
			weightMsgReactorInfuse = 35
		},
	)

	var weightCommandShipBuildInitiate int
	simState.AppParams.GetOrGenerate(OpWeightCommandShipBuildInitiate, &weightCommandShipBuildInitiate, nil,
		func(_ *rand.Rand) {
			weightCommandShipBuildInitiate = 25
		},
	)

	var weightCommandShipBuildComplete int
	simState.AppParams.GetOrGenerate(OpWeightCommandShipBuildComplete, &weightCommandShipBuildComplete, nil,
		func(_ *rand.Rand) {
			weightCommandShipBuildComplete = 20
		},
	)

	var weightGiftUalpha int
	simState.AppParams.GetOrGenerate(OpWeightGiftUalpha, &weightGiftUalpha, nil,
		func(_ *rand.Rand) {
			weightGiftUalpha = 50
		},
	)

	operations = append(operations,
		simulation.NewWeightedOperation(
			weightMsgStructBuildInitiate,
			structssimulation.SimulateMsgStructBuildInitiate(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgStructMove,
			structssimulation.SimulateMsgStructMove(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgGuildCreate,
			structssimulation.SimulateMsgGuildCreate(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgGuildBankMint,
			structssimulation.SimulateMsgGuildBankMint(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgGuildBankRedeem,
			structssimulation.SimulateMsgGuildBankRedeem(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgGuildBankConfiscateAndBurn,
			structssimulation.SimulateMsgGuildBankConfiscateAndBurn(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgAddressRegister,
			structssimulation.SimulateMsgAddressRegister(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgPlayerSend,
			structssimulation.SimulateMsgPlayerSend(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgGuildMembershipRequest,
			structssimulation.SimulateMsgGuildMembershipRequest(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgGuildMembershipJoin,
			structssimulation.SimulateMsgGuildMembershipJoin(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgPlanetExplore,
			structssimulation.SimulateMsgPlanetExplore(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgReactorInfuse,
			structssimulation.SimulateMsgReactorInfuse(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightCommandShipBuildInitiate,
			structssimulation.SimulateCommandShipBuildInitiate(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightCommandShipBuildComplete,
			structssimulation.SimulateCommandShipBuildComplete(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightGiftUalpha,
			structssimulation.SimulateGiftUalpha(am.keeper, am.accountKeeper, am.bankKeeper),
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
