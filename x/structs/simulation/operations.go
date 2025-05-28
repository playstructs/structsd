package simulation

import (
	"math/rand"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"

	"structs/x/structs/keeper"
	"structs/x/structs/types"
)

// EncodingConfig specifies the concrete encoding types to use for a given app.
type EncodingConfig struct {
	InterfaceRegistry codectypes.InterfaceRegistry
	Codec             *codec.ProtoCodec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

var (
	// Create a test encoding config
	encodingConfig = makeTestEncodingConfig()
)

// makeTestEncodingConfig creates a test encoding config
func makeTestEncodingConfig() EncodingConfig {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	types.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)
	txConfig := tx.NewTxConfig(cdc, tx.DefaultSignModes)

	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             cdc,
		TxConfig:          txConfig,
		Amino:             codec.NewLegacyAmino(),
	}
}

// SimulateMsgStructBuildInitiate generates a MsgStructBuildInitiate with random values
func SimulateMsgStructBuildInitiate(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		msg := &types.MsgStructBuildInitiate{
			Creator:        simAccount.Address.String(),
			PlayerId:       simAccount.Address.String(),
			StructTypeId:   uint64(r.Int63n(5)),      // Random struct type
			OperatingAmbit: types.Ambit(r.Int63n(5)), // Random ambit
			Slot:           uint64(r.Int63n(10)),     // Random slot
		}

		// Create and deliver the transaction
		txBuilder := encodingConfig.TxConfig.NewTxBuilder()
		err := txBuilder.SetMsgs(msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, "struct_build_initiate", "failed to set message"), nil, err
		}

		// Set random fee
		fee := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(100)))
		txBuilder.SetFeeAmount(fee)
		txBuilder.SetGasLimit(200000)

		// Sign the transaction
		sigData := txsigning.SingleSignatureData{
			SignMode:  txsigning.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
			Signature: nil,
		}
		sig := txsigning.SignatureV2{
			PubKey:   simAccount.PubKey,
			Data:     &sigData,
			Sequence: 0,
		}
		err = txBuilder.SetSignatures(sig)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, "struct_build_initiate", "failed to set signature"), nil, err
		}

		// Deliver the transaction
		txBytes, err := encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, "struct_build_initiate", "failed to encode transaction"), nil, err
		}

		// Update the context with the transaction
		ctx = ctx.WithTxBytes(txBytes)

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.StructBuildInitiate(ctx, msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, "struct_build_initiate", "failed to execute message"), nil, err
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgStructMove generates a MsgStructMove with random values
func SimulateMsgStructMove(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		// Get a random struct from the store
		structs := k.GetAllStruct(ctx)
		if len(structs) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, "struct_move", "no structs found"), nil, nil
		}

		structToMove := structs[r.Intn(len(structs))]

		msg := &types.MsgStructMove{
			Creator:      simAccount.Address.String(),
			StructId:     structToMove.Id,
			LocationType: types.ObjectType(r.Int63n(5)), // Random location type
			Ambit:        types.Ambit(r.Int63n(5)),      // Random ambit
			Slot:         uint64(r.Int63n(10)),          // Random slot
		}

		// Create and deliver the transaction
		txBuilder := encodingConfig.TxConfig.NewTxBuilder()
		err := txBuilder.SetMsgs(msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, "struct_move", "failed to set message"), nil, err
		}

		// Set random fee
		fee := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(100)))
		txBuilder.SetFeeAmount(fee)
		txBuilder.SetGasLimit(200000)

		// Sign the transaction
		sigData := txsigning.SingleSignatureData{
			SignMode:  txsigning.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
			Signature: nil,
		}
		sig := txsigning.SignatureV2{
			PubKey:   simAccount.PubKey,
			Data:     &sigData,
			Sequence: 0,
		}
		err = txBuilder.SetSignatures(sig)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, "struct_move", "failed to set signature"), nil, err
		}

		// Deliver the transaction
		txBytes, err := encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, "struct_move", "failed to encode transaction"), nil, err
		}

		// Update the context with the transaction
		ctx = ctx.WithTxBytes(txBytes)

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.StructMove(ctx, msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, "struct_move", "failed to execute message"), nil, err
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}
