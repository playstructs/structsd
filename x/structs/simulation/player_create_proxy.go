package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"structs/x/structs/keeper"
	"structs/x/structs/types"
)

func SimulateMsgGuildJoinProxy(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)


		msg := &types.MsgGuildJoinProxy{
			Creator: simAccount.Address.String(),
		}

		// TODO: Handling the GuildJoinProxy simulation

        //return msg
		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "GuildJoinProxy simulation not implemented"), nil, nil
	}
}
