package simulation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"structs/x/structs/keeper"
	"structs/x/structs/types"
)

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
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildInitiate{}), "account not found"), nil, nil
		}

		// Get player ID from address - if player doesn't exist, skip this operation
		playerIndex := k.GetPlayerIndexFromAddress(ctx, simAccount.Address.String())
		if playerIndex == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildInitiate{}), "player not found"), nil, nil
		}
		player, found := k.GetPlayerFromIndex(ctx, playerIndex)
		if !found {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildInitiate{}), "player not found"), nil, nil
		}

		// Ensure player has explored a planet (which creates the fleet) before building structs
		playerCache, err := k.GetPlayerCacheFromId(ctx, player.Id)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildInitiate{}), "failed to get player cache"), nil, nil
		}

		// If player doesn't have a planet, they need to explore one first (which creates the fleet)
		if !playerCache.HasPlanet() {
			// Explore a planet to create the fleet
			exploreErr := playerCache.AttemptPlanetExplore()
			if exploreErr != nil {
				return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildInitiate{}), "failed to explore planet: "+exploreErr.Error()), nil, nil
			}
			// After exploring, the fleet needs to be set up
			playerCache.GetFleet().ManualLoadOwner(&playerCache)
			playerCache.GetFleet().MigrateToNewPlanet(playerCache.GetPlanet())
			playerCache.Commit()
		}

		// Get available struct types (limit to reasonable range)
		structTypeId := uint64(r.Int63n(10) + 1) // 1-10

		// Random ambit (ensure valid enum value)
		ambitValues := []types.Ambit{
			types.Ambit_space,
			types.Ambit_land,
			types.Ambit_water,
		}
		operatingAmbit := ambitValues[r.Intn(len(ambitValues))]

		// Random slot (0-9)
		slot := uint64(r.Int63n(10))

		msg := &types.MsgStructBuildInitiate{
			Creator:        simAccount.Address.String(),
			PlayerId:       player.Id,
			StructTypeId:   structTypeId,
			OperatingAmbit: operatingAmbit,
			Slot:           slot,
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.StructBuildInitiate(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
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
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructMove{}), "account not found"), nil, nil
		}

		// Get a random struct from the store
		structs := k.GetAllStruct(ctx)
		if len(structs) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructMove{}), "no structs found"), nil, nil
		}

		structToMove := structs[r.Intn(len(structs))]

		// Random location type (ensure valid enum value)
		locationTypeValues := []types.ObjectType{
			types.ObjectType_player,
			types.ObjectType_fleet,
			types.ObjectType_substation,
		}
		locationType := locationTypeValues[r.Intn(len(locationTypeValues))]

		// Random ambit (ensure valid enum value)
		ambitValues := []types.Ambit{
			types.Ambit_space,
			types.Ambit_land,
			types.Ambit_water,
		}
		ambit := ambitValues[r.Intn(len(ambitValues))]

		// Random slot (0-9)
		slot := uint64(r.Int63n(10))

		msg := &types.MsgStructMove{
			Creator:      simAccount.Address.String(),
			StructId:     structToMove.Id,
			LocationType: locationType,
			Ambit:        ambit,
			Slot:         slot,
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.StructMove(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildCreate generates a MsgGuildCreate with random values
func SimulateMsgGuildCreate(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildCreate{}), "account not found"), nil, nil
		}

		// Get player ID from address
		playerIndex := k.GetPlayerIndexFromAddress(ctx, simAccount.Address.String())
		if playerIndex == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildCreate{}), "player not found"), nil, nil
		}
		player, found := k.GetPlayerFromIndex(ctx, playerIndex)
		if !found {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildCreate{}), "player not found"), nil, nil
		}

		// Check if player already has a guild
		if player.GuildId != "" {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildCreate{}), "player already in guild"), nil, nil
		}

		// Check if player has a reactor (required for guild creation)
		validatorAddress := sdk.ValAddress(simAccount.Address.Bytes())
		reactorBytes, _ := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
		_, reactorFound := k.GetReactorByBytes(ctx, reactorBytes)
		if !reactorFound {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildCreate{}), "reactor not found"), nil, nil
		}

		// Optionally use a substation if available
		entrySubstationId := ""
		substations := k.GetAllSubstation(ctx)
		if len(substations) > 0 && r.Intn(2) == 0 { // 50% chance to use substation
			// Find a substation the player has access to
			for _, substation := range substations {
				substationPermissionId := keeper.GetObjectPermissionIDBytes(substation.Id, player.Id)
				if k.PermissionHasOneOf(ctx, substationPermissionId, types.PermissionGrid) {
					entrySubstationId = substation.Id
					break
				}
			}
		}

		// Generate random endpoint
		endpoint := simtypes.RandStringOfLength(r, 10)

		msg := &types.MsgGuildCreate{
			Creator:           simAccount.Address.String(),
			Endpoint:          endpoint,
			EntrySubstationId: entrySubstationId,
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.GuildCreate(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildBankMint generates a MsgGuildBankMint with random values
func SimulateMsgGuildBankMint(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildBankMint{}), "account not found"), nil, nil
		}

		// Get player cache
		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildBankMint{}), "player not found"), nil, nil
		}

		// Check if player is in a guild
		if activePlayer.GetGuildId() == "" {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildBankMint{}), "player not in guild"), nil, nil
		}

		// Check bank administration permissions
		guild := k.GetGuildCacheFromId(ctx, activePlayer.GetGuildId())
		permissionError := guild.CanAdministrateBank(&activePlayer)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildBankMint{}), "no bank admin permission"), nil, nil
		}

		// Generate random amounts (reasonable ranges)
		amountAlpha := uint64(r.Int63n(1000000) + 1000) // 1000-1000000
		amountToken := uint64(r.Int63n(1000000) + 1000) // 1000-1000000

		msg := &types.MsgGuildBankMint{
			Creator:     simAccount.Address.String(),
			AmountAlpha: amountAlpha,
			AmountToken: amountToken,
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.GuildBankMint(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildBankRedeem generates a MsgGuildBankRedeem with random values
func SimulateMsgGuildBankRedeem(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildBankRedeem{}), "account not found"), nil, nil
		}

		// Get player cache
		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildBankRedeem{}), "player not found"), nil, nil
		}

		// Get a random guild that has bank tokens (try player's guild first, then random)
		var guildId string
		if activePlayer.GetGuildId() != "" {
			guildId = activePlayer.GetGuildId()
		} else {
			// Get a random guild
			guilds := k.GetAllGuild(ctx)
			if len(guilds) == 0 {
				return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildBankRedeem{}), "no guilds found"), nil, nil
			}
			guildId = guilds[r.Intn(len(guilds))].Id
		}

		// Generate random token amount
		amountToken := uint64(r.Int63n(100000) + 100) // 100-100000

		// Create the denom in the format "uguild.{guildId}"
		denom := "uguild." + guildId

		msg := &types.MsgGuildBankRedeem{
			Creator: simAccount.Address.String(),
			AmountToken: sdk.Coin{
				Denom:  denom,
				Amount: math.NewIntFromUint64(amountToken),
			},
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.GuildBankRedeem(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildBankConfiscateAndBurn generates a MsgGuildBankConfiscateAndBurn with random values
func SimulateMsgGuildBankConfiscateAndBurn(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildBankConfiscateAndBurn{}), "account not found"), nil, nil
		}

		// Get player cache
		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildBankConfiscateAndBurn{}), "player not found"), nil, nil
		}

		// Check if player is in a guild
		if activePlayer.GetGuildId() == "" {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildBankConfiscateAndBurn{}), "player not in guild"), nil, nil
		}

		// Check bank administration permissions
		guild := k.GetGuildCacheFromId(ctx, activePlayer.GetGuildId())
		permissionError := guild.CanAdministrateBank(&activePlayer)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildBankConfiscateAndBurn{}), "no bank admin permission"), nil, nil
		}

		// Get a random address to confiscate from
		targetAccount, _ := simtypes.RandomAcc(r, accs)
		targetAddress := targetAccount.Address.String()

		// Generate random token amount
		amountToken := uint64(r.Int63n(100000) + 100) // 100-100000

		msg := &types.MsgGuildBankConfiscateAndBurn{
			Creator:     simAccount.Address.String(),
			Address:     targetAddress,
			AmountToken: amountToken,
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.GuildBankConfiscateAndBurn(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgAddressRegister generates a MsgAddressRegister to register a new address for a player
func SimulateMsgAddressRegister(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Get a random account that will be the creator (must have a player)
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAddressRegister{}), "account not found"), nil, nil
		}

		// Get or create player for the creator
		player := k.UpsertPlayer(ctx, simAccount.Address.String())

		// Get a random account to register as a new address (different from creator)
		newAccount, _ := simtypes.RandomAcc(r, accs)
		if newAccount.Address.String() == simAccount.Address.String() {
			// Skip if same account
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAddressRegister{}), "same account"), nil, nil
		}

		// Check if address is already registered
		existingPlayerIndex := k.GetPlayerIndexFromAddress(ctx, newAccount.Address.String())
		if existingPlayerIndex > 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAddressRegister{}), "address already registered"), nil, nil
		}

		// Generate proof signature (simplified for simulation - in real usage this would be cryptographic)
		// For simulation, we'll use a simple approach: create a message hash and sign it
		hashInput := fmt.Sprintf("PLAYER%sADDRESS%s", player.Id, newAccount.Address.String())
		hashBytes := []byte(hashInput)

		// Sign with the new account's private key
		signature, err := newAccount.PrivKey.Sign(hashBytes)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAddressRegister{}), "failed to sign proof"), nil, nil
		}

		// Encode pubkey and signature to hex
		proofPubKey := hex.EncodeToString(newAccount.PubKey.Bytes())
		proofSignature := hex.EncodeToString(signature)

		// Set permissions (use basic play permissions for simulation)
		permissions := uint64(types.PermissionPlay)

		msg := &types.MsgAddressRegister{
			Creator:        simAccount.Address.String(),
			PlayerId:       player.Id,
			Address:        newAccount.Address.String(),
			Permissions:    permissions,
			ProofPubKey:    proofPubKey,
			ProofSignature: proofSignature,
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.AddressRegister(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgPlayerSend generates a MsgPlayerSend to transfer tokens between addresses
func SimulateMsgPlayerSend(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Get a random account that will send tokens
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgPlayerSend{}), "account not found"), nil, nil
		}

		// Get or create player for the sender
		player := k.UpsertPlayer(ctx, simAccount.Address.String())
		playerCache, err := k.GetPlayerCacheFromId(ctx, player.Id)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgPlayerSend{}), "failed to get player cache"), nil, nil
		}

		// Check if sender has assets permission
		permissionError := playerCache.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionAssets)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgPlayerSend{}), "no assets permission"), nil, nil
		}

		// Get a random recipient (different from sender)
		recipientAccount, _ := simtypes.RandomAcc(r, accs)
		if recipientAccount.Address.String() == simAccount.Address.String() {
			// Try again if same account
			recipientAccount, _ = simtypes.RandomAcc(r, accs)
		}

		// Check sender's balance
		senderBalance := bk.SpendableCoins(ctx, simAccount.Address)
		if senderBalance.IsZero() {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgPlayerSend{}), "insufficient balance"), nil, nil
		}

		// Generate a small random amount to send (1-1000 ualpha)
		amountToSend := uint64(r.Int63n(1000) + 1)
		sendAmount := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewIntFromUint64(amountToSend)))

		// Make sure we don't send more than available
		if senderBalance.AmountOf("ualpha").LT(sendAmount.AmountOf("ualpha")) {
			sendAmount = sdk.NewCoins(sdk.NewCoin("ualpha", senderBalance.AmountOf("ualpha")))
		}

		msg := &types.MsgPlayerSend{
			Creator:     simAccount.Address.String(),
			PlayerId:    player.Id,
			FromAddress: simAccount.Address.String(),
			ToAddress:   recipientAccount.Address.String(),
			Amount:      sendAmount,
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.PlayerSend(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildMembershipRequest generates a MsgGuildMembershipRequest to request joining a guild
func SimulateMsgGuildMembershipRequest(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipRequest{}), "account not found"), nil, nil
		}

		// Get or create player
		player := k.UpsertPlayer(ctx, simAccount.Address.String())
		playerCache, err := k.GetPlayerCacheFromId(ctx, player.Id)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipRequest{}), "failed to get player cache"), nil, nil
		}

		// Check if player already in a guild
		if playerCache.GetGuildId() != "" {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipRequest{}), "player already in guild"), nil, nil
		}

		// Get a random guild to request joining
		guilds := k.GetAllGuild(ctx)
		if len(guilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipRequest{}), "no guilds found"), nil, nil
		}

		targetGuild := guilds[r.Intn(len(guilds))]

		msg := &types.MsgGuildMembershipRequest{
			Creator:  simAccount.Address.String(),
			GuildId:  targetGuild.Id,
			PlayerId: player.Id,
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.GuildMembershipRequest(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildMembershipJoin generates a MsgGuildMembershipJoin for direct guild joining
func SimulateMsgGuildMembershipJoin(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipJoin{}), "account not found"), nil, nil
		}

		// Get or create player
		player := k.UpsertPlayer(ctx, simAccount.Address.String())
		playerCache, err := k.GetPlayerCacheFromId(ctx, player.Id)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipJoin{}), "failed to get player cache"), nil, nil
		}

		// Check if player already in a guild
		if playerCache.GetGuildId() != "" {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipJoin{}), "player already in guild"), nil, nil
		}

		// Get a random guild to join
		guilds := k.GetAllGuild(ctx)
		if len(guilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipJoin{}), "no guilds found"), nil, nil
		}

		targetGuild := guilds[r.Intn(len(guilds))]

		msg := &types.MsgGuildMembershipJoin{
			Creator:  simAccount.Address.String(),
			GuildId:  targetGuild.Id,
			PlayerId: player.Id,
			// InfusionId can be empty if guild doesn't require minimum infusion
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.GuildMembershipJoin(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgPlanetExplore generates a MsgPlanetExplore to explore a new planet
func SimulateMsgPlanetExplore(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgPlanetExplore{}), "account not found"), nil, nil
		}

		// Get or create player
		player := k.UpsertPlayer(ctx, simAccount.Address.String())

		msg := &types.MsgPlanetExplore{
			Creator:  simAccount.Address.String(),
			PlayerId: player.Id,
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.PlanetExplore(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgReactorInfuse generates a MsgReactorInfuse to add ualpha to a guild's reactor
func SimulateMsgReactorInfuse(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorInfuse{}), "account not found"), nil, nil
		}

		// Get or create player
		player := k.UpsertPlayer(ctx, simAccount.Address.String())
		playerCache, err := k.GetPlayerCacheFromId(ctx, player.Id)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorInfuse{}), "failed to get player cache"), nil, nil
		}

		// Check if player is in a guild
		if playerCache.GetGuildId() == "" {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorInfuse{}), "player not in guild"), nil, nil
		}

		// Check if player has assets permission
		permissionError := playerCache.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionAssets)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorInfuse{}), "no assets permission"), nil, nil
		}

		// Get the guild
		guild := k.GetGuildCacheFromId(ctx, playerCache.GetGuildId())
		primaryReactorId := guild.GetPrimaryReactorId()
		if primaryReactorId == "" {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorInfuse{}), "guild has no primary reactor"), nil, nil
		}

		// Get the reactor
		reactor, reactorFound := k.GetReactor(ctx, primaryReactorId)
		if !reactorFound {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorInfuse{}), "reactor not found"), nil, nil
		}

		// Use "ualpha" as the bond denom for simulation
		// In production, this would be retrieved from the staking keeper
		bondDenom := "ualpha"

		// Check player's balance
		playerBalance := bk.SpendableCoins(ctx, simAccount.Address)
		if playerBalance.IsZero() {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorInfuse{}), "insufficient balance"), nil, nil
		}

		// Generate a random amount to infuse (1-10000 ualpha)
		maxAmount := uint64(10000)
		balanceAmount := playerBalance.AmountOf(bondDenom)
		if balanceAmount.IsPositive() && balanceAmount.Uint64() < maxAmount {
			maxAmount = balanceAmount.Uint64()
		}
		if maxAmount == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorInfuse{}), "insufficient balance"), nil, nil
		}

		amountToInfuse := uint64(r.Int63n(int64(maxAmount)) + 1)
		infuseAmount := sdk.NewCoin(bondDenom, math.NewIntFromUint64(amountToInfuse))

		msg := &types.MsgReactorInfuse{
			Creator:          simAccount.Address.String(),
			DelegatorAddress: simAccount.Address.String(),
			ValidatorAddress: reactor.Validator,
			Amount:           infuseAmount,
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.ReactorInfuse(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateCommandShipBuildInitiate generates a MsgStructBuildInitiate for command ships (ID 1) when player has a planet
func SimulateCommandShipBuildInitiate(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildInitiate{}), "account not found"), nil, nil
		}

		// Get or create player
		player := k.UpsertPlayer(ctx, simAccount.Address.String())
		playerCache, err := k.GetPlayerCacheFromId(ctx, player.Id)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildInitiate{}), "failed to get player cache"), nil, nil
		}

		// Ensure player has explored a planet
		if !playerCache.HasPlanet() {
			// Explore a planet to create the fleet
			exploreErr := playerCache.AttemptPlanetExplore()
			if exploreErr != nil {
				return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildInitiate{}), "failed to explore planet: "+exploreErr.Error()), nil, nil
			}
			// After exploring, the fleet needs to be set up
			playerCache.GetFleet().ManualLoadOwner(&playerCache)
			playerCache.GetFleet().MigrateToNewPlanet(playerCache.GetPlanet())
			playerCache.Commit()
		}

		// Check if player already has a command ship
		if playerCache.GetFleet().HasCommandStruct() {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildInitiate{}), "player already has command ship"), nil, nil
		}

		// Command Ship is struct type ID 1, must be built in fleet (ObjectType_fleet)
		// Random ambit (command ships can be in space, land, or water)
		ambitValues := []types.Ambit{
			types.Ambit_space,
			types.Ambit_land,
			types.Ambit_water,
		}
		operatingAmbit := ambitValues[r.Intn(len(ambitValues))]

		// Command ships don't use slots
		slot := uint64(0)

		msg := &types.MsgStructBuildInitiate{
			Creator:        simAccount.Address.String(),
			PlayerId:       player.Id,
			StructTypeId:   types.CommandStructTypeId, // 1
			OperatingAmbit: operatingAmbit,
			Slot:           slot,
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.StructBuildInitiate(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateCommandShipBuildComplete generates a MsgStructBuildComplete for command ships when difficulty <= 2
func SimulateCommandShipBuildComplete(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Get all structs that are being built
		structs := k.GetAllStruct(ctx)
		if len(structs) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildComplete{}), "no structs found"), nil, nil
		}

		// Find a command ship (ID 1) that is being built
		var targetStruct *types.Struct
		for i := range structs {
			structCache := k.GetStructCacheFromId(ctx, structs[i].Id)
			if !structCache.LoadStruct() {
				continue
			}
			if structCache.GetStructType().Id == types.CommandStructTypeId && !structCache.IsBuilt() {
				targetStruct = &structs[i]
				break
			}
		}

		if targetStruct == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildComplete{}), "no command ship builds in progress"), nil, nil
		}

		// Get the struct cache to check difficulty
		structCache := k.GetStructCacheFromId(ctx, targetStruct.Id)
		if !structCache.LoadStruct() {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildComplete{}), "struct not found"), nil, nil
		}

		// Check current difficulty
		currentAge := uint64(ctx.BlockHeight()) - structCache.GetBlockStartBuild()
		difficulty := types.CalculateDifficulty(float64(currentAge), structCache.GetStructType().BuildDifficulty)

		// Only complete if difficulty <= 2
		if difficulty > 2 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildComplete{}), fmt.Sprintf("difficulty too high: %d", difficulty)), nil, nil
		}

		// Find a random account that can play this struct
		simAccount, _ := simtypes.RandomAcc(r, accs)
		permissionError := structCache.CanBePlayedBy(simAccount.Address.String())
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildComplete{}), "no permission"), nil, nil
		}

		// Calculate hash that meets difficulty requirements
		buildStartBlockString := strconv.FormatUint(structCache.GetBlockStartBuild(), 10)
		nonce := uint64(0)
		var proof string
		var hashInput string

		// Try nonces until we find one that meets the difficulty
		for nonce < 1000000 { // Limit search to prevent infinite loops
			nonce++
			hashInput = targetStruct.Id + "BUILD" + buildStartBlockString + "NONCE" + strconv.FormatUint(nonce, 10)
			hash := sha256.New()
			hash.Write([]byte(hashInput))
			proof = hex.EncodeToString(hash.Sum(nil))

			// Check if proof meets difficulty
			if types.HashBuildAndCheckDifficulty(hashInput, proof, currentAge, structCache.GetStructType().BuildDifficulty) {
				break
			}
		}

		if nonce >= 1000000 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildComplete{}), "could not find valid proof"), nil, nil
		}

		msg := &types.MsgStructBuildComplete{
			Creator:  simAccount.Address.String(),
			StructId: targetStruct.Id,
			Proof:    proof,
			Nonce:    strconv.FormatUint(nonce, 10),
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.StructBuildComplete(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateGiftUalpha gifts ualpha to random players from the module account
func SimulateGiftUalpha(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Get a random account to gift
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, "gift_ualpha", "account not found"), nil, nil
		}

		// Generate a random amount to gift (1000-50000 ualpha)
		amountToGift := uint64(r.Int63n(49000) + 1000)
		giftAmount := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewIntFromUint64(amountToGift)))

		// Mint coins from the structs module and send to the player
		err := bk.MintCoins(ctx, types.ModuleName, giftAmount)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, "gift_ualpha", "failed to mint coins: "+err.Error()), nil, nil
		}

		err = bk.SendCoinsFromModuleToAccount(ctx, types.ModuleName, simAccount.Address, giftAmount)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, "gift_ualpha", "failed to send coins: "+err.Error()), nil, nil
		}

		// Return a no-op message since this isn't a real message type
		return simtypes.NoOpMsg(types.ModuleName, "gift_ualpha", fmt.Sprintf("gifted %s to %s", giftAmount.String(), simAccount.Address.String())), nil, nil
	}
}

