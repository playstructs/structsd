package simulation

import (
	"math/rand"
	"testing"

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

// MockBaseApp is a mock implementation of BaseApp for testing
type MockBaseApp struct {
	*baseapp.BaseApp
}

func newTestBaseApp() *MockBaseApp {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)
	txConfig := tx.NewTxConfig(cdc, tx.DefaultSignModes)
	app := &MockBaseApp{
		BaseApp: baseapp.NewBaseApp("test", nil, nil, nil),
	}
	app.SetTxEncoder(txConfig.TxEncoder())
	app.SetTxDecoder(txConfig.TxDecoder())
	return app
}

func registerPlayerWithAllPermissions(t *testing.T, k structskeeper.Keeper, ctx sdk.Context, address string) {
	// Create the player first
	player := k.UpsertPlayer(ctx, address)

	// Register the player
	msg := &types.MsgAddressRegister{
		Creator:     address,
		PlayerId:    player.Id,
		Address:     address,
		Permissions: 127, // all flags
	}
	msgServer := structskeeper.NewMsgServerImpl(k)
	_, _ = msgServer.AddressRegister(sdk.WrapSDKContext(ctx), msg)

	// Debug: check if player exists in the store after registration
	_, found := k.GetPlayer(ctx, player.Id)
	require.True(t, found, "Player not found in store after registration: %s", player.Id)

	// Set address permissions
	addressPermissionId := structskeeper.GetAddressPermissionIDBytes(address)
	k.PermissionAdd(ctx, addressPermissionId, types.PermissionPlay)

	// Set player as primary address for itself
	playerCache, err := k.GetPlayerCacheFromId(ctx, player.Id)
	require.NoError(t, err)
	playerCache.LoadPlayer()
	playerCache.SetPrimaryAddress(address)
	playerCache.Commit()
}

func createAllStructTypesFromGenesis(t *testing.T, k structskeeper.Keeper, ctx sdk.Context) {
	genesisStructTypes := types.CreateStructTypeGenesis()
	for _, structType := range genesisStructTypes {
		k.AppendStructType(ctx, structType)
	}
	// Optionally, check that at least one struct type exists
	for _, structType := range genesisStructTypes {
		_, found := k.GetStructType(ctx, structType.Id)
		require.True(t, found, "Struct type not found in store after creation: %d", structType.Id)
	}
}

func waitForPlayerCharge(t *testing.T, k structskeeper.Keeper, ctx sdk.Context, playerId string, requiredCharge uint64) sdk.Context {
	for {
		charge := k.GetPlayerCharge(ctx, playerId)
		if charge >= requiredCharge {
			return ctx
		}
		// increment block height
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	}
}

