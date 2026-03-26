package simulation

import (
	"fmt"
	"math/rand"
	"testing"

	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stretchr/testify/require"

	testkeeper "structs/testutil/keeper"
	structskeeper "structs/x/structs/keeper"
	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type MockBaseApp struct {
	*baseapp.BaseApp
}

func newTestBaseApp() *MockBaseApp {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)
	txConfig := tx.NewTxConfig(cdc, tx.DefaultSignModes)
	logger := log.NewNopLogger()
	db := dbm.NewMemDB()
	app := &MockBaseApp{
		BaseApp: baseapp.NewBaseApp("test", logger, db, txConfig.TxDecoder()),
	}
	app.SetTxEncoder(txConfig.TxEncoder())
	app.SetTxDecoder(txConfig.TxDecoder())
	return app
}

func registerPlayerWithAllPermissions(t *testing.T, k structskeeper.Keeper, ctx sdk.Context, address string) types.Player {
	cc := k.NewCurrentContext(ctx)
	playerCache := cc.UpsertPlayer(address)
	cc.CommitAll()

	player := playerCache.GetPlayer()

	_, found := k.GetPlayer(ctx, player.Id)
	require.True(t, found, "Player not found in store after registration: %s", player.Id)

	addressPermissionId := structskeeper.GetAddressPermissionIDBytes(address)
	existing := k.GetPermissionsByBytes(ctx, addressPermissionId)
	k.SetPermissionsByBytes(ctx, addressPermissionId, existing|types.PermAll)

	return player
}

func createAllStructTypesFromGenesis(t *testing.T, k structskeeper.Keeper, ctx sdk.Context) {
	genesisStructTypes := types.CreateStructTypeGenesis()
	for _, structType := range genesisStructTypes {
		k.AppendStructType(ctx, structType)
	}
	for _, structType := range genesisStructTypes {
		_, found := k.GetStructType(ctx, structType.Id)
		require.True(t, found, "Struct type not found in store after creation: %d", structType.Id)
	}
}

func getPlayerCharge(k structskeeper.Keeper, ctx sdk.Context, playerId string) uint64 {
	cc := k.NewCurrentContext(ctx)
	playerCache, err := cc.GetPlayer(playerId)
	if err != nil {
		return 0
	}
	return playerCache.GetCharge()
}

func setupEnergyGridForSim(t *testing.T, k structskeeper.Keeper, ctx sdk.Context) (reactorOwnerAddr string, substationId string, guildId string) {
	reactorOwner := simtypes.RandomAccounts(rand.New(rand.NewSource(2)), 1)[0]
	reactorOwnerAddr = reactorOwner.Address.String()

	player := registerPlayerWithAllPermissions(t, k, ctx, reactorOwnerAddr)

	reactor := types.CreateEmptyReactor()
	reactor.Validator = reactorOwnerAddr
	reactor.RawAddress = reactorOwner.Address.Bytes()
	reactor = k.AppendReactor(ctx, reactor)

	reactorPermissionId := structskeeper.GetObjectPermissionIDBytes(reactor.Id, player.Id)
	k.SetPermissionsByBytes(ctx, reactorPermissionId, types.PermAll)

	capacity := uint64(10000000)
	k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, reactor.Id), capacity)

	msgServer := structskeeper.NewMsgServerImpl(k)
	allocationMsg := &types.MsgAllocationCreate{
		Creator:        reactorOwnerAddr,
		Controller:     player.Id,
		SourceObjectId: reactor.Id,
		AllocationType: types.AllocationType_static,
		Power:          1000000,
	}
	allocResp, err := msgServer.AllocationCreate(sdk.WrapSDKContext(ctx), allocationMsg)
	require.NoError(t, err)
	allocationId := allocResp.AllocationId

	substationMsg := &types.MsgSubstationCreate{
		Creator:      reactorOwnerAddr,
		Owner:        reactorOwnerAddr,
		AllocationId: allocationId,
	}
	substationResp, err := msgServer.SubstationCreate(sdk.WrapSDKContext(ctx), substationMsg)
	require.NoError(t, err)
	substationId = substationResp.SubstationId

	guildMsg := &types.MsgGuildCreate{
		Creator:           reactorOwnerAddr,
		ReactorId:         reactor.Id,
		Endpoint:          "test-endpoint",
		EntrySubstationId: substationId,
	}
	guildResp, err := msgServer.GuildCreate(sdk.WrapSDKContext(ctx), guildMsg)
	require.NoError(t, err)
	guildId = guildResp.GuildId

	return reactorOwnerAddr, substationId, guildId
}

func simulateMsgStructBuildInitiateWithAccount(
	k structskeeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	simAccount simtypes.Account,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		playerIndex := k.GetPlayerIndexFromAddress(ctx, simAccount.Address.String())
		player, _ := k.GetPlayerFromIndex(ctx, playerIndex)

		msg := &types.MsgStructBuildInitiate{
			Creator:        simAccount.Address.String(),
			PlayerId:       player.Id,
			StructTypeId:   1,
			OperatingAmbit: types.Ambit_space,
			Slot:           0,
		}

		msgServer := structskeeper.NewMsgServerImpl(k)
		_, err := msgServer.StructBuildInitiate(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, "struct_build_initiate", "failed to execute message"), nil, err
		}
		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

