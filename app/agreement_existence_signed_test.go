package app_test

import (
	"math/rand"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stretchr/testify/require"

	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Registration really grants object permission on the player's ID. Supplying
// that ID where an agreement belongs must not let the permission row authorize
// a phantom agreement. Exercise BaseApp too: a recovered panic is not validation.
func TestSignedAgreementMutationsRejectNonAgreements(t *testing.T) {
	for _, action := range []string{"close", "increase", "decrease", "duration"} {
		for _, target := range []string{"player id with real grant", "missing agreement"} {
			t.Run(action+"/"+target, func(t *testing.T) {
				bApp, ctx := setupJailGateApp(t)
				key := secp256k1.GenPrivKey()
				address := sdk.AccAddress(key.PubKey().Address())
				cc := bApp.StructsKeeper.NewCurrentContext(ctx)
				actor := cc.UpsertPlayer(address.String())
				playerID := actor.ID()
				cc.CommitAll()
				require.Equal(t, types.PermAll, bApp.StructsKeeper.GetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(playerID, playerID)))
				id := playerID
				if target == "missing agreement" {
					id = "11-999999"
				}
				var msg sdk.Msg
				switch action {
				case "close":
					msg = &types.MsgAgreementClose{Creator: address.String(), AgreementId: id}
				case "increase":
					msg = &types.MsgAgreementCapacityIncrease{Creator: address.String(), AgreementId: id, CapacityIncrease: 1}
				case "decrease":
					msg = &types.MsgAgreementCapacityDecrease{Creator: address.String(), AgreementId: id, CapacityDecrease: 1}
				case "duration":
					msg = &types.MsgAgreementDurationIncrease{Creator: address.String(), AgreementId: id, DurationIncrease: 1}
				}
				signer := bApp.AccountKeeper.GetAccount(ctx, address)
				config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
				tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, []sdk.Msg{msg}, nil,
					ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
				require.NoError(t, err)
				wire, err := config.TxEncoder()(tx)
				require.NoError(t, err)
				block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
				require.NoError(t, err)
				require.Len(t, block.TxResults, 1)
				result := block.TxResults[0]
				require.NotZero(t, result.Code, result.Log)
				require.Contains(t, result.Log, types.NewObjectNotFoundError("agreement", id).Error())
				require.NotContains(t, result.Log, "panic")
				current := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
				require.Empty(t, bApp.StructsKeeper.GetAllAgreement(current))
				require.Empty(t, bApp.StructsKeeper.GetAllProvider(current))
				require.Equal(t, types.PermAll, bApp.StructsKeeper.GetPermissionsByBytes(current, keeperlib.GetObjectPermissionIDBytes(playerID, playerID)))
				for _, event := range result.Events {
					require.NotContains(t, []string{"coin_spent", "coin_received", "structs.structs.EventAgreement", "structs.structs.EventDelete"}, event.Type)
				}
			})
		}
	}
}