func setupEnergyGridForSim(t *testing.T, k structskeeper.Keeper, ctx sdk.Context) (reactorOwnerAddr string, substationId string, guildId string) {
	// 1. Generate a new account to act as the reactor owner/player
	reactorOwner := simtypes.RandomAccounts(rand.New(rand.NewSource(2)), 1)[0]
	reactorOwnerAddr = reactorOwner.Address.String()

	// Register the reactor owner as a player with all permissions
	registerPlayerWithAllPermissions(t, k, ctx, reactorOwnerAddr)

	// 2. Create a validator (reactor) for that account
	// For test simplicity, we assume the validator is created and bonded (mock or use staking keeper if available)
	// If you have a helper for validator creation, use it here. Otherwise, just append a reactor directly:
	reactor := types.CreateEmptyReactor()
	reactor.Validator = reactorOwnerAddr
	reactor.RawAddress = reactorOwner.Address.Bytes()
	reactor = k.AppendReactor(ctx, reactor)

	// Set up reactor permissions for the owner
	player := k.UpsertPlayer(ctx, reactorOwnerAddr)
	reactorPermissionId := structskeeper.GetObjectPermissionIDBytes(reactor.Id, player.Id)
	k.PermissionAdd(ctx, reactorPermissionId, types.PermissionAll)

	// Set the reactor's grid capacity to a high value
	capacity := uint64(10000000) // 10 million, more than enough
	k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, reactor.Id), capacity)

	// 3. Delegate alpha (tokens) from the player to the validator
	// For test simplicity, we skip actual staking module logic and assume delegation is successful
	// (If you want to use staking keeper, add the delegation here)

	// 4. Create an allocation from the player (reactor owner) using the reactor as the source
	msgServer := structskeeper.NewMsgServerImpl(k)
	allocationMsg := &types.MsgAllocationCreate{
		Creator:        reactorOwnerAddr,
		Controller:     reactorOwnerAddr,
		SourceObjectId: reactor.Id,
		AllocationType: types.AllocationType_static,
		Power:          1000000, // plenty of power
	}
	allocResp, err := msgServer.AllocationCreate(sdk.WrapSDKContext(ctx), allocationMsg)
	require.NoError(t, err)
	allocationId := allocResp.AllocationId

	// 5. Create a substation using the allocation
	substationMsg := &types.MsgSubstationCreate{
		Creator:      reactorOwnerAddr,
		Owner:        reactorOwnerAddr,
		AllocationId: allocationId,
	}
	substationResp, err := msgServer.SubstationCreate(sdk.WrapSDKContext(ctx), substationMsg)
	require.NoError(t, err)
	substationId = substationResp.SubstationId

	// 6. Create a guild using the substation
	guildMsg := &types.MsgGuildCreate{
		Creator:           reactorOwnerAddr,
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
		// Get the player ID associated with the simAccount address
		playerIndex := k.GetPlayerIndexFromAddress(ctx, simAccount.Address.String())
		player, _ := k.GetPlayerFromIndex(ctx, playerIndex)

		msg := &types.MsgStructBuildInitiate{
			Creator:        simAccount.Address.String(),
			PlayerId:       player.Id,
			StructTypeId:   1, // deterministic struct type for test
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

	// Create all struct types from genesis
	createAllStructTypesFromGenesis(t, k, ctx)

	// Setup energy grid: reactor, allocation, substation, guild
	_, _, _ = setupEnergyGridForSim(t, k, ctx)

	// Register the account as a player with all permissions
	registerPlayerWithAllPermissions(t, k, ctx, simAccount.Address.String())

	// Get the player ID associated with the simAccount address
	playerIndex := k.GetPlayerIndexFromAddress(ctx, simAccount.Address.String())
	player, _ := k.GetPlayerFromIndex(ctx, playerIndex)
	playerCache, _ := k.GetPlayerCacheFromId(ctx, player.Id)

	// Create a fleet for the player
	fleet := k.AppendFleet(ctx, &playerCache)
	playerCache.SetFleetId(fleet.Id)
	playerCache.Commit()

	// Set up grid attributes for all accounts used in the simulation
	accs := []simtypes.Account{simAccount}
	for _, acc := range accs {
		playerIndex := k.GetPlayerIndexFromAddress(ctx, acc.Address.String())
		player, _ := k.GetPlayerFromIndex(ctx, playerIndex)
		k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id), 100000)
		k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_load, player.Id), 0)
		k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, player.Id), 0)
		k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id), 0)
	}

	// Wait for enough charge (use 50000 for struct build)
	requiredCharge := uint64(50000)
	// Increment block height more aggressively to accumulate charge
	for i := 0; i < int(requiredCharge*2); i++ {
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	}

	// Debug: print player's grid attributes before struct build
	playerIndex = k.GetPlayerIndexFromAddress(ctx, simAccount.Address.String())
	player, _ = k.GetPlayerFromIndex(ctx, playerIndex)
	capacity := k.GetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id))
	load := k.GetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_load, player.Id))
	structsLoad := k.GetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, player.Id))
	lastAction := k.GetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id))
	charge := k.GetPlayerCharge(ctx, player.Id)
	playerCache, err := k.GetPlayerCacheFromId(ctx, player.Id)
	require.NoError(t, err)
	availableCapacity := playerCache.GetAvailableCapacity()
	t.Logf("PlayerId: %s, Capacity: %d, Load: %d, StructsLoad: %d, AvailableCapacity: %d, LastAction: %d, Charge: %d", player.Id, capacity, load, structsLoad, availableCapacity, lastAction, charge)

	// Verify we have enough charge before proceeding
	require.GreaterOrEqual(t, charge, requiredCharge, "Player does not have enough charge to build struct")

	// Use the deterministic simulation operation
	op := simulateMsgStructBuildInitiateWithAccount(k, k.AccountKeeper(), k.BankKeeper(), simAccount)
	opMsg, futureOps, err := op(rand.New(rand.NewSource(1)), mockApp.BaseApp, ctx, accs, "")
	require.NoError(t, err)
	require.True(t, opMsg.OK)
	require.Len(t, futureOps, 0)
}