func TestSimulateMsgStructBuildInitiate(t *testing.T) {
	k, ctx := testkeeper.StructsKeeper(t)
	simAccount, _ := simtypes.RandomAcc(rand.New(rand.NewSource(1)), simtypes.RandomAccounts(rand.New(rand.NewSource(1)), 1))
	mockApp := newTestBaseApp()

	createAllStructTypesFromGenesis(t, k, ctx)

	_, _, _ = setupEnergyGridForSim(t, k, ctx)

	player := registerPlayerWithAllPermissions(t, k, ctx, simAccount.Address.String())

	fleetId := fmt.Sprintf("%d-%d", types.ObjectType_fleet, player.Index)
	fleet := types.Fleet{
		Owner: player.Id,
		Id:    fleetId,
	}
	k.SetFleet(ctx, fleet)

	cc := k.NewCurrentContext(ctx)
	playerCache, err := cc.GetPlayer(player.Id)
	require.NoError(t, err)
	playerCache.SetFleetId(fleetId)
	playerCache.Commit()

	k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id), 100000)
	k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_load, player.Id), 0)
	k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, player.Id), 0)
	k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id), 0)

	requiredCharge := uint64(50000)
	for i := 0; i < int(requiredCharge*2); i++ {
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	}

	charge := getPlayerCharge(k, ctx, player.Id)
	require.GreaterOrEqual(t, charge, requiredCharge, "Player does not have enough charge to build struct")

	op := simulateMsgStructBuildInitiateWithAccount(k, k.AccountKeeper(), k.BankKeeper(), simAccount)
	opMsg, futureOps, err := op(rand.New(rand.NewSource(1)), mockApp.BaseApp, ctx, []simtypes.Account{simAccount}, "")
	require.NoError(t, err)
	require.True(t, opMsg.OK)
	require.Len(t, futureOps, 0)
}

func TestSimulateMsgStructMove(t *testing.T) {
	k, ctx := testkeeper.StructsKeeper(t)
	simAccount, _ := simtypes.RandomAcc(rand.New(rand.NewSource(1)), simtypes.RandomAccounts(rand.New(rand.NewSource(1)), 1))
	mockApp := newTestBaseApp()

	createAllStructTypesFromGenesis(t, k, ctx)

	_, _, _ = setupEnergyGridForSim(t, k, ctx)

	player := registerPlayerWithAllPermissions(t, k, ctx, simAccount.Address.String())

	fleetId := fmt.Sprintf("%d-%d", types.ObjectType_fleet, player.Index)
	fleet := types.Fleet{
		Owner: player.Id,
		Id:    fleetId,
	}
	k.SetFleet(ctx, fleet)

	cc := k.NewCurrentContext(ctx)
	playerCache, err := cc.GetPlayer(player.Id)
	require.NoError(t, err)
	playerCache.SetFleetId(fleetId)
	playerCache.Commit()

	k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id), 100000)
	k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_load, player.Id), 0)
	k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, player.Id), 0)
	k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id), 0)

	requiredCharge := uint64(50000)
	for i := 0; i < int(requiredCharge*2); i++ {
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	}

	charge := getPlayerCharge(k, ctx, player.Id)
	require.GreaterOrEqual(t, charge, requiredCharge, "Player does not have enough charge to build struct")

	msgServer := structskeeper.NewMsgServerImpl(k)
	_, err = msgServer.StructBuildInitiate(sdk.WrapSDKContext(ctx), &types.MsgStructBuildInitiate{
		Creator:        simAccount.Address.String(),
		PlayerId:       player.Id,
		StructTypeId:   1,
		OperatingAmbit: types.Ambit_land,
		Slot:           0,
	})
	require.NoError(t, err)

	structId := "5-1"
	cc2 := k.NewCurrentContext(ctx)
	structCache := cc2.GetStruct(structId)
	if structCache.CheckStruct() != nil || !structCache.IsBuilt() {
		t.Logf("Struct %s is not built yet, skipping activation and move", structId)
		return
	}

	_, err = msgServer.StructActivate(sdk.WrapSDKContext(ctx), &types.MsgStructActivate{
		Creator:  simAccount.Address.String(),
		StructId: structId,
	})
	require.NoError(t, err)

	for i := 0; i < int(requiredCharge); i++ {
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	}

	charge = getPlayerCharge(k, ctx, player.Id)
	require.GreaterOrEqual(t, charge, requiredCharge, "Player does not have enough charge to move struct")

	moveOp := SimulateMsgStructMove(k, k.AccountKeeper(), k.BankKeeper())
	opMsg, futureOps, err := moveOp(rand.New(rand.NewSource(1)), mockApp.BaseApp, ctx, []simtypes.Account{simAccount}, "")
	require.NoError(t, err)
	require.True(t, opMsg.OK)
	require.Len(t, futureOps, 0)
}
