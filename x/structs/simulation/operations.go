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