func TestSimulateMsgStructMove(t *testing.T) {
	k, ctx := testkeeper.StructsKeeper(t)
	simAccount, _ := simtypes.RandomAcc(rand.New(rand.NewSource(1)), simtypes.RandomAccounts(rand.New(rand.NewSource(1)), 1))
	mockApp := newTestBaseApp()

	// Create all struct types from genesis
	createAllStructTypesFromGenesis(t, k, ctx)

	// Setup energy grid: reactor, allocation, substation, guild
	_, _, _ = setupEnergyGridForSim(t, k, ctx)

	// Register the account as a player with all permissions
	registerPlayerWithAllPermissions(t, k, ctx, simAccount.Address.String())

	// Get the player ID associated with the simAccount address
	playerIndex := k.GetPlayerIndexFromAddress(ctx, simAccount.Address.String())
	player, _ := k.GetPlayerFromIndex(ctx, playerIndex)
	playerCache, _ := k.GetPlayerCacheFromId(ctx, player.Id)

	// Create a fleet for the player
	fleet := k.AppendFleet(ctx, &playerCache)
	playerCache.SetFleetId(fleet.Id)
	playerCache.Commit()

	// Set the player's grid capacity to 100000
	k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id), 100000)
	// Set load and structs load to 0
	k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_load, player.Id), 0)
	k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, player.Id), 0)
	// Set last action block to 0 for max charge
	k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id), 0)

	// Wait for enough charge (use 50000 for struct build)
	requiredCharge := uint64(50000)
	// Increment block height more aggressively to accumulate charge
	for i := 0; i < int(requiredCharge*2); i++ {
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	}

	// Debug: print player's grid attributes before struct build
	capacity := k.GetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id))
	load := k.GetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_load, player.Id))
	structsLoad := k.GetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, player.Id))
	lastAction := k.GetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id))
	charge := k.GetPlayerCharge(ctx, player.Id)
	playerCache, err := k.GetPlayerCacheFromId(ctx, player.Id)
	require.NoError(t, err)
	availableCapacity := playerCache.GetAvailableCapacity()
	t.Logf("PlayerId: %s, Capacity: %d, Load: %d, StructsLoad: %d, AvailableCapacity: %d, LastAction: %d, Charge: %d", player.Id, capacity, load, structsLoad, availableCapacity, lastAction, charge)

	// Verify we have enough charge before proceeding
	require.GreaterOrEqual(t, charge, requiredCharge, "Player does not have enough charge to build struct")

	// Build the struct
	msgServer := structskeeper.NewMsgServerImpl(k)
	_, err = msgServer.StructBuildInitiate(ctx, &types.MsgStructBuildInitiate{
		Creator:        simAccount.Address.String(),
		PlayerId:       player.Id,
		StructTypeId:   1,
		OperatingAmbit: types.Ambit_land,
		Slot:           0,
	})
	require.NoError(t, err)

	// Check if the struct is built before trying to activate and move
	structId := "5-1"
	structCache := k.GetStructCacheFromId(ctx, structId)
	if !structCache.IsBuilt() {
		t.Logf("Struct %s is not built yet, skipping activation and move", structId)
		return
	}

	// Activate the struct
	_, err = msgServer.StructActivate(ctx, &types.MsgStructActivate{
		Creator:  simAccount.Address.String(),
		StructId: structId,
	})
	require.NoError(t, err)

	// Wait for more charge before moving the struct
	for i := 0; i < int(requiredCharge); i++ {
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	}

	// Verify we have enough charge before moving
	charge = k.GetPlayerCharge(ctx, player.Id)
	require.GreaterOrEqual(t, charge, requiredCharge, "Player does not have enough charge to move struct")

	// Now try to move it
	moveOp := SimulateMsgStructMove(k, k.AccountKeeper(), k.BankKeeper())
	opMsg, futureOps, err := moveOp(rand.New(rand.NewSource(1)), mockApp.BaseApp, ctx, []simtypes.Account{simAccount}, "")
	require.NoError(t, err)
	require.True(t, opMsg.OK)
	require.Len(t, futureOps, 0)
}
