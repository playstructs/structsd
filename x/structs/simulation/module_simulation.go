package simulation

import (
	"structs/x/structs/keeper"
	"structs/x/structs/types"

	"github.com/cosmos/cosmos-sdk/x/simulation"
)

// WeightedOperations returns all the operations from the module with their respective weights
func WeightedOperations(
	appParams map[string]interface{},
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simulation.WeightedOperations {
	var weightMsgStructBuildInitiate int = 100
	var weightMsgStructMove int = 100

	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			weightMsgStructBuildInitiate,
			SimulateMsgStructBuildInitiate(k, ak, bk),
		),
		simulation.NewWeightedOperation(
			weightMsgStructMove,
			SimulateMsgStructMove(k, ak, bk),
		),
	}
}