// SimulateMsgAllocationCreate generates a MsgAllocationCreate with random values
// Players create allocations from themselves or substations they control
func SimulateMsgAllocationCreate(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAllocationCreate{}), "account not found"), nil, nil
		}

		// Get player cache
		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAllocationCreate{}), "player not found"), nil, nil
		}

		// Check if player has assets permission
		permissionError := activePlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionAssets)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAllocationCreate{}), "no assets permission"), nil, nil
		}

		// Choose source: either the player themselves or a substation they control
		var sourceObjectId string
		usePlayerAsSource := r.Intn(2) == 0 // 50% chance

		if usePlayerAsSource {
			// Create allocation from player
			sourceObjectId = activePlayer.GetPlayerId()
		} else {
			// Try to find a substation the player controls
			allSubstations := k.GetAllSubstation(ctx)
			validSubstations := make([]types.Substation, 0)
			for _, substation := range allSubstations {
				substationCache := k.GetSubstationCacheFromId(ctx, substation.Id)
				if substationCache.GetOwnerId() == activePlayer.GetPlayerId() {
					validSubstations = append(validSubstations, substation)
				}
			}

			if len(validSubstations) == 0 {
				// Fall back to using player as source
				sourceObjectId = activePlayer.GetPlayerId()
			} else {
				sourceObjectId = validSubstations[r.Intn(len(validSubstations))].Id
			}
		}

		// Random allocation type (static, dynamic, or automated - not providerAgreement)
		allocationTypes := []types.AllocationType{
			types.AllocationType_static,
			types.AllocationType_dynamic,
			types.AllocationType_automated,
		}
		allocationType := allocationTypes[r.Intn(len(allocationTypes))]

		// Random power amount (1-1000)
		power := uint64(r.Int63n(1000) + 1)

		msg := &types.MsgAllocationCreate{
			Creator:        simAccount.Address.String(),
			Controller:     simAccount.Address.String(), // Controller defaults to creator
			SourceObjectId: sourceObjectId,
			AllocationType: allocationType,
			Power:          power,
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.AllocationCreate(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgSubstationCreate generates a MsgSubstationCreate with random values
// Players create new substations with allocations they control
func SimulateMsgSubstationCreate(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationCreate{}), "account not found"), nil, nil
		}

		// Get player cache
		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationCreate{}), "player not found"), nil, nil
		}

		// Check if player has assets permission
		permissionError := activePlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionAssets)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationCreate{}), "no assets permission"), nil, nil
		}

		// Find an allocation that the player controls (has no destination yet)
		allAllocations := k.GetAllAllocation(ctx)
		validAllocations := make([]types.Allocation, 0)
		for _, allocation := range allAllocations {
			// Allocation must be controlled by this player and not have a destination yet
			if allocation.Controller == simAccount.Address.String() && allocation.DestinationId == "" {
				validAllocations = append(validAllocations, allocation)
			}
		}

		if len(validAllocations) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationCreate{}), "no available allocations"), nil, nil
		}

		// Pick a random allocation
		allocation := validAllocations[r.Intn(len(validAllocations))]

		msg := &types.MsgSubstationCreate{
			Creator:      simAccount.Address.String(),
			AllocationId: allocation.Id,
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.SubstationCreate(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgProviderCreate generates a MsgProviderCreate with random values
// Players create providers on substations they control
func SimulateMsgProviderCreate(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderCreate{}), "account not found"), nil, nil
		}

		// Get player cache
		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderCreate{}), "player not found"), nil, nil
		}

		// Find substations the player controls
		allSubstations := k.GetAllSubstation(ctx)
		validSubstations := make([]types.Substation, 0)
		for _, substation := range allSubstations {
			substationCache := k.GetSubstationCacheFromId(ctx, substation.Id)
			permissionError := substationCache.CanCreateAllocations(&activePlayer)
			if permissionError == nil {
				validSubstations = append(validSubstations, substation)
			}
		}

		if len(validSubstations) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderCreate{}), "no accessible substations"), nil, nil
		}

		// Pick a random substation
		substation := validSubstations[r.Intn(len(validSubstations))]

		// Random provider parameters
		rateAmount := math.NewIntFromUint64(uint64(r.Int63n(1000) + 1)) // 1-1000
		rate := sdk.NewCoin("ualpha", rateAmount)

		// Random access policy
		accessPolicies := []types.ProviderAccessPolicy{
			types.ProviderAccessPolicy_openMarket,
			types.ProviderAccessPolicy_guildMarket,
		}
		accessPolicy := accessPolicies[r.Intn(len(accessPolicies))]

		// Random cancellation penalties (0.0 to 0.5)
		providerPenalty := math.LegacyNewDecWithPrec(int64(r.Int63n(51)), 2) // 0.00 to 0.50
		consumerPenalty := math.LegacyNewDecWithPrec(int64(r.Int63n(51)), 2) // 0.00 to 0.50

		// Random capacity range (1-1000)
		capacityMin := uint64(r.Int63n(100) + 1)               // 1-100
		capacityMax := capacityMin + uint64(r.Int63n(900)) + 1 // capacityMin+1 to 1000
		if capacityMax > 1000 {
			capacityMax = 1000
		}

		// Random duration range (1-100 blocks)
		durationMin := uint64(r.Int63n(10) + 1)               // 1-10
		durationMax := durationMin + uint64(r.Int63n(90)) + 1 // durationMin+1 to 100
		if durationMax > 100 {
			durationMax = 100
		}

		msg := &types.MsgProviderCreate{
			Creator:                     simAccount.Address.String(),
			SubstationId:                substation.Id,
			Rate:                        rate,
			AccessPolicy:                accessPolicy,
			ProviderCancellationPenalty: providerPenalty,
			ConsumerCancellationPenalty: consumerPenalty,
			CapacityMinimum:             capacityMin,
			CapacityMaximum:             capacityMax,
			DurationMinimum:             durationMin,
			DurationMaximum:             durationMax,
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.ProviderCreate(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgAgreementOpen generates a MsgAgreementOpen with random values
// Players enter into agreements on providers
func SimulateMsgAgreementOpen(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementOpen{}), "account not found"), nil, nil
		}

		// Get player cache
		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementOpen{}), "player not found"), nil, nil
		}

		// Find available providers
		allProviders := k.GetAllProvider(ctx)
		validProviders := make([]types.Provider, 0)
		for _, provider := range allProviders {
			providerCache := k.GetProviderCacheFromId(ctx, provider.Id)
			permissionError := providerCache.CanOpenAgreement(&activePlayer)
			if permissionError == nil {
				validProviders = append(validProviders, provider)
			}
		}

		if len(validProviders) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementOpen{}), "no accessible providers"), nil, nil
		}

		// Pick a random provider
		provider := validProviders[r.Intn(len(validProviders))]
		providerCache := k.GetProviderCacheFromId(ctx, provider.Id)

		// Get provider constraints
		capacityMin := providerCache.GetCapacityMinimum()
		capacityMax := providerCache.GetCapacityMaximum()
		durationMin := providerCache.GetDurationMinimum()
		durationMax := providerCache.GetDurationMaximum()

		// Random capacity and duration within provider's constraints
		capacity := capacityMin
		if capacityMax > capacityMin {
			capacity = capacityMin + uint64(r.Int63n(int64(capacityMax-capacityMin)+1))
		}

		duration := durationMin
		if durationMax > durationMin {
			duration = durationMin + uint64(r.Int63n(int64(durationMax-durationMin)+1))
		}

		// Check if player can afford the collateral
		rate := providerCache.GetRate()
		durationInt := math.NewIntFromUint64(duration)
		capacityInt := math.NewIntFromUint64(capacity)
		collateralAmount := durationInt.Mul(capacityInt).Mul(rate.Amount)
		collateralCoin := sdk.NewCoin(rate.Denom, collateralAmount)

		sourceAcc, errParam := sdk.AccAddressFromBech32(simAccount.Address.String())
		if errParam != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementOpen{}), "invalid address"), nil, nil
		}

		if !bk.HasBalance(ctx, sourceAcc, collateralCoin) {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementOpen{}), "insufficient balance for collateral"), nil, nil
		}

		msg := &types.MsgAgreementOpen{
			Creator:    simAccount.Address.String(),
			ProviderId: provider.Id,
			Duration:   duration,
			Capacity:   capacity,
		}

		// Execute the message using the message server
		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.AgreementOpen(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// ============================================================================
// AGREEMENT OPERATIONS (continued)
// ============================================================================

// SimulateMsgAgreementClose generates a MsgAgreementClose with random values
func SimulateMsgAgreementClose(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementClose{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementClose{}), "player not found"), nil, nil
		}

		allAgreements := k.GetAllAgreement(ctx)
		validAgreements := make([]types.Agreement, 0)
		for _, agreement := range allAgreements {
			agreementCache := k.GetAgreementCacheFromId(ctx, agreement.Id)
			if agreementCache.CanUpdate(&activePlayer) == nil {
				validAgreements = append(validAgreements, agreement)
			}
		}

		if len(validAgreements) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementClose{}), "no closable agreements"), nil, nil
		}

		agreement := validAgreements[r.Intn(len(validAgreements))]
		msg := &types.MsgAgreementClose{
			Creator:     simAccount.Address.String(),
			AgreementId: agreement.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.AgreementClose(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgAgreementCapacityIncrease generates a MsgAgreementCapacityIncrease with random values
func SimulateMsgAgreementCapacityIncrease(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementCapacityIncrease{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementCapacityIncrease{}), "player not found"), nil, nil
		}

		allAgreements := k.GetAllAgreement(ctx)
		validAgreements := make([]types.Agreement, 0)
		for _, agreement := range allAgreements {
			agreementCache := k.GetAgreementCacheFromId(ctx, agreement.Id)
			if agreementCache.CanUpdate(&activePlayer) == nil {
				provider := agreementCache.GetProvider()
				if agreement.Capacity < provider.GetCapacityMaximum() {
					validAgreements = append(validAgreements, agreement)
				}
			}
		}

		if len(validAgreements) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementCapacityIncrease{}), "no updatable agreements"), nil, nil
		}

		agreement := validAgreements[r.Intn(len(validAgreements))]
		agreementCache := k.GetAgreementCacheFromId(ctx, agreement.Id)
		provider := agreementCache.GetProvider()

		capacityIncrease := uint64(r.Int63n(100) + 1)
		if agreement.Capacity+capacityIncrease > provider.GetCapacityMaximum() {
			capacityIncrease = provider.GetCapacityMaximum() - agreement.Capacity
			if capacityIncrease == 0 {
				return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementCapacityIncrease{}), "at max capacity"), nil, nil
			}
		}

		msg := &types.MsgAgreementCapacityIncrease{
			Creator:          simAccount.Address.String(),
			AgreementId:      agreement.Id,
			CapacityIncrease: capacityIncrease,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.AgreementCapacityIncrease(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgAgreementCapacityDecrease generates a MsgAgreementCapacityDecrease with random values
func SimulateMsgAgreementCapacityDecrease(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementCapacityDecrease{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementCapacityDecrease{}), "player not found"), nil, nil
		}

		allAgreements := k.GetAllAgreement(ctx)
		validAgreements := make([]types.Agreement, 0)
		for _, agreement := range allAgreements {
			agreementCache := k.GetAgreementCacheFromId(ctx, agreement.Id)
			if agreementCache.CanUpdate(&activePlayer) == nil {
				provider := agreementCache.GetProvider()
				if agreement.Capacity > provider.GetCapacityMinimum() {
					validAgreements = append(validAgreements, agreement)
				}
			}
		}

		if len(validAgreements) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementCapacityDecrease{}), "no decreasable agreements"), nil, nil
		}

		agreement := validAgreements[r.Intn(len(validAgreements))]
		agreementCache := k.GetAgreementCacheFromId(ctx, agreement.Id)
		provider := agreementCache.GetProvider()

		maxDecrease := agreement.Capacity - provider.GetCapacityMinimum()
		if maxDecrease == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementCapacityDecrease{}), "at min capacity"), nil, nil
		}

		capacityDecrease := uint64(r.Int63n(int64(maxDecrease)) + 1)

		msg := &types.MsgAgreementCapacityDecrease{
			Creator:          simAccount.Address.String(),
			AgreementId:      agreement.Id,
			CapacityDecrease: capacityDecrease,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.AgreementCapacityDecrease(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgAgreementDurationIncrease generates a MsgAgreementDurationIncrease with random values
func SimulateMsgAgreementDurationIncrease(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementDurationIncrease{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementDurationIncrease{}), "player not found"), nil, nil
		}

		allAgreements := k.GetAllAgreement(ctx)
		validAgreements := make([]types.Agreement, 0)
		for _, agreement := range allAgreements {
			agreementCache := k.GetAgreementCacheFromId(ctx, agreement.Id)
			if agreementCache.CanUpdate(&activePlayer) == nil {
				validAgreements = append(validAgreements, agreement)
			}
		}

		if len(validAgreements) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementDurationIncrease{}), "no updatable agreements"), nil, nil
		}

		agreement := validAgreements[r.Intn(len(validAgreements))]
		agreementCache := k.GetAgreementCacheFromId(ctx, agreement.Id)
		provider := agreementCache.GetProvider()

		durationIncrease := uint64(r.Int63n(50) + 1)

		rate := provider.GetRate()
		durationInt := math.NewIntFromUint64(durationIncrease)
		capacityInt := math.NewIntFromUint64(agreement.Capacity)
		collateralAmount := durationInt.Mul(capacityInt).Mul(rate.Amount)
		collateralCoin := sdk.NewCoin(rate.Denom, collateralAmount)

		sourceAcc, errParam := sdk.AccAddressFromBech32(simAccount.Address.String())
		if errParam != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementDurationIncrease{}), "invalid address"), nil, nil
		}

		if !bk.HasBalance(ctx, sourceAcc, collateralCoin) {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAgreementDurationIncrease{}), "insufficient balance"), nil, nil
		}

		msg := &types.MsgAgreementDurationIncrease{
			Creator:          simAccount.Address.String(),
			AgreementId:      agreement.Id,
			DurationIncrease: durationIncrease,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.AgreementDurationIncrease(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// ============================================================================
// ALLOCATION OPERATIONS
// ============================================================================

// SimulateMsgAllocationDelete generates a MsgAllocationDelete with random values
func SimulateMsgAllocationDelete(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAllocationDelete{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAllocationDelete{}), "player not found"), nil, nil
		}

		permissionError := activePlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionAssets)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAllocationDelete{}), "no assets permission"), nil, nil
		}

		allAllocations := k.GetAllAllocation(ctx)
		validAllocations := make([]types.Allocation, 0)
		for _, allocation := range allAllocations {
			if allocation.Controller == simAccount.Address.String() && allocation.Type == types.AllocationType_dynamic {
				validAllocations = append(validAllocations, allocation)
			}
		}

		if len(validAllocations) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAllocationDelete{}), "no deletable allocations"), nil, nil
		}

		allocation := validAllocations[r.Intn(len(validAllocations))]
		msg := &types.MsgAllocationDelete{
			Creator:      simAccount.Address.String(),
			AllocationId: allocation.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.AllocationDelete(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgAllocationUpdate generates a MsgAllocationUpdate with random values
func SimulateMsgAllocationUpdate(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAllocationUpdate{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAllocationUpdate{}), "player not found"), nil, nil
		}

		permissionError := activePlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionAssets)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAllocationUpdate{}), "no assets permission"), nil, nil
		}

		allAllocations := k.GetAllAllocation(ctx)
		validAllocations := make([]types.Allocation, 0)
		for _, allocation := range allAllocations {
			if allocation.Controller == simAccount.Address.String() && allocation.Type == types.AllocationType_dynamic {
				validAllocations = append(validAllocations, allocation)
			}
		}

		if len(validAllocations) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAllocationUpdate{}), "no updatable allocations"), nil, nil
		}

		allocation := validAllocations[r.Intn(len(validAllocations))]
		power := uint64(r.Int63n(1000) + 1)

		msg := &types.MsgAllocationUpdate{
			Creator:      simAccount.Address.String(),
			AllocationId: allocation.Id,
			Power:        power,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.AllocationUpdate(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgAllocationTransfer generates a MsgAllocationTransfer with random values
func SimulateMsgAllocationTransfer(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAllocationTransfer{}), "account not found"), nil, nil
		}

		allAllocations := k.GetAllAllocation(ctx)
		validAllocations := make([]types.Allocation, 0)
		for _, allocation := range allAllocations {
			if allocation.Controller == simAccount.Address.String() && allocation.DestinationId == "" {
				validAllocations = append(validAllocations, allocation)
			}
		}

		if len(validAllocations) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAllocationTransfer{}), "no transferable allocations"), nil, nil
		}

		allocation := validAllocations[r.Intn(len(validAllocations))]
		// Transfer to a random account (could be same or different)
		targetAccount, _ := simtypes.RandomAcc(r, accs)
		controller := targetAccount.Address.String()

		msg := &types.MsgAllocationTransfer{
			Creator:      simAccount.Address.String(),
			AllocationId: allocation.Id,
			Controller:   controller,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.AllocationTransfer(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// ============================================================================
// FLEET OPERATIONS
// ============================================================================

// SimulateMsgFleetMove generates a MsgFleetMove with random values
func SimulateMsgFleetMove(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgFleetMove{}), "account not found"), nil, nil
		}

		player := k.UpsertPlayer(ctx, simAccount.Address.String())
		playerCache, err := k.GetPlayerCacheFromId(ctx, player.Id)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgFleetMove{}), "player not found"), nil, nil
		}

		if !playerCache.HasPlanet() {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgFleetMove{}), "player has no planet"), nil, nil
		}

		fleet := playerCache.GetFleet()
		if fleet == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgFleetMove{}), "player has no fleet"), nil, nil
		}

		// Get all planets
		allPlanets := k.GetAllPlanet(ctx)
		if len(allPlanets) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgFleetMove{}), "no planets available"), nil, nil
		}

		// Pick a random planet (could be current location)
		destinationPlanet := allPlanets[r.Intn(len(allPlanets))]

		msg := &types.MsgFleetMove{
			Creator:               simAccount.Address.String(),
			FleetId:               fleet.GetFleetId(),
			DestinationLocationId: destinationPlanet.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.FleetMove(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// ============================================================================
// STRUCT OPERATIONS
// ============================================================================

// SimulateMsgStructBuildComplete generates a MsgStructBuildComplete with random values
func SimulateMsgStructBuildComplete(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildComplete{}), "account not found"), nil, nil
		}

		allStructs := k.GetAllStruct(ctx)
		validStructs := make([]types.Struct, 0)
		for _, structure := range allStructs {
			structCache := k.GetStructCacheFromId(ctx, structure.Id)
			if structCache.CanBePlayedBy(simAccount.Address.String()) == nil && !structCache.IsBuilt() {
				validStructs = append(validStructs, structure)
			}
		}

		if len(validStructs) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildComplete{}), "no buildable structs"), nil, nil
		}

		structure := validStructs[r.Intn(len(validStructs))]
		structCache := k.GetStructCacheFromId(ctx, structure.Id)

		// Calculate proof of work if difficulty is low enough
		currentAge := uint64(ctx.BlockHeight()) - structCache.GetBlockStartBuild()
		difficulty := types.CalculateDifficulty(float64(currentAge), structCache.GetStructType().BuildDifficulty)

		var proof string
		var nonce string
		if difficulty <= 2 {
			// Generate proof of work
			for i := 0; i < 10000; i++ {
				nonce = fmt.Sprintf("%d", r.Int63())
				hashInput := structure.Id + nonce
				proof = types.HashBuild(hashInput)
				if types.HashBuildAndCheckDifficulty(hashInput, proof, currentAge, structCache.GetStructType().BuildDifficulty) {
					break
				}
			}
		} else {
			// Difficulty too high, skip
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildComplete{}), "difficulty too high"), nil, nil
		}

		msg := &types.MsgStructBuildComplete{
			Creator:  simAccount.Address.String(),
			StructId: structure.Id,
			Proof:    proof,
			Nonce:    nonce,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.StructBuildComplete(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgStructBuildCancel generates a MsgStructBuildCancel with random values
func SimulateMsgStructBuildCancel(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildCancel{}), "account not found"), nil, nil
		}

		allStructs := k.GetAllStruct(ctx)
		validStructs := make([]types.Struct, 0)
		for _, structure := range allStructs {
			structCache := k.GetStructCacheFromId(ctx, structure.Id)
			if structCache.CanBePlayedBy(simAccount.Address.String()) == nil && !structCache.IsBuilt() {
				validStructs = append(validStructs, structure)
			}
		}

		if len(validStructs) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructBuildCancel{}), "no cancellable structs"), nil, nil
		}

		structure := validStructs[r.Intn(len(validStructs))]

		msg := &types.MsgStructBuildCancel{
			Creator:  simAccount.Address.String(),
			StructId: structure.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.StructBuildCancel(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgStructActivate generates a MsgStructActivate with random values
func SimulateMsgStructActivate(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructActivate{}), "account not found"), nil, nil
		}

		allStructs := k.GetAllStruct(ctx)
		validStructs := make([]types.Struct, 0)
		for _, structure := range allStructs {
			structCache := k.GetStructCacheFromId(ctx, structure.Id)
			if structCache.CanBePlayedBy(simAccount.Address.String()) == nil && structCache.IsBuilt() && structCache.IsOffline() {
				validStructs = append(validStructs, structure)
			}
		}

		if len(validStructs) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructActivate{}), "no activatable structs"), nil, nil
		}

		structure := validStructs[r.Intn(len(validStructs))]

		msg := &types.MsgStructActivate{
			Creator:  simAccount.Address.String(),
			StructId: structure.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.StructActivate(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgStructDeactivate generates a MsgStructDeactivate with random values
func SimulateMsgStructDeactivate(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructDeactivate{}), "account not found"), nil, nil
		}

		allStructs := k.GetAllStruct(ctx)
		validStructs := make([]types.Struct, 0)
		for _, structure := range allStructs {
			structCache := k.GetStructCacheFromId(ctx, structure.Id)
			if structCache.CanBePlayedBy(simAccount.Address.String()) == nil && structCache.IsBuilt() && !structCache.IsOffline() {
				validStructs = append(validStructs, structure)
			}
		}

		if len(validStructs) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgStructDeactivate{}), "no deactivatable structs"), nil, nil
		}

		structure := validStructs[r.Intn(len(validStructs))]

		msg := &types.MsgStructDeactivate{
			Creator:  simAccount.Address.String(),
			StructId: structure.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.StructDeactivate(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// ============================================================================
// PROVIDER OPERATIONS
// ============================================================================

// SimulateMsgProviderWithdrawBalance generates a MsgProviderWithdrawBalance with random values
func SimulateMsgProviderWithdrawBalance(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderWithdrawBalance{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderWithdrawBalance{}), "player not found"), nil, nil
		}

		allProviders := k.GetAllProvider(ctx)
		validProviders := make([]types.Provider, 0)
		for _, provider := range allProviders {
			providerCache := k.GetProviderCacheFromId(ctx, provider.Id)
			if providerCache.CanWithdrawBalance(&activePlayer) == nil {
				validProviders = append(validProviders, provider)
			}
		}

		if len(validProviders) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderWithdrawBalance{}), "no withdrawable providers"), nil, nil
		}

		provider := validProviders[r.Intn(len(validProviders))]
		destAccount, _ := simtypes.RandomAcc(r, accs)

		msg := &types.MsgProviderWithdrawBalance{
			Creator:            simAccount.Address.String(),
			ProviderId:         provider.Id,
			DestinationAddress: destAccount.Address.String(),
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.ProviderWithdrawBalance(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgProviderUpdateCapacityMinimum generates a MsgProviderUpdateCapacityMinimum with random values
func SimulateMsgProviderUpdateCapacityMinimum(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderUpdateCapacityMinimum{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderUpdateCapacityMinimum{}), "player not found"), nil, nil
		}

		allProviders := k.GetAllProvider(ctx)
		validProviders := make([]types.Provider, 0)
		for _, provider := range allProviders {
			providerCache := k.GetProviderCacheFromId(ctx, provider.Id)
			if providerCache.GetOwnerId() == activePlayer.GetPlayerId() {
				validProviders = append(validProviders, provider)
			}
		}

		if len(validProviders) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderUpdateCapacityMinimum{}), "no owned providers"), nil, nil
		}

		provider := validProviders[r.Intn(len(validProviders))]
		providerCache := k.GetProviderCacheFromId(ctx, provider.Id)
		newMin := uint64(r.Int63n(int64(providerCache.GetCapacityMaximum())) + 1)

		msg := &types.MsgProviderUpdateCapacityMinimum{
			Creator:            simAccount.Address.String(),
			ProviderId:         provider.Id,
			NewMinimumCapacity: newMin,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.ProviderUpdateCapacityMinimum(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgProviderUpdateCapacityMaximum generates a MsgProviderUpdateCapacityMaximum with random values
func SimulateMsgProviderUpdateCapacityMaximum(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderUpdateCapacityMaximum{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderUpdateCapacityMaximum{}), "player not found"), nil, nil
		}

		allProviders := k.GetAllProvider(ctx)
		validProviders := make([]types.Provider, 0)
		for _, provider := range allProviders {
			providerCache := k.GetProviderCacheFromId(ctx, provider.Id)
			if providerCache.GetOwnerId() == activePlayer.GetPlayerId() {
				validProviders = append(validProviders, provider)
			}
		}

		if len(validProviders) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderUpdateCapacityMaximum{}), "no owned providers"), nil, nil
		}

		provider := validProviders[r.Intn(len(validProviders))]
		providerCache := k.GetProviderCacheFromId(ctx, provider.Id)
		newMax := providerCache.GetCapacityMinimum() + uint64(r.Int63n(900)+1)

		msg := &types.MsgProviderUpdateCapacityMaximum{
			Creator:            simAccount.Address.String(),
			ProviderId:         provider.Id,
			NewMaximumCapacity: newMax,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.ProviderUpdateCapacityMaximum(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgProviderUpdateDurationMinimum generates a MsgProviderUpdateDurationMinimum with random values
func SimulateMsgProviderUpdateDurationMinimum(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderUpdateDurationMinimum{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderUpdateDurationMinimum{}), "player not found"), nil, nil
		}

		allProviders := k.GetAllProvider(ctx)
		validProviders := make([]types.Provider, 0)
		for _, provider := range allProviders {
			providerCache := k.GetProviderCacheFromId(ctx, provider.Id)
			if providerCache.GetOwnerId() == activePlayer.GetPlayerId() {
				validProviders = append(validProviders, provider)
			}
		}

		if len(validProviders) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderUpdateDurationMinimum{}), "no owned providers"), nil, nil
		}

		provider := validProviders[r.Intn(len(validProviders))]
		providerCache := k.GetProviderCacheFromId(ctx, provider.Id)
		newMin := uint64(r.Int63n(int64(providerCache.GetDurationMaximum())) + 1)

		msg := &types.MsgProviderUpdateDurationMinimum{
			Creator:            simAccount.Address.String(),
			ProviderId:         provider.Id,
			NewMinimumDuration: newMin,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.ProviderUpdateDurationMinimum(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgProviderUpdateDurationMaximum generates a MsgProviderUpdateDurationMaximum with random values
func SimulateMsgProviderUpdateDurationMaximum(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderUpdateDurationMaximum{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderUpdateDurationMaximum{}), "player not found"), nil, nil
		}

		allProviders := k.GetAllProvider(ctx)
		validProviders := make([]types.Provider, 0)
		for _, provider := range allProviders {
			providerCache := k.GetProviderCacheFromId(ctx, provider.Id)
			if providerCache.GetOwnerId() == activePlayer.GetPlayerId() {
				validProviders = append(validProviders, provider)
			}
		}

		if len(validProviders) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderUpdateDurationMaximum{}), "no owned providers"), nil, nil
		}

		provider := validProviders[r.Intn(len(validProviders))]
		providerCache := k.GetProviderCacheFromId(ctx, provider.Id)
		newMax := providerCache.GetDurationMinimum() + uint64(r.Int63n(90)+1)

		msg := &types.MsgProviderUpdateDurationMaximum{
			Creator:            simAccount.Address.String(),
			ProviderId:         provider.Id,
			NewMaximumDuration: newMax,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.ProviderUpdateDurationMaximum(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgProviderUpdateAccessPolicy generates a MsgProviderUpdateAccessPolicy with random values
func SimulateMsgProviderUpdateAccessPolicy(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderUpdateAccessPolicy{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderUpdateAccessPolicy{}), "player not found"), nil, nil
		}

		allProviders := k.GetAllProvider(ctx)
		validProviders := make([]types.Provider, 0)
		for _, provider := range allProviders {
			providerCache := k.GetProviderCacheFromId(ctx, provider.Id)
			if providerCache.GetOwnerId() == activePlayer.GetPlayerId() {
				validProviders = append(validProviders, provider)
			}
		}

		if len(validProviders) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderUpdateAccessPolicy{}), "no owned providers"), nil, nil
		}

		provider := validProviders[r.Intn(len(validProviders))]
		accessPolicies := []types.ProviderAccessPolicy{
			types.ProviderAccessPolicy_openMarket,
			types.ProviderAccessPolicy_guildMarket,
			types.ProviderAccessPolicy_closedMarket,
		}
		accessPolicy := accessPolicies[r.Intn(len(accessPolicies))]

		msg := &types.MsgProviderUpdateAccessPolicy{
			Creator:      simAccount.Address.String(),
			ProviderId:   provider.Id,
			AccessPolicy: accessPolicy,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.ProviderUpdateAccessPolicy(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgProviderGuildGrant generates a MsgProviderGuildGrant with random values
func SimulateMsgProviderGuildGrant(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderGuildGrant{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderGuildGrant{}), "player not found"), nil, nil
		}

		allProviders := k.GetAllProvider(ctx)
		validProviders := make([]types.Provider, 0)
		for _, provider := range allProviders {
			providerCache := k.GetProviderCacheFromId(ctx, provider.Id)
			if providerCache.GetOwnerId() == activePlayer.GetPlayerId() {
				validProviders = append(validProviders, provider)
			}
		}

		if len(validProviders) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderGuildGrant{}), "no owned providers"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		if len(allGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderGuildGrant{}), "no guilds available"), nil, nil
		}

		provider := validProviders[r.Intn(len(validProviders))]
		guild := allGuilds[r.Intn(len(allGuilds))]

		msg := &types.MsgProviderGuildGrant{
			Creator:    simAccount.Address.String(),
			ProviderId: provider.Id,
			GuildId:    []string{guild.Id},
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.ProviderGuildGrant(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgProviderGuildRevoke generates a MsgProviderGuildRevoke with random values
func SimulateMsgProviderGuildRevoke(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderGuildRevoke{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderGuildRevoke{}), "player not found"), nil, nil
		}

		allProviders := k.GetAllProvider(ctx)
		validProviders := make([]types.Provider, 0)
		for _, provider := range allProviders {
			providerCache := k.GetProviderCacheFromId(ctx, provider.Id)
			if providerCache.GetOwnerId() == activePlayer.GetPlayerId() {
				validProviders = append(validProviders, provider)
			}
		}

		if len(validProviders) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderGuildRevoke{}), "no owned providers"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		if len(allGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderGuildRevoke{}), "no guilds available"), nil, nil
		}

		provider := validProviders[r.Intn(len(validProviders))]
		guild := allGuilds[r.Intn(len(allGuilds))]

		msg := &types.MsgProviderGuildRevoke{
			Creator:    simAccount.Address.String(),
			ProviderId: provider.Id,
			GuildId:    []string{guild.Id},
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.ProviderGuildRevoke(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgProviderDelete generates a MsgProviderDelete with random values
func SimulateMsgProviderDelete(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderDelete{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderDelete{}), "player not found"), nil, nil
		}

		allProviders := k.GetAllProvider(ctx)
		validProviders := make([]types.Provider, 0)
		for _, provider := range allProviders {
			providerCache := k.GetProviderCacheFromId(ctx, provider.Id)
			if providerCache.GetOwnerId() == activePlayer.GetPlayerId() {
				validProviders = append(validProviders, provider)
			}
		}

		if len(validProviders) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgProviderDelete{}), "no owned providers"), nil, nil
		}

		provider := validProviders[r.Intn(len(validProviders))]

		msg := &types.MsgProviderDelete{
			Creator:    simAccount.Address.String(),
			ProviderId: provider.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.ProviderDelete(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// ============================================================================
// SUBSTATION OPERATIONS
// ============================================================================

// SimulateMsgSubstationAllocationConnect generates a MsgSubstationAllocationConnect with random values
func SimulateMsgSubstationAllocationConnect(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationAllocationConnect{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationAllocationConnect{}), "player not found"), nil, nil
		}

		permissionError := activePlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionGrid)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationAllocationConnect{}), "no grid permission"), nil, nil
		}

		allAllocations := k.GetAllAllocation(ctx)
		validAllocations := make([]types.Allocation, 0)
		for _, allocation := range allAllocations {
			if allocation.Controller == simAccount.Address.String() && allocation.DestinationId == "" {
				validAllocations = append(validAllocations, allocation)
			}
		}

		if len(validAllocations) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationAllocationConnect{}), "no connectable allocations"), nil, nil
		}

		allSubstations := k.GetAllSubstation(ctx)
		if len(allSubstations) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationAllocationConnect{}), "no substations available"), nil, nil
		}

		allocation := validAllocations[r.Intn(len(validAllocations))]
		substation := allSubstations[r.Intn(len(allSubstations))]

		msg := &types.MsgSubstationAllocationConnect{
			Creator:       simAccount.Address.String(),
			AllocationId:  allocation.Id,
			DestinationId: substation.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.SubstationAllocationConnect(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgSubstationAllocationDisconnect generates a MsgSubstationAllocationDisconnect with random values
func SimulateMsgSubstationAllocationDisconnect(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationAllocationDisconnect{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationAllocationDisconnect{}), "player not found"), nil, nil
		}

		permissionError := activePlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionGrid)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationAllocationDisconnect{}), "no grid permission"), nil, nil
		}

		allAllocations := k.GetAllAllocation(ctx)
		validAllocations := make([]types.Allocation, 0)
		for _, allocation := range allAllocations {
			if (allocation.Controller == simAccount.Address.String() && allocation.DestinationId != "") ||
				(allocation.DestinationId != "" && k.PermissionHasOneOf(ctx, keeper.GetObjectPermissionIDBytes(allocation.DestinationId, activePlayer.GetPlayerId()), types.PermissionGrid)) {
				validAllocations = append(validAllocations, allocation)
			}
		}

		if len(validAllocations) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationAllocationDisconnect{}), "no disconnectable allocations"), nil, nil
		}

		allocation := validAllocations[r.Intn(len(validAllocations))]

		msg := &types.MsgSubstationAllocationDisconnect{
			Creator:      simAccount.Address.String(),
			AllocationId: allocation.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.SubstationAllocationDisconnect(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgSubstationPlayerConnect generates a MsgSubstationPlayerConnect with random values
func SimulateMsgSubstationPlayerConnect(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationPlayerConnect{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationPlayerConnect{}), "player not found"), nil, nil
		}

		permissionError := activePlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionGrid)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationPlayerConnect{}), "no grid permission"), nil, nil
		}

		allSubstations := k.GetAllSubstation(ctx)
		validSubstations := make([]types.Substation, 0)
		for _, substation := range allSubstations {
			substationCache := k.GetSubstationCacheFromId(ctx, substation.Id)
			if substationCache.CanManagePlayerConnections(&activePlayer) == nil {
				validSubstations = append(validSubstations, substation)
			}
		}

		if len(validSubstations) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationPlayerConnect{}), "no accessible substations"), nil, nil
		}

		allPlayers := k.GetAllPlayer(ctx)
		if len(allPlayers) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationPlayerConnect{}), "no players available"), nil, nil
		}

		substation := validSubstations[r.Intn(len(validSubstations))]
		targetPlayer := allPlayers[r.Intn(len(allPlayers))]

		msg := &types.MsgSubstationPlayerConnect{
			Creator:      simAccount.Address.String(),
			SubstationId: substation.Id,
			PlayerId:     targetPlayer.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.SubstationPlayerConnect(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgSubstationPlayerDisconnect generates a MsgSubstationPlayerDisconnect with random values
func SimulateMsgSubstationPlayerDisconnect(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationPlayerDisconnect{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationPlayerDisconnect{}), "player not found"), nil, nil
		}

		permissionError := activePlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionGrid)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationPlayerDisconnect{}), "no grid permission"), nil, nil
		}

		allPlayers := k.GetAllPlayer(ctx)
		validPlayers := make([]types.Player, 0)
		for _, player := range allPlayers {
			if player.SubstationId != "" {
				substationCache := k.GetSubstationCacheFromId(ctx, player.SubstationId)
				if substationCache.CanManagePlayerConnections(&activePlayer) == nil {
					validPlayers = append(validPlayers, player)
				}
			}
		}

		if len(validPlayers) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationPlayerDisconnect{}), "no disconnectable players"), nil, nil
		}

		targetPlayer := validPlayers[r.Intn(len(validPlayers))]

		msg := &types.MsgSubstationPlayerDisconnect{
			Creator:  simAccount.Address.String(),
			PlayerId: targetPlayer.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.SubstationPlayerDisconnect(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgSubstationPlayerMigrate generates a MsgSubstationPlayerMigrate with random values
func SimulateMsgSubstationPlayerMigrate(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationPlayerMigrate{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationPlayerMigrate{}), "player not found"), nil, nil
		}

		permissionError := activePlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionGrid)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationPlayerMigrate{}), "no grid permission"), nil, nil
		}

		allSubstations := k.GetAllSubstation(ctx)
		validSubstations := make([]types.Substation, 0)
		for _, substation := range allSubstations {
			substationCache := k.GetSubstationCacheFromId(ctx, substation.Id)
			if substationCache.CanManagePlayerConnections(&activePlayer) == nil {
				validSubstations = append(validSubstations, substation)
			}
		}

		if len(validSubstations) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationPlayerMigrate{}), "no accessible substations"), nil, nil
		}

		allPlayers := k.GetAllPlayer(ctx)
		connectedPlayers := make([]string, 0)
		for _, player := range allPlayers {
			if player.SubstationId != "" {
				connectedPlayers = append(connectedPlayers, player.Id)
			}
		}

		if len(connectedPlayers) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationPlayerMigrate{}), "no connected players"), nil, nil
		}

		substation := validSubstations[r.Intn(len(validSubstations))]
		// Migrate 1-3 random players
		numToMigrate := r.Intn(3) + 1
		if numToMigrate > len(connectedPlayers) {
			numToMigrate = len(connectedPlayers)
		}
		playersToMigrate := make([]string, numToMigrate)
		for i := 0; i < numToMigrate; i++ {
			playersToMigrate[i] = connectedPlayers[r.Intn(len(connectedPlayers))]
		}

		msg := &types.MsgSubstationPlayerMigrate{
			Creator:      simAccount.Address.String(),
			SubstationId: substation.Id,
			PlayerId:     playersToMigrate,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.SubstationPlayerMigrate(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgSubstationDelete generates a MsgSubstationDelete with random values
func SimulateMsgSubstationDelete(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationDelete{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationDelete{}), "player not found"), nil, nil
		}

		allSubstations := k.GetAllSubstation(ctx)
		validSubstations := make([]types.Substation, 0)
		for _, substation := range allSubstations {
			substationCache := k.GetSubstationCacheFromId(ctx, substation.Id)
			if substationCache.CanBeDeleteDBy(&activePlayer) == nil {
				validSubstations = append(validSubstations, substation)
			}
		}

		if len(validSubstations) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubstationDelete{}), "no deletable substations"), nil, nil
		}

		substation := validSubstations[r.Intn(len(validSubstations))]
		// Pick a migration substation (could be empty)
		migrationSubstationId := ""
		if len(allSubstations) > 1 && r.Intn(2) == 0 {
			for _, s := range allSubstations {
				if s.Id != substation.Id {
					migrationSubstationId = s.Id
					break
				}
			}
		}

		msg := &types.MsgSubstationDelete{
			Creator:               simAccount.Address.String(),
			SubstationId:          substation.Id,
			MigrationSubstationId: migrationSubstationId,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.SubstationDelete(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// ============================================================================
// ADDRESS OPERATIONS
// ============================================================================

// SimulateMsgAddressRevoke generates a MsgAddressRevoke with random values
func SimulateMsgAddressRevoke(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAddressRevoke{}), "account not found"), nil, nil
		}

		allAddressRecords := k.GetAllAddressExport(ctx)
		validAddresses := make([]*types.AddressRecord, 0)
		for _, addrRecord := range allAddressRecords {
			player, err := k.GetPlayerCacheFromAddress(ctx, addrRecord.Address)
			if err == nil {
				if player.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionDelete) == nil {
					if player.GetPrimaryAddress() != addrRecord.Address {
						validAddresses = append(validAddresses, addrRecord)
					}
				}
			}
		}

		if len(validAddresses) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgAddressRevoke{}), "no revokable addresses"), nil, nil
		}

		addrRecord := validAddresses[r.Intn(len(validAddresses))]

		msg := &types.MsgAddressRevoke{
			Creator: simAccount.Address.String(),
			Address: addrRecord.Address,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.AddressRevoke(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// ============================================================================
// PLAYER OPERATIONS
// ============================================================================

// SimulateMsgPlayerUpdatePrimaryAddress generates a MsgPlayerUpdatePrimaryAddress with random values
func SimulateMsgPlayerUpdatePrimaryAddress(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgPlayerUpdatePrimaryAddress{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgPlayerUpdatePrimaryAddress{}), "player not found"), nil, nil
		}

		// Get all addresses for this player by querying
		allAddressRecords := k.GetAllAddressExport(ctx)
		playerAddresses := make([]string, 0)
		for _, addrRecord := range allAddressRecords {
			playerIndex := k.GetPlayerIndexFromAddress(ctx, addrRecord.Address)
			playerId := keeper.GetObjectID(types.ObjectType_player, playerIndex)
			if playerId == activePlayer.GetPlayerId() {
				playerAddresses = append(playerAddresses, addrRecord.Address)
			}
		}

		if len(playerAddresses) < 2 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgPlayerUpdatePrimaryAddress{}), "player has less than 2 addresses"), nil, nil
		}

		// Pick a different address as primary
		var newPrimary string
		for _, addr := range playerAddresses {
			if addr != activePlayer.GetPrimaryAddress() {
				newPrimary = addr
				break
			}
		}

		if newPrimary == "" {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgPlayerUpdatePrimaryAddress{}), "no alternative address"), nil, nil
		}

		msg := &types.MsgPlayerUpdatePrimaryAddress{
			Creator:        simAccount.Address.String(),
			PlayerId:       activePlayer.GetPlayerId(),
			PrimaryAddress: newPrimary,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.PlayerUpdatePrimaryAddress(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgPlayerResume generates a MsgPlayerResume with random values
func SimulateMsgPlayerResume(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgPlayerResume{}), "account not found"), nil, nil
		}

		allPlayers := k.GetAllPlayer(ctx)
		haltedPlayers := make([]types.Player, 0)
		for _, player := range allPlayers {
			playerCache, err := k.GetPlayerCacheFromId(ctx, player.Id)
			if err == nil && playerCache.IsHalted() {
				if playerCache.CanBeUpdatedBy(simAccount.Address.String()) == nil {
					haltedPlayers = append(haltedPlayers, player)
				}
			}
		}

		if len(haltedPlayers) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgPlayerResume{}), "no resumable players"), nil, nil
		}

		player := haltedPlayers[r.Intn(len(haltedPlayers))]

		msg := &types.MsgPlayerResume{
			Creator:  simAccount.Address.String(),
			PlayerId: player.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.PlayerResume(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// ============================================================================
// PLANET OPERATIONS
// ============================================================================

// SimulateMsgPlanetRaidComplete generates a MsgPlanetRaidComplete with random values
func SimulateMsgPlanetRaidComplete(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgPlanetRaidComplete{}), "account not found"), nil, nil
		}

		player := k.UpsertPlayer(ctx, simAccount.Address.String())
		playerCache, err := k.GetPlayerCacheFromId(ctx, player.Id)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgPlanetRaidComplete{}), "player not found"), nil, nil
		}

		if !playerCache.HasPlanet() {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgPlanetRaidComplete{}), "player has no planet"), nil, nil
		}

		fleet := playerCache.GetFleet()
		if fleet == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgPlanetRaidComplete{}), "player has no fleet"), nil, nil
		}

		// Generate proof and nonce (simplified for simulation)
		nonce := fmt.Sprintf("%d", r.Int63())
		proof := types.HashBuild(fleet.GetFleetId() + nonce)

		msg := &types.MsgPlanetRaidComplete{
			Creator: simAccount.Address.String(),
			FleetId: fleet.GetFleetId(),
			Proof:   proof,
			Nonce:   nonce,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.PlanetRaidComplete(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// ============================================================================
// REACTOR OPERATIONS
// ============================================================================

// SimulateMsgReactorDefuse generates a MsgReactorDefuse with random values
func SimulateMsgReactorDefuse(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorDefuse{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorDefuse{}), "player not found"), nil, nil
		}

		permissionError := activePlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionAssets)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorDefuse{}), "no assets permission"), nil, nil
		}

		// Get all reactors and find one the player can defuse from
		// Get all reactors - simplified for simulation
		allReactors := k.GetAllReactor(ctx)
		if len(allReactors) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorDefuse{}), "no reactors available"), nil, nil
		}

		reactor := allReactors[r.Intn(len(allReactors))]
		// Use a reasonable defuse amount for simulation
		defuseAmount := math.NewInt(1000 + int64(r.Intn(9000)))

		msg := &types.MsgReactorDefuse{
			Creator:          simAccount.Address.String(),
			DelegatorAddress: activePlayer.GetPrimaryAddress(),
			ValidatorAddress: reactor.Validator,
			Amount:           sdk.NewCoin("ualpha", defuseAmount),
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.ReactorDefuse(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgReactorBeginMigration generates a MsgReactorBeginMigration with random values
func SimulateMsgReactorBeginMigration(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorBeginMigration{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorBeginMigration{}), "player not found"), nil, nil
		}

		permissionError := activePlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionAssets)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorBeginMigration{}), "no assets permission"), nil, nil
		}

		allReactors := k.GetAllReactor(ctx)
		if len(allReactors) < 2 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorBeginMigration{}), "need at least 2 reactors"), nil, nil
		}

		// Pick source and destination reactors
		srcReactor := allReactors[r.Intn(len(allReactors))]
		var dstReactor types.Reactor
		for _, reactor := range allReactors {
			if reactor.Id != srcReactor.Id {
				dstReactor = reactor
				break
			}
		}
		if dstReactor.Id == "" {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorBeginMigration{}), "need at least 2 different reactors"), nil, nil
		}

		// Use a reasonable migration amount for simulation
		migrateAmount := math.NewInt(1000 + int64(r.Intn(9000)))

		msg := &types.MsgReactorBeginMigration{
			Creator:             simAccount.Address.String(),
			DelegatorAddress:    activePlayer.GetPrimaryAddress(),
			ValidatorSrcAddress: srcReactor.Validator,
			ValidatorDstAddress: dstReactor.Validator,
			Amount:              sdk.NewCoin("ualpha", migrateAmount),
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.ReactorBeginMigration(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgReactorCancelDefusion generates a MsgReactorCancelDefusion with random values
func SimulateMsgReactorCancelDefusion(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorCancelDefusion{}), "account not found"), nil, nil
		}

		activePlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorCancelDefusion{}), "player not found"), nil, nil
		}

		permissionError := activePlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionAssets)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorCancelDefusion{}), "no assets permission"), nil, nil
		}

		// Simplified for simulation - use a random reactor
		allReactors := k.GetAllReactor(ctx)
		if len(allReactors) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgReactorCancelDefusion{}), "no reactors available"), nil, nil
		}

		reactor := allReactors[r.Intn(len(allReactors))]
		cancelAmount := math.NewInt(1000 + int64(r.Intn(9000)))

		msg := &types.MsgReactorCancelDefusion{
			Creator:          simAccount.Address.String(),
			DelegatorAddress: activePlayer.GetPrimaryAddress(),
			ValidatorAddress: reactor.Validator,
			Amount:           sdk.NewCoin("ualpha", cancelAmount),
			CreationHeight:   ctx.BlockHeight() - 10, // Simulate an older unbonding
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.ReactorCancelDefusion(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// ============================================================================
// GUILD MEMBERSHIP OPERATIONS
// ============================================================================

// SimulateMsgGuildMembershipInvite generates a MsgGuildMembershipInvite with random values
func SimulateMsgGuildMembershipInvite(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInvite{}), "account not found"), nil, nil
		}

		callingPlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInvite{}), "player not found"), nil, nil
		}

		permissionError := callingPlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionAssociations)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInvite{}), "no associations permission"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		if len(allGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInvite{}), "no guilds available"), nil, nil
		}

		allPlayers := k.GetAllPlayer(ctx)
		if len(allPlayers) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInvite{}), "no players available"), nil, nil
		}

		guild := allGuilds[r.Intn(len(allGuilds))]
		targetPlayer := allPlayers[r.Intn(len(allPlayers))]

		msg := &types.MsgGuildMembershipInvite{
			Creator:      simAccount.Address.String(),
			GuildId:      guild.Id,
			PlayerId:     targetPlayer.Id,
			SubstationId: "",
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.GuildMembershipInvite(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildMembershipInviteApprove generates a MsgGuildMembershipInviteApprove with random values
func SimulateMsgGuildMembershipInviteApprove(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInviteApprove{}), "account not found"), nil, nil
		}

		callingPlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInviteApprove{}), "player not found"), nil, nil
		}

		permissionError := callingPlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionAssociations)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInviteApprove{}), "no associations permission"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		if len(allGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInviteApprove{}), "no guilds available"), nil, nil
		}

		guild := allGuilds[r.Intn(len(allGuilds))]

		msg := &types.MsgGuildMembershipInviteApprove{
			Creator:      simAccount.Address.String(),
			GuildId:      guild.Id,
			PlayerId:     callingPlayer.GetPlayerId(),
			SubstationId: "",
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.GuildMembershipInviteApprove(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildMembershipInviteDeny generates a MsgGuildMembershipInviteDeny with random values
func SimulateMsgGuildMembershipInviteDeny(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInviteDeny{}), "account not found"), nil, nil
		}

		callingPlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInviteDeny{}), "player not found"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		if len(allGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInviteDeny{}), "no guilds available"), nil, nil
		}

		guild := allGuilds[r.Intn(len(allGuilds))]

		msg := &types.MsgGuildMembershipInviteDeny{
			Creator:  simAccount.Address.String(),
			GuildId:  guild.Id,
			PlayerId: callingPlayer.GetPlayerId(),
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.GuildMembershipInviteDeny(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildMembershipInviteRevoke generates a MsgGuildMembershipInviteRevoke with random values
func SimulateMsgGuildMembershipInviteRevoke(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInviteRevoke{}), "account not found"), nil, nil
		}

		callingPlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInviteRevoke{}), "player not found"), nil, nil
		}

		permissionError := callingPlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionAssociations)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInviteRevoke{}), "no associations permission"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		if len(allGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInviteRevoke{}), "no guilds available"), nil, nil
		}

		allPlayers := k.GetAllPlayer(ctx)
		if len(allPlayers) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipInviteRevoke{}), "no players available"), nil, nil
		}

		guild := allGuilds[r.Intn(len(allGuilds))]
		targetPlayer := allPlayers[r.Intn(len(allPlayers))]

		msg := &types.MsgGuildMembershipInviteRevoke{
			Creator:  simAccount.Address.String(),
			GuildId:  guild.Id,
			PlayerId: targetPlayer.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.GuildMembershipInviteRevoke(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildMembershipJoinProxy generates a MsgGuildMembershipJoinProxy with random values
func SimulateMsgGuildMembershipJoinProxy(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipJoinProxy{}), "account not found"), nil, nil
		}

		proxyPlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipJoinProxy{}), "player not found"), nil, nil
		}

		permissionError := proxyPlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionAssociations)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipJoinProxy{}), "no associations permission"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		if len(allGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipJoinProxy{}), "no guilds available"), nil, nil
		}

		allSubstations := k.GetAllSubstation(ctx)
		if len(allSubstations) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipJoinProxy{}), "no substations available"), nil, nil
		}

		substation := allSubstations[r.Intn(len(allSubstations))]

		// Generate a new address for the proxy player
		newAccount, _ := simtypes.RandomAcc(r, accs)

		// Generate proof (simplified for simulation)
		proofPubKey := fmt.Sprintf("proof_%d", r.Int63())
		proofSignature := fmt.Sprintf("sig_%d", r.Int63())

		msg := &types.MsgGuildMembershipJoinProxy{
			Creator:        simAccount.Address.String(),
			Address:        newAccount.Address.String(),
			SubstationId:   substation.Id,
			ProofPubKey:    proofPubKey,
			ProofSignature: proofSignature,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.GuildMembershipJoinProxy(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildMembershipKick generates a MsgGuildMembershipKick with random values
func SimulateMsgGuildMembershipKick(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipKick{}), "account not found"), nil, nil
		}

		callingPlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipKick{}), "player not found"), nil, nil
		}

		permissionError := callingPlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionAssociations)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipKick{}), "no associations permission"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		if len(allGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipKick{}), "no guilds available"), nil, nil
		}

		allPlayers := k.GetAllPlayer(ctx)
		if len(allPlayers) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipKick{}), "no players available"), nil, nil
		}

		guild := allGuilds[r.Intn(len(allGuilds))]
		targetPlayer := allPlayers[r.Intn(len(allPlayers))]

		msg := &types.MsgGuildMembershipKick{
			Creator:  simAccount.Address.String(),
			GuildId:  guild.Id,
			PlayerId: targetPlayer.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.GuildMembershipKick(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildMembershipRequestApprove generates a MsgGuildMembershipRequestApprove with random values
func SimulateMsgGuildMembershipRequestApprove(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipRequestApprove{}), "account not found"), nil, nil
		}

		callingPlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipRequestApprove{}), "player not found"), nil, nil
		}

		permissionError := callingPlayer.CanBeAdministratedBy(simAccount.Address.String(), types.PermissionAssociations)
		if permissionError != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipRequestApprove{}), "no associations permission"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		if len(allGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipRequestApprove{}), "no guilds available"), nil, nil
		}

		guild := allGuilds[r.Intn(len(allGuilds))]

		msg := &types.MsgGuildMembershipRequestApprove{
			Creator:      simAccount.Address.String(),
			GuildId:      guild.Id,
			PlayerId:     callingPlayer.GetPlayerId(),
			SubstationId: "",
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.GuildMembershipRequestApprove(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildMembershipRequestDeny generates a MsgGuildMembershipRequestDeny with random values
func SimulateMsgGuildMembershipRequestDeny(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipRequestDeny{}), "account not found"), nil, nil
		}

		callingPlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipRequestDeny{}), "player not found"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		if len(allGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipRequestDeny{}), "no guilds available"), nil, nil
		}

		guild := allGuilds[r.Intn(len(allGuilds))]

		msg := &types.MsgGuildMembershipRequestDeny{
			Creator:  simAccount.Address.String(),
			GuildId:  guild.Id,
			PlayerId: callingPlayer.GetPlayerId(),
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.GuildMembershipRequestDeny(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildMembershipRequestRevoke generates a MsgGuildMembershipRequestRevoke with random values
func SimulateMsgGuildMembershipRequestRevoke(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipRequestRevoke{}), "account not found"), nil, nil
		}

		callingPlayer, err := k.GetPlayerCacheFromAddress(ctx, simAccount.Address.String())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipRequestRevoke{}), "player not found"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		if len(allGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildMembershipRequestRevoke{}), "no guilds available"), nil, nil
		}

		guild := allGuilds[r.Intn(len(allGuilds))]

		msg := &types.MsgGuildMembershipRequestRevoke{
			Creator:  simAccount.Address.String(),
			GuildId:  guild.Id,
			PlayerId: callingPlayer.GetPlayerId(),
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err = msgServer.GuildMembershipRequestRevoke(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// ============================================================================
// GUILD UPDATE OPERATIONS
// ============================================================================

// SimulateMsgGuildUpdateOwnerId generates a MsgGuildUpdateOwnerId with random values
func SimulateMsgGuildUpdateOwnerId(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateOwnerId{}), "account not found"), nil, nil
		}

		player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, simAccount.Address.String()))
		if !playerFound {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateOwnerId{}), "player not found"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		validGuilds := make([]types.Guild, 0)
		for _, guild := range allGuilds {
			guildObjectPermissionId := keeper.GetObjectPermissionIDBytes(guild.Id, player.Id)
			addressPermissionId := keeper.GetAddressPermissionIDBytes(simAccount.Address.String())
			if k.PermissionHasOneOf(ctx, guildObjectPermissionId, types.PermissionUpdate) &&
				k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets) {
				validGuilds = append(validGuilds, guild)
			}
		}

		if len(validGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateOwnerId{}), "no updatable guilds"), nil, nil
		}

		allPlayers := k.GetAllPlayer(ctx)
		if len(allPlayers) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateOwnerId{}), "no players available"), nil, nil
		}

		guild := validGuilds[r.Intn(len(validGuilds))]
		newOwner := allPlayers[r.Intn(len(allPlayers))]

		msg := &types.MsgGuildUpdateOwnerId{
			Creator: simAccount.Address.String(),
			GuildId: guild.Id,
			Owner:   newOwner.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.GuildUpdateOwnerId(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildUpdateEntrySubstationId generates a MsgGuildUpdateEntrySubstationId with random values
func SimulateMsgGuildUpdateEntrySubstationId(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateEntrySubstationId{}), "account not found"), nil, nil
		}

		player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, simAccount.Address.String()))
		if !playerFound {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateEntrySubstationId{}), "player not found"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		validGuilds := make([]types.Guild, 0)
		for _, guild := range allGuilds {
			guildObjectPermissionId := keeper.GetObjectPermissionIDBytes(guild.Id, player.Id)
			addressPermissionId := keeper.GetAddressPermissionIDBytes(simAccount.Address.String())
			if k.PermissionHasOneOf(ctx, guildObjectPermissionId, types.PermissionUpdate) &&
				k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets) {
				validGuilds = append(validGuilds, guild)
			}
		}

		if len(validGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateEntrySubstationId{}), "no updatable guilds"), nil, nil
		}

		allSubstations := k.GetAllSubstation(ctx)
		if len(allSubstations) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateEntrySubstationId{}), "no substations available"), nil, nil
		}

		guild := validGuilds[r.Intn(len(validGuilds))]
		substation := allSubstations[r.Intn(len(allSubstations))]

		msg := &types.MsgGuildUpdateEntrySubstationId{
			Creator:           simAccount.Address.String(),
			GuildId:           guild.Id,
			EntrySubstationId: substation.Id,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.GuildUpdateEntrySubstationId(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildUpdateEndpoint generates a MsgGuildUpdateEndpoint with random values
func SimulateMsgGuildUpdateEndpoint(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateEndpoint{}), "account not found"), nil, nil
		}

		player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, simAccount.Address.String()))
		if !playerFound {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateEndpoint{}), "player not found"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		validGuilds := make([]types.Guild, 0)
		for _, guild := range allGuilds {
			guildObjectPermissionId := keeper.GetObjectPermissionIDBytes(guild.Id, player.Id)
			addressPermissionId := keeper.GetAddressPermissionIDBytes(simAccount.Address.String())
			if k.PermissionHasOneOf(ctx, guildObjectPermissionId, types.PermissionUpdate) &&
				k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets) {
				validGuilds = append(validGuilds, guild)
			}
		}

		if len(validGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateEndpoint{}), "no updatable guilds"), nil, nil
		}

		guild := validGuilds[r.Intn(len(validGuilds))]
		endpoint := fmt.Sprintf("https://guild-%d.example.com", r.Int63())

		msg := &types.MsgGuildUpdateEndpoint{
			Creator:  simAccount.Address.String(),
			GuildId:  guild.Id,
			Endpoint: endpoint,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.GuildUpdateEndpoint(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildUpdateJoinInfusionMinimum generates a MsgGuildUpdateJoinInfusionMinimum with random values
func SimulateMsgGuildUpdateJoinInfusionMinimum(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateJoinInfusionMinimum{}), "account not found"), nil, nil
		}

		player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, simAccount.Address.String()))
		if !playerFound {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateJoinInfusionMinimum{}), "player not found"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		validGuilds := make([]types.Guild, 0)
		for _, guild := range allGuilds {
			guildObjectPermissionId := keeper.GetObjectPermissionIDBytes(guild.Id, player.Id)
			addressPermissionId := keeper.GetAddressPermissionIDBytes(simAccount.Address.String())
			if k.PermissionHasOneOf(ctx, guildObjectPermissionId, types.PermissionUpdate) &&
				k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets) {
				validGuilds = append(validGuilds, guild)
			}
		}

		if len(validGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateJoinInfusionMinimum{}), "no updatable guilds"), nil, nil
		}

		guild := validGuilds[r.Intn(len(validGuilds))]
		joinInfusionMinimum := uint64(r.Int63n(1000000) + 1)

		msg := &types.MsgGuildUpdateJoinInfusionMinimum{
			Creator:             simAccount.Address.String(),
			GuildId:             guild.Id,
			JoinInfusionMinimum: joinInfusionMinimum,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.GuildUpdateJoinInfusionMinimum(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildUpdateJoinInfusionMinimumBypassByInvite generates a MsgGuildUpdateJoinInfusionMinimumBypassByInvite with random values
func SimulateMsgGuildUpdateJoinInfusionMinimumBypassByInvite(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateJoinInfusionMinimumBypassByInvite{}), "account not found"), nil, nil
		}

		player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, simAccount.Address.String()))
		if !playerFound {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateJoinInfusionMinimumBypassByInvite{}), "player not found"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		validGuilds := make([]types.Guild, 0)
		for _, guild := range allGuilds {
			guildObjectPermissionId := keeper.GetObjectPermissionIDBytes(guild.Id, player.Id)
			addressPermissionId := keeper.GetAddressPermissionIDBytes(simAccount.Address.String())
			if k.PermissionHasOneOf(ctx, guildObjectPermissionId, types.PermissionUpdate) &&
				k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets) {
				validGuilds = append(validGuilds, guild)
			}
		}

		if len(validGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateJoinInfusionMinimumBypassByInvite{}), "no updatable guilds"), nil, nil
		}

		guild := validGuilds[r.Intn(len(validGuilds))]
		bypassLevels := []types.GuildJoinBypassLevel{
			types.GuildJoinBypassLevel_closed,
			types.GuildJoinBypassLevel_permissioned,
			types.GuildJoinBypassLevel_member,
		}
		bypassLevel := bypassLevels[r.Intn(len(bypassLevels))]

		msg := &types.MsgGuildUpdateJoinInfusionMinimumBypassByInvite{
			Creator:              simAccount.Address.String(),
			GuildId:              guild.Id,
			GuildJoinBypassLevel: bypassLevel,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.GuildUpdateJoinInfusionMinimumBypassByInvite(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgGuildUpdateJoinInfusionMinimumBypassByRequest generates a MsgGuildUpdateJoinInfusionMinimumBypassByRequest with random values
func SimulateMsgGuildUpdateJoinInfusionMinimumBypassByRequest(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateJoinInfusionMinimumBypassByRequest{}), "account not found"), nil, nil
		}

		player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, simAccount.Address.String()))
		if !playerFound {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateJoinInfusionMinimumBypassByRequest{}), "player not found"), nil, nil
		}

		allGuilds := k.GetAllGuild(ctx)
		validGuilds := make([]types.Guild, 0)
		for _, guild := range allGuilds {
			guildObjectPermissionId := keeper.GetObjectPermissionIDBytes(guild.Id, player.Id)
			addressPermissionId := keeper.GetAddressPermissionIDBytes(simAccount.Address.String())
			if k.PermissionHasOneOf(ctx, guildObjectPermissionId, types.PermissionUpdate) &&
				k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets) {
				validGuilds = append(validGuilds, guild)
			}
		}

		if len(validGuilds) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgGuildUpdateJoinInfusionMinimumBypassByRequest{}), "no updatable guilds"), nil, nil
		}

		guild := validGuilds[r.Intn(len(validGuilds))]
		bypassLevels := []types.GuildJoinBypassLevel{
			types.GuildJoinBypassLevel_closed,
			types.GuildJoinBypassLevel_permissioned,
			types.GuildJoinBypassLevel_member,
		}
		bypassLevel := bypassLevels[r.Intn(len(bypassLevels))]

		msg := &types.MsgGuildUpdateJoinInfusionMinimumBypassByRequest{
			Creator:              simAccount.Address.String(),
			GuildId:              guild.Id,
			GuildJoinBypassLevel: bypassLevel,
		}

		msgServer := keeper.NewMsgServerImpl(k)
		_, err := msgServer.GuildUpdateJoinInfusionMinimumBypassByRequest(sdk.WrapSDKContext(ctx), msg)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}
