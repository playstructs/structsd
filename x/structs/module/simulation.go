package structs

import (
	"fmt"
	"math/rand"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

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

	// Ensure we have enough accounts
	if len(simState.Accounts) < 23 {
		// Not enough accounts, use default genesis
		structsGenesis := types.GenesisState{
			Params: types.DefaultParams(),
			PortId: types.PortID,
		}
		simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&structsGenesis)
		return
	}

	// Create 3 reactors
	reactors := make([]types.Reactor, 3)
	for i := 0; i < 3; i++ {
		reactorOwner := simState.Accounts[i]
		reactor := types.CreateEmptyReactor()
		reactor.Id = fmt.Sprintf("%d-%d", types.ObjectType_reactor, uint64(i))
		reactor.Validator = reactorOwner.Address.String()
		reactor.RawAddress = reactorOwner.Address.Bytes()
		reactor.DefaultCommission = math.LegacyZeroDec()
		reactors[i] = reactor
	}

	// Create 3 allocations (one per reactor)
	allocations := make([]types.Allocation, 3)
	for i := 0; i < 3; i++ {
		allocation := types.Allocation{
			Id:             fmt.Sprintf("%d-%d", types.ObjectType_allocation, uint64(i)),
			Type:           types.AllocationType_static,
			SourceObjectId: reactors[i].Id,
			Index:          uint64(i),
			Creator:        simState.Accounts[i].Address.String(),
			Controller:     simState.Accounts[i].Address.String(),
		}
		allocations[i] = allocation
	}

	// Create 3 substations (one per allocation)
	substations := make([]types.Substation, 3)
	for i := 0; i < 3; i++ {
		substation := types.Substation{
			Id:      fmt.Sprintf("%d-%d", types.ObjectType_substation, uint64(i)),
			Owner:   "", // Will be set to player ID
			Creator: simState.Accounts[i].Address.String(),
		}
		substations[i] = substation
		// Update allocation destination
		allocations[i].DestinationId = substation.Id
	}

	// Create 3 guilds (one per substation/reactor pair)
	guilds := make([]types.Guild, 3)
	for i := 0; i < 3; i++ {
		guild := types.Guild{
			Id:                  fmt.Sprintf("%d-%d", types.ObjectType_guild, uint64(i)),
			Index:               uint64(i),
			Endpoint:            fmt.Sprintf("guild-%d-endpoint", i),
			Creator:             simState.Accounts[i].Address.String(),
			Owner:               "", // Will be set to player ID
			JoinInfusionMinimum: 0,
			PrimaryReactorId:    reactors[i].Id,
			EntrySubstationId:   substations[i].Id,
		}
		guilds[i] = guild
		// Link reactor to guild
		reactors[i].GuildId = guild.Id
	}

	// Create 20 players, distributing them across the 3 guilds
	players := make([]types.Player, 20)
	for i := 0; i < 20; i++ {
		accountIdx := 3 + i // Start from account 3 (0-2 are used for reactors)
		if accountIdx >= len(simState.Accounts) {
			accountIdx = i % len(simState.Accounts)
		}
		account := simState.Accounts[accountIdx]

		// Distribute players across guilds (0-6 in guild 0, 7-13 in guild 1, 14-19 in guild 2)
		guildIdx := i / 7
		if guildIdx >= 3 {
			guildIdx = 2
		}
		substationIdx := guildIdx

		player := types.Player{
			Id:             fmt.Sprintf("%d-%d", types.ObjectType_player, uint64(i)),
			Index:          uint64(i),
			GuildId:        guilds[guildIdx].Id,
			SubstationId:   substations[substationIdx].Id,
			Creator:        account.Address.String(),
			PrimaryAddress: account.Address.String(),
		}
		players[i] = player

		// Set guild owner to first player in each guild
		if i%7 == 0 {
			guilds[guildIdx].Owner = player.Id
		}
		// Set substation owner to first player in each substation
		if i%7 == 0 {
			substations[substationIdx].Owner = player.Id
		}
	}

	structsGenesis := types.GenesisState{
		Params:          types.DefaultParams(),
		PortId:          types.PortID,
		ReactorList:     reactors,
		ReactorCount:    3,
		AllocationList:  allocations,
		SubstationList:  substations,
		SubstationCount: 3,
		GuildList:       guilds,
		GuildCount:      3,
		PlayerList:      players,
		PlayerCount:     20,
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&structsGenesis)

	// Set initial balances for all player accounts (1000000ualpha each)
	// Get or create bank genesis state
	var bankGenesis *banktypes.GenesisState
	if bankGenStateBytes, ok := simState.GenState[banktypes.ModuleName]; ok {
		bankGenesis = &banktypes.GenesisState{}
		simState.Cdc.MustUnmarshalJSON(bankGenStateBytes, bankGenesis)
	} else {
		bankGenesis = banktypes.DefaultGenesisState()
	}

	// Add 1000000ualpha to each player account
	playerBalance := sdk.NewCoin("ualpha", math.NewIntFromUint64(1000000))

	for i := 0; i < 20; i++ {
		accountIdx := 3 + i // Start from account 3 (0-2 are used for reactors)
		if accountIdx >= len(simState.Accounts) {
			accountIdx = i % len(simState.Accounts)
		}
		account := simState.Accounts[accountIdx]
		accountAddr := account.Address.String()

		// Check if balance already exists for this account
		found := false
		for j, balance := range bankGenesis.Balances {
			if balance.Address == accountAddr {
				// Add to existing balance
				bankGenesis.Balances[j].Coins = balance.Coins.Add(playerBalance)
				found = true
				break
			}
		}

		// If balance doesn't exist, add new balance entry
		if !found {
			bankGenesis.Balances = append(bankGenesis.Balances, banktypes.Balance{
				Address: accountAddr,
				Coins:   sdk.NewCoins(playerBalance),
			})
		}
	}

	// Add 10,000 new accounts with 10,000 stake each, all delegated to the first validator
	// Get staking genesis to find the first validator
	var stakingGenesis *stakingtypes.GenesisState
	if stakingGenStateBytes, ok := simState.GenState[stakingtypes.ModuleName]; ok {
		stakingGenesis = &stakingtypes.GenesisState{}
		simState.Cdc.MustUnmarshalJSON(stakingGenStateBytes, stakingGenesis)
	} else {
		stakingGenesis = stakingtypes.DefaultGenesisState()
	}

	// Find the first validator (or use the first one if multiple exist)
	if len(stakingGenesis.Validators) == 0 {
		// No validators exist, skip delegation setup
		simState.GenState[banktypes.ModuleName] = simState.Cdc.MustMarshalJSON(bankGenesis)
		return
	}
	firstValidatorIdx := 0
	firstValidatorAddr := stakingGenesis.Validators[0].OperatorAddress

	// Get bond denom (this is what validators use for staking)
	bondDenom := "ualpha"
	if stakingGenesis.Params.BondDenom != "" {
		bondDenom = stakingGenesis.Params.BondDenom
	}

	delegationAmount := math.NewIntFromUint64(10000) // 10,000 tokens per account (in bond denom)

	// Generate 10,000 accounts and create delegations
	numAccounts := 10000
	totalDelegationAmount := math.NewIntFromUint64(100000000) // 100 million tokens total (10,000 * 10,000)
	for i := 0; i < numAccounts; i++ {
		// Generate a new account with a random private key
		privKey := secp256k1.GenPrivKey()
		pubKey := privKey.PubKey()
		addr := sdk.AccAddress(pubKey.Address())
		addrStr := addr.String()

		// Add balance to the account in the bond denom (stake, not ualpha)
		// Accounts need bond denom tokens to delegate, not ualpha
		accountBalance := sdk.NewCoin(bondDenom, delegationAmount)
		balanceFound := false
		for j, balance := range bankGenesis.Balances {
			if balance.Address == addrStr {
				bankGenesis.Balances[j].Coins = balance.Coins.Add(accountBalance)
				balanceFound = true
				break
			}
		}
		if !balanceFound {
			bankGenesis.Balances = append(bankGenesis.Balances, banktypes.Balance{
				Address: addrStr,
				Coins:   sdk.NewCoins(accountBalance),
			})
		}

		// Create delegation to the first validator
		delegation := stakingtypes.Delegation{
			DelegatorAddress: addrStr,
			ValidatorAddress: firstValidatorAddr,
			Shares:           math.LegacyNewDecFromInt(delegationAmount),
		}
		stakingGenesis.Delegations = append(stakingGenesis.Delegations, delegation)
	}

	// Update the validator's DelegatorShares and Tokens to include all new delegations
	// The validator's DelegatorShares must equal the sum of all delegator shares
	totalNewShares := math.LegacyNewDecFromInt(totalDelegationAmount)
	if firstValidatorIdx >= 0 {
		stakingGenesis.Validators[firstValidatorIdx].DelegatorShares = stakingGenesis.Validators[firstValidatorIdx].DelegatorShares.Add(totalNewShares)
		stakingGenesis.Validators[firstValidatorIdx].Tokens = stakingGenesis.Validators[firstValidatorIdx].Tokens.Add(totalDelegationAmount)
	}

	// CRITICAL: Clear supply and let the bank module calculate it automatically from balances.
	// The bank module's InitGenesis will calculate supply from all balances if supply is empty.
	// This ensures supply exactly matches the sum of all balances without any manual calculation errors.
	bankGenesis.Supply = sdk.Coins{}

	// Update bank genesis state
	simState.GenState[banktypes.ModuleName] = simState.Cdc.MustMarshalJSON(bankGenesis)

	// Update staking genesis state with new delegations
	simState.GenState[stakingtypes.ModuleName] = simState.Cdc.MustMarshalJSON(stakingGenesis)
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
