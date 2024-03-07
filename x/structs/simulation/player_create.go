package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"structs/x/structs/keeper"
	"structs/x/structs/types"
)

func SimulateMsgGuildJoin(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgGuildJoin{
			Creator: simAccount.Address.String(),
		}

		// TODO: Handling the GuildJoin simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "GuildJoin simulation not implemented"), nil, nil
	}
}
