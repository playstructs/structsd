package structs

import (
	"fmt"
	"math/rand"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
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
	OpWeightMsgAllocationCreate           = "op_weight_msg_allocation_create"
	OpWeightMsgSubstationCreate           = "op_weight_msg_substation_create"
	OpWeightMsgProviderCreate             = "op_weight_msg_provider_create"
	OpWeightMsgAgreementOpen              = "op_weight_msg_agreement_open"
	// Agreement operations
	OpWeightMsgAgreementClose            = "op_weight_msg_agreement_close"
	OpWeightMsgAgreementCapacityIncrease = "op_weight_msg_agreement_capacity_increase"
	OpWeightMsgAgreementCapacityDecrease = "op_weight_msg_agreement_capacity_decrease"
	OpWeightMsgAgreementDurationIncrease = "op_weight_msg_agreement_duration_increase"
	// Allocation operations
	OpWeightMsgAllocationDelete   = "op_weight_msg_allocation_delete"
	OpWeightMsgAllocationUpdate   = "op_weight_msg_allocation_update"
	OpWeightMsgAllocationTransfer = "op_weight_msg_allocation_transfer"
	// Fleet operations
	OpWeightMsgFleetMove = "op_weight_msg_fleet_move"
	// Struct operations
	OpWeightMsgStructBuildComplete = "op_weight_msg_struct_build_complete"
	OpWeightMsgStructBuildCancel   = "op_weight_msg_struct_build_cancel"
	OpWeightMsgStructActivate      = "op_weight_msg_struct_activate"
	OpWeightMsgStructDeactivate    = "op_weight_msg_struct_deactivate"
	// Provider operations
	OpWeightMsgProviderWithdrawBalance    = "op_weight_msg_provider_withdraw_balance"
	OpWeightMsgProviderUpdateCapacityMin  = "op_weight_msg_provider_update_capacity_min"
	OpWeightMsgProviderUpdateCapacityMax  = "op_weight_msg_provider_update_capacity_max"
	OpWeightMsgProviderUpdateDurationMin  = "op_weight_msg_provider_update_duration_min"
	OpWeightMsgProviderUpdateDurationMax  = "op_weight_msg_provider_update_duration_max"
	OpWeightMsgProviderUpdateAccessPolicy = "op_weight_msg_provider_update_access_policy"
	OpWeightMsgProviderGuildGrant         = "op_weight_msg_provider_guild_grant"
	OpWeightMsgProviderGuildRevoke        = "op_weight_msg_provider_guild_revoke"
	OpWeightMsgProviderDelete             = "op_weight_msg_provider_delete"
	// Substation operations
	OpWeightMsgSubstationAllocationConnect    = "op_weight_msg_substation_allocation_connect"
	OpWeightMsgSubstationAllocationDisconnect = "op_weight_msg_substation_allocation_disconnect"
	OpWeightMsgSubstationPlayerConnect        = "op_weight_msg_substation_player_connect"
	OpWeightMsgSubstationPlayerDisconnect     = "op_weight_msg_substation_player_disconnect"
	OpWeightMsgSubstationPlayerMigrate        = "op_weight_msg_substation_player_migrate"
	OpWeightMsgSubstationDelete               = "op_weight_msg_substation_delete"
	// Address operations
	OpWeightMsgAddressRevoke = "op_weight_msg_address_revoke"
	// Player operations
	OpWeightMsgPlayerUpdatePrimaryAddress = "op_weight_msg_player_update_primary_address"
	OpWeightMsgPlayerResume               = "op_weight_msg_player_resume"
	// Planet operations
	OpWeightMsgPlanetRaidComplete = "op_weight_msg_planet_raid_complete"
	// Reactor operations
	OpWeightMsgReactorDefuse         = "op_weight_msg_reactor_defuse"
	OpWeightMsgReactorBeginMigration = "op_weight_msg_reactor_begin_migration"
	OpWeightMsgReactorCancelDefusion = "op_weight_msg_reactor_cancel_defusion"
	// Guild membership operations
	OpWeightMsgGuildMembershipInvite         = "op_weight_msg_guild_membership_invite"
	OpWeightMsgGuildMembershipInviteApprove  = "op_weight_msg_guild_membership_invite_approve"
	OpWeightMsgGuildMembershipInviteDeny     = "op_weight_msg_guild_membership_invite_deny"
	OpWeightMsgGuildMembershipInviteRevoke   = "op_weight_msg_guild_membership_invite_revoke"
	OpWeightMsgGuildMembershipJoinProxy      = "op_weight_msg_guild_membership_join_proxy"
	OpWeightMsgGuildMembershipKick           = "op_weight_msg_guild_membership_kick"
	OpWeightMsgGuildMembershipRequestApprove = "op_weight_msg_guild_membership_request_approve"
	OpWeightMsgGuildMembershipRequestDeny    = "op_weight_msg_guild_membership_request_deny"
	OpWeightMsgGuildMembershipRequestRevoke  = "op_weight_msg_guild_membership_request_revoke"
	// Guild update operations
	OpWeightMsgGuildUpdateOwnerId                   = "op_weight_msg_guild_update_owner_id"
	OpWeightMsgGuildUpdateEntrySubstationId         = "op_weight_msg_guild_update_entry_substation_id"
	OpWeightMsgGuildUpdateEndpoint                  = "op_weight_msg_guild_update_endpoint"
	OpWeightMsgGuildUpdateJoinInfusionMin           = "op_weight_msg_guild_update_join_infusion_min"
	OpWeightMsgGuildUpdateJoinInfusionBypassInvite  = "op_weight_msg_guild_update_join_infusion_bypass_invite"
	OpWeightMsgGuildUpdateJoinInfusionBypassRequest = "op_weight_msg_guild_update_join_infusion_bypass_request"
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
	firstValidatorAddr := stakingGenesis.Validators[0].OperatorAddress

	// Get bond denom (this is what validators use for staking)
	bondDenom := "ualpha"
	if stakingGenesis.Params.BondDenom != "" {
		bondDenom = stakingGenesis.Params.BondDenom
	}

	delegationAmount := math.NewIntFromUint64(10000) // 10,000 tokens per account (in bond denom)

	// Generate 10,000 accounts and create delegations
	numAccounts := 10000
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

	// NOTE: We don't update validator shares here incrementally.
	// Instead, we'll recalculate all validator shares from the sum of all delegations at the end
	// to ensure the invariant: validator.DelegatorShares == sum of all delegation.Shares

	// CRITICAL: Ensure all validators stay bonded by:
	// 1. Adding large self-delegations that are unlikely to be fully undelegated
	// 2. Ensuring validators are in Bonded status
	// 3. Increasing unbonding time to prevent quick unbonding
	// 4. Ensuring validator operator addresses have sufficient balance
	for i := range stakingGenesis.Validators {
		validator := &stakingGenesis.Validators[i]

		// Ensure validator is in Bonded status
		validator.Status = stakingtypes.Bonded

		// Add a large self-delegation (100 million tokens) that won't be easily undelegated
		// This ensures the validator always has enough stake to stay bonded
		selfDelegationAmount := math.NewIntFromUint64(100000000) // 100 million
		selfDelegationShares := math.LegacyNewDecFromInt(selfDelegationAmount)

		// Ensure validator operator address has sufficient balance for self-delegation
		// Convert validator operator address (structsvaloper...) to account address (structs...)
		valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
		if err != nil {
			continue // Skip this validator if address conversion fails
		}
		// Convert ValAddress to AccAddress (they share the same bytes, just different bech32 prefixes)
		operatorAccAddr := sdk.AccAddress(valAddr.Bytes())
		operatorAddr := operatorAccAddr.String() // This will have the correct 'structs' prefix

		operatorBalance := sdk.NewCoin(bondDenom, selfDelegationAmount)
		operatorBalanceFound := false
		for j, balance := range bankGenesis.Balances {
			if balance.Address == operatorAddr {
				// Add to existing balance if needed
				hasEnough := false
				for _, coin := range balance.Coins {
					if coin.Denom == bondDenom && coin.Amount.GTE(selfDelegationAmount) {
						hasEnough = true
						break
					}
				}
				if !hasEnough {
					bankGenesis.Balances[j].Coins = balance.Coins.Add(operatorBalance)
				}
				operatorBalanceFound = true
				break
			}
		}
		if !operatorBalanceFound {
			bankGenesis.Balances = append(bankGenesis.Balances, banktypes.Balance{
				Address: operatorAddr,
				Coins:   sdk.NewCoins(operatorBalance),
			})
		}

		// Check if self-delegation already exists
		// Note: DelegatorAddress must be an account address (structs...), not validator address (structsvaloper...)
		selfDelegationFound := false
		for j, delegation := range stakingGenesis.Delegations {
			if delegation.DelegatorAddress == operatorAddr && delegation.ValidatorAddress == validator.OperatorAddress {
				// Update existing self-delegation to be larger
				stakingGenesis.Delegations[j].Shares = selfDelegationShares
				selfDelegationFound = true
				break
			}
		}

		// If no self-delegation exists, add one
		// DelegatorAddress must be account address, ValidatorAddress is validator address
		if !selfDelegationFound {
			selfDelegation := stakingtypes.Delegation{
				DelegatorAddress: operatorAddr,              // Account address (structs...)
				ValidatorAddress: validator.OperatorAddress, // Validator address (structsvaloper...)
				Shares:           selfDelegationShares,
			}
			stakingGenesis.Delegations = append(stakingGenesis.Delegations, selfDelegation)
		}
	}

	// CRITICAL: Recalculate validator DelegatorShares and Tokens from the sum of all delegations
	// The invariant requires: validator.DelegatorShares == sum of all delegation.Shares for that validator
	// We must recalculate this after adding all delegations, not incrementally add to existing shares
	minSelfDelegationTokens := math.NewIntFromUint64(1000000000) // 1 billion tokens minimum
	minSelfDelegationShares := math.LegacyNewDecFromInt(minSelfDelegationTokens)

	for i := range stakingGenesis.Validators {
		validator := &stakingGenesis.Validators[i]

		// CRITICAL: First, ensure the self-delegation exists and is at least the minimum
		// This must be done BEFORE calculating the sum, so the sum includes the correct self-delegation
		valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
		if err == nil {
			operatorAccAddr := sdk.AccAddress(valAddr.Bytes())
			operatorAddr := operatorAccAddr.String()

			// Find and update self-delegation to ensure it's at least the minimum
			selfDelegationFound := false
			for j, delegation := range stakingGenesis.Delegations {
				if delegation.DelegatorAddress == operatorAddr && delegation.ValidatorAddress == validator.OperatorAddress {
					// Update self-delegation to be at least the minimum
					if delegation.Shares.LT(minSelfDelegationShares) {
						stakingGenesis.Delegations[j].Shares = minSelfDelegationShares
					}
					selfDelegationFound = true
					break
				}
			}

			// If no self-delegation exists, add one
			if !selfDelegationFound {
				selfDelegation := stakingtypes.Delegation{
					DelegatorAddress: operatorAddr,
					ValidatorAddress: validator.OperatorAddress,
					Shares:           minSelfDelegationShares,
				}
				stakingGenesis.Delegations = append(stakingGenesis.Delegations, selfDelegation)
			}
		}

		// Now sum all delegation shares for this validator (including the updated/added self-delegation)
		totalShares := math.LegacyZeroDec()
		for _, delegation := range stakingGenesis.Delegations {
			if delegation.ValidatorAddress == validator.OperatorAddress {
				totalShares = totalShares.Add(delegation.Shares)
			}
		}

		// Set validator shares to match the sum of all delegations
		// For tokens, we use the shares value (assuming 1:1 ratio for new validators in genesis)
		// This is correct because we created delegations with shares = tokens (1:1 ratio)
		validator.DelegatorShares = totalShares
		validator.Tokens = totalShares.TruncateInt()

		// Ensure validator has at least minimum tokens (should already be satisfied by self-delegation above)
		// This is a safety check in case something went wrong
		if validator.Tokens.LT(minSelfDelegationTokens) {
			validator.Tokens = minSelfDelegationTokens
			validator.DelegatorShares = minSelfDelegationShares
		}
	}

	// Increase unbonding time to 10 years (in seconds) to prevent validators from unbonding quickly
	// This gives validators plenty of time before they can fully unbond
	if stakingGenesis.Params.UnbondingTime < 10*365*24*60*60 {
		stakingGenesis.Params.UnbondingTime = 10 * 365 * 24 * 60 * 60 // 10 years in seconds
	}

	// CRITICAL: Update bonded pool balance to match total bonded tokens
	// The staking module requires the bonded pool balance to equal the sum of all validator tokens
	// for validators in Bonded status. Calculate total bonded tokens and update the pool.
	totalBondedTokens := math.ZeroInt()
	for _, validator := range stakingGenesis.Validators {
		if validator.Status == stakingtypes.Bonded {
			totalBondedTokens = totalBondedTokens.Add(validator.Tokens)
		}
	}

	// Update bonded pool balance in bank genesis
	// Get the actual module account address from the module name
	bondedPoolAddr := authtypes.NewModuleAddress(stakingtypes.BondedPoolName)
	bondedPoolBalance := sdk.NewCoin(bondDenom, totalBondedTokens)
	bondedPoolFound := false
	for j, balance := range bankGenesis.Balances {
		if balance.Address == bondedPoolAddr.String() {
			// Update existing bonded pool balance
			// Replace the balance for the bond denom
			updatedCoins := sdk.Coins{}
			for _, coin := range balance.Coins {
				if coin.Denom != bondDenom {
					updatedCoins = updatedCoins.Add(coin)
				}
			}
			updatedCoins = updatedCoins.Add(bondedPoolBalance)
			bankGenesis.Balances[j].Coins = updatedCoins
			bondedPoolFound = true
			break
		}
	}
	if !bondedPoolFound {
		// Add bonded pool balance if it doesn't exist
		bankGenesis.Balances = append(bankGenesis.Balances, banktypes.Balance{
			Address: bondedPoolAddr.String(),
			Coins:   sdk.NewCoins(bondedPoolBalance),
		})
	}

	// CRITICAL: Clear supply and let the bank module calculate it automatically from balances.
	// The bank module's InitGenesis will calculate supply from all balances if supply is empty.
	// This ensures supply exactly matches the sum of all balances without any manual calculation errors.
	bankGenesis.Supply = sdk.Coins{}

	// Update bank genesis state
	simState.GenState[banktypes.ModuleName] = simState.Cdc.MustMarshalJSON(bankGenesis)

	// Update staking genesis state with new delegations
	simState.GenState[stakingtypes.ModuleName] = simState.Cdc.MustMarshalJSON(stakingGenesis)

	// Initialize distribution module genesis state with validator distribution info
	// This is required when delegations exist - the distribution module needs validator distribution info
	var distrGenesis *distrtypes.GenesisState
	if distrGenStateBytes, ok := simState.GenState[distrtypes.ModuleName]; ok {
		distrGenesis = &distrtypes.GenesisState{}
		simState.Cdc.MustUnmarshalJSON(distrGenStateBytes, distrGenesis)
	} else {
		distrGenesis = distrtypes.DefaultGenesisState()
	}

	// Initialize validator distribution info for ALL validators
	// This prevents "no delegation distribution info" errors during simulation
	// Simulation operations may create delegations to any validator, so all validators need distribution info
	for _, validator := range stakingGenesis.Validators {
		valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
		if err != nil {
			continue // Skip if address conversion fails
		}
		valAddrStr := valAddr.String()

		// Check if validator current rewards already exists
		found := false
		for _, valRewards := range distrGenesis.ValidatorCurrentRewards {
			if valRewards.ValidatorAddress == valAddrStr {
				found = true
				break
			}
		}

		// If not found, add validator current rewards (required for validators with delegations)
		if !found {
			distrGenesis.ValidatorCurrentRewards = append(distrGenesis.ValidatorCurrentRewards, distrtypes.ValidatorCurrentRewardsRecord{
				ValidatorAddress: valAddrStr,
				Rewards: distrtypes.ValidatorCurrentRewards{
					Rewards: sdk.DecCoins{},
					Period:  0,
				},
			})
		}

		// Initialize validator historical rewards (required for reward calculations)
		foundHistorical := false
		for _, valHistorical := range distrGenesis.ValidatorHistoricalRewards {
			if valHistorical.ValidatorAddress == valAddrStr {
				foundHistorical = true
				break
			}
		}
		if !foundHistorical {
			distrGenesis.ValidatorHistoricalRewards = append(distrGenesis.ValidatorHistoricalRewards, distrtypes.ValidatorHistoricalRewardsRecord{
				ValidatorAddress: valAddrStr,
				Period:           0,
				Rewards: distrtypes.ValidatorHistoricalRewards{
					CumulativeRewardRatio: sdk.DecCoins{},
					ReferenceCount:        0,
				},
			})
		}

		// Initialize validator accumulated commissions
		foundCommission := false
		for _, valComm := range distrGenesis.ValidatorAccumulatedCommissions {
			if valComm.ValidatorAddress == valAddrStr {
				foundCommission = true
				break
			}
		}
		if !foundCommission {
			distrGenesis.ValidatorAccumulatedCommissions = append(distrGenesis.ValidatorAccumulatedCommissions, distrtypes.ValidatorAccumulatedCommissionRecord{
				ValidatorAddress: valAddrStr,
				Accumulated: distrtypes.ValidatorAccumulatedCommission{
					Commission: sdk.DecCoins{},
				},
			})
		}

		// Initialize outstanding rewards
		foundOutstanding := false
		for _, valOutstanding := range distrGenesis.OutstandingRewards {
			if valOutstanding.ValidatorAddress == valAddrStr {
				foundOutstanding = true
				break
			}
		}
		if !foundOutstanding {
			distrGenesis.OutstandingRewards = append(distrGenesis.OutstandingRewards, distrtypes.ValidatorOutstandingRewardsRecord{
				ValidatorAddress:   valAddrStr,
				OutstandingRewards: sdk.DecCoins{},
			})
		}
	}

	// Initialize DelegatorStartingInfos for all delegations in genesis
	// This is required for the distribution module to track delegator rewards correctly
	for _, delegation := range stakingGenesis.Delegations {
		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			continue
		}
		delAddr, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
		if err != nil {
			continue
		}

		// Create a unique key for this delegator-validator pair
		// The distribution module uses this to track starting info
		startingInfoKey := fmt.Sprintf("%s_%s", delAddr.String(), valAddr.String())

		// Check if starting info already exists
		found := false
		for _, startingInfo := range distrGenesis.DelegatorStartingInfos {
			if startingInfo.DelegatorAddress == delAddr.String() && startingInfo.ValidatorAddress == valAddr.String() {
				found = true
				break
			}
		}

		// If not found, add delegator starting info
		if !found {
			distrGenesis.DelegatorStartingInfos = append(distrGenesis.DelegatorStartingInfos, distrtypes.DelegatorStartingInfoRecord{
				DelegatorAddress: delAddr.String(),
				ValidatorAddress: valAddr.String(),
				StartingInfo: distrtypes.DelegatorStartingInfo{
					PreviousPeriod: 0,
					Stake:          delegation.Shares,
					Height:         0,
				},
			})
		}
		_ = startingInfoKey // Avoid unused variable warning
	}

	// Update distribution genesis state
	simState.GenState[distrtypes.ModuleName] = simState.Cdc.MustMarshalJSON(distrGenesis)

	// Initialize slashing module genesis state with validator signing info
	// This is required for all validators - the slashing module needs signing info for each validator
	var slashingGenesis *slashingtypes.GenesisState
	if slashingGenStateBytes, ok := simState.GenState[slashingtypes.ModuleName]; ok {
		slashingGenesis = &slashingtypes.GenesisState{}
		simState.Cdc.MustUnmarshalJSON(slashingGenStateBytes, slashingGenesis)
	} else {
		slashingGenesis = slashingtypes.DefaultGenesisState()
	}

	// Add signing info for all validators
	for _, validator := range stakingGenesis.Validators {
		// Get consensus pubkey from validator using the ConsPubKey() method
		consPubKey, err := validator.ConsPubKey()
		if err != nil {
			continue // Skip if we can't get the consensus pubkey
		}

		// Get consensus address from pubkey
		consAddr := sdk.ConsAddress(consPubKey.Address())
		consAddrStr := consAddr.String()

		// Check if signing info already exists for this validator
		signingInfoFound := false
		for j, signingInfo := range slashingGenesis.SigningInfos {
			if signingInfo.Address == consAddrStr {
				// Update existing signing info if needed
				signingInfoFound = true
				// Reset start height to 0 for genesis
				slashingGenesis.SigningInfos[j].ValidatorSigningInfo.StartHeight = 0
				slashingGenesis.SigningInfos[j].ValidatorSigningInfo.IndexOffset = 0
				slashingGenesis.SigningInfos[j].ValidatorSigningInfo.MissedBlocksCounter = 0
				break
			}
		}

		// If no signing info exists, add one
		if !signingInfoFound {
			signingInfo := slashingtypes.SigningInfo{
				Address: consAddrStr,
				ValidatorSigningInfo: slashingtypes.ValidatorSigningInfo{
					Address:             consAddrStr,
					StartHeight:         0,
					IndexOffset:         0,
					JailedUntil:         time.Time{},
					Tombstoned:          false,
					MissedBlocksCounter: 0,
				},
			}
			slashingGenesis.SigningInfos = append(slashingGenesis.SigningInfos, signingInfo)
		}
	}

	// Update slashing genesis state
	simState.GenState[slashingtypes.ModuleName] = simState.Cdc.MustMarshalJSON(slashingGenesis)
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

	var weightMsgAllocationCreate int
	simState.AppParams.GetOrGenerate(OpWeightMsgAllocationCreate, &weightMsgAllocationCreate, nil,
		func(_ *rand.Rand) {
			weightMsgAllocationCreate = 30
		},
	)

	var weightMsgSubstationCreate int
	simState.AppParams.GetOrGenerate(OpWeightMsgSubstationCreate, &weightMsgSubstationCreate, nil,
		func(_ *rand.Rand) {
			weightMsgSubstationCreate = 25
		},
	)

	var weightMsgProviderCreate int
	simState.AppParams.GetOrGenerate(OpWeightMsgProviderCreate, &weightMsgProviderCreate, nil,
		func(_ *rand.Rand) {
			weightMsgProviderCreate = 20
		},
	)

	var weightMsgAgreementOpen int
	simState.AppParams.GetOrGenerate(OpWeightMsgAgreementOpen, &weightMsgAgreementOpen, nil,
		func(_ *rand.Rand) {
			weightMsgAgreementOpen = 40
		},
	)

	// Agreement operations
	var weightMsgAgreementClose int
	simState.AppParams.GetOrGenerate(OpWeightMsgAgreementClose, &weightMsgAgreementClose, nil,
		func(_ *rand.Rand) { weightMsgAgreementClose = 20 },
	)
	var weightMsgAgreementCapacityIncrease int
	simState.AppParams.GetOrGenerate(OpWeightMsgAgreementCapacityIncrease, &weightMsgAgreementCapacityIncrease, nil,
		func(_ *rand.Rand) { weightMsgAgreementCapacityIncrease = 15 },
	)
	var weightMsgAgreementCapacityDecrease int
	simState.AppParams.GetOrGenerate(OpWeightMsgAgreementCapacityDecrease, &weightMsgAgreementCapacityDecrease, nil,
		func(_ *rand.Rand) { weightMsgAgreementCapacityDecrease = 15 },
	)
	var weightMsgAgreementDurationIncrease int
	simState.AppParams.GetOrGenerate(OpWeightMsgAgreementDurationIncrease, &weightMsgAgreementDurationIncrease, nil,
		func(_ *rand.Rand) { weightMsgAgreementDurationIncrease = 15 },
	)
	// Allocation operations
	var weightMsgAllocationDelete int
	simState.AppParams.GetOrGenerate(OpWeightMsgAllocationDelete, &weightMsgAllocationDelete, nil,
		func(_ *rand.Rand) { weightMsgAllocationDelete = 10 },
	)
	var weightMsgAllocationUpdate int
	simState.AppParams.GetOrGenerate(OpWeightMsgAllocationUpdate, &weightMsgAllocationUpdate, nil,
		func(_ *rand.Rand) { weightMsgAllocationUpdate = 20 },
	)
	var weightMsgAllocationTransfer int
	simState.AppParams.GetOrGenerate(OpWeightMsgAllocationTransfer, &weightMsgAllocationTransfer, nil,
		func(_ *rand.Rand) { weightMsgAllocationTransfer = 10 },
	)
	// Fleet operations
	var weightMsgFleetMove int
	simState.AppParams.GetOrGenerate(OpWeightMsgFleetMove, &weightMsgFleetMove, nil,
		func(_ *rand.Rand) { weightMsgFleetMove = 30 },
	)
	// Struct operations
	var weightMsgStructBuildComplete int
	simState.AppParams.GetOrGenerate(OpWeightMsgStructBuildComplete, &weightMsgStructBuildComplete, nil,
		func(_ *rand.Rand) { weightMsgStructBuildComplete = 15 },
	)
	var weightMsgStructBuildCancel int
	simState.AppParams.GetOrGenerate(OpWeightMsgStructBuildCancel, &weightMsgStructBuildCancel, nil,
		func(_ *rand.Rand) { weightMsgStructBuildCancel = 5 },
	)
	var weightMsgStructActivate int
	simState.AppParams.GetOrGenerate(OpWeightMsgStructActivate, &weightMsgStructActivate, nil,
		func(_ *rand.Rand) { weightMsgStructActivate = 25 },
	)
	var weightMsgStructDeactivate int
	simState.AppParams.GetOrGenerate(OpWeightMsgStructDeactivate, &weightMsgStructDeactivate, nil,
		func(_ *rand.Rand) { weightMsgStructDeactivate = 20 },
	)
	// Provider operations
	var weightMsgProviderWithdrawBalance int
	simState.AppParams.GetOrGenerate(OpWeightMsgProviderWithdrawBalance, &weightMsgProviderWithdrawBalance, nil,
		func(_ *rand.Rand) { weightMsgProviderWithdrawBalance = 10 },
	)
	var weightMsgProviderUpdateCapacityMin int
	simState.AppParams.GetOrGenerate(OpWeightMsgProviderUpdateCapacityMin, &weightMsgProviderUpdateCapacityMin, nil,
		func(_ *rand.Rand) { weightMsgProviderUpdateCapacityMin = 5 },
	)
	var weightMsgProviderUpdateCapacityMax int
	simState.AppParams.GetOrGenerate(OpWeightMsgProviderUpdateCapacityMax, &weightMsgProviderUpdateCapacityMax, nil,
		func(_ *rand.Rand) { weightMsgProviderUpdateCapacityMax = 5 },
	)
	var weightMsgProviderUpdateDurationMin int
	simState.AppParams.GetOrGenerate(OpWeightMsgProviderUpdateDurationMin, &weightMsgProviderUpdateDurationMin, nil,
		func(_ *rand.Rand) { weightMsgProviderUpdateDurationMin = 5 },
	)
	var weightMsgProviderUpdateDurationMax int
	simState.AppParams.GetOrGenerate(OpWeightMsgProviderUpdateDurationMax, &weightMsgProviderUpdateDurationMax, nil,
		func(_ *rand.Rand) { weightMsgProviderUpdateDurationMax = 5 },
	)
	var weightMsgProviderUpdateAccessPolicy int
	simState.AppParams.GetOrGenerate(OpWeightMsgProviderUpdateAccessPolicy, &weightMsgProviderUpdateAccessPolicy, nil,
		func(_ *rand.Rand) { weightMsgProviderUpdateAccessPolicy = 5 },
	)
	var weightMsgProviderGuildGrant int
	simState.AppParams.GetOrGenerate(OpWeightMsgProviderGuildGrant, &weightMsgProviderGuildGrant, nil,
		func(_ *rand.Rand) { weightMsgProviderGuildGrant = 5 },
	)
	var weightMsgProviderGuildRevoke int
	simState.AppParams.GetOrGenerate(OpWeightMsgProviderGuildRevoke, &weightMsgProviderGuildRevoke, nil,
		func(_ *rand.Rand) { weightMsgProviderGuildRevoke = 5 },
	)
	var weightMsgProviderDelete int
	simState.AppParams.GetOrGenerate(OpWeightMsgProviderDelete, &weightMsgProviderDelete, nil,
		func(_ *rand.Rand) { weightMsgProviderDelete = 5 },
	)
	// Substation operations
	var weightMsgSubstationAllocationConnect int
	simState.AppParams.GetOrGenerate(OpWeightMsgSubstationAllocationConnect, &weightMsgSubstationAllocationConnect, nil,
		func(_ *rand.Rand) { weightMsgSubstationAllocationConnect = 20 },
	)
	var weightMsgSubstationAllocationDisconnect int
	simState.AppParams.GetOrGenerate(OpWeightMsgSubstationAllocationDisconnect, &weightMsgSubstationAllocationDisconnect, nil,
		func(_ *rand.Rand) { weightMsgSubstationAllocationDisconnect = 15 },
	)
	var weightMsgSubstationPlayerConnect int
	simState.AppParams.GetOrGenerate(OpWeightMsgSubstationPlayerConnect, &weightMsgSubstationPlayerConnect, nil,
		func(_ *rand.Rand) { weightMsgSubstationPlayerConnect = 20 },
	)
	var weightMsgSubstationPlayerDisconnect int
	simState.AppParams.GetOrGenerate(OpWeightMsgSubstationPlayerDisconnect, &weightMsgSubstationPlayerDisconnect, nil,
		func(_ *rand.Rand) { weightMsgSubstationPlayerDisconnect = 15 },
	)
	var weightMsgSubstationPlayerMigrate int
	simState.AppParams.GetOrGenerate(OpWeightMsgSubstationPlayerMigrate, &weightMsgSubstationPlayerMigrate, nil,
		func(_ *rand.Rand) { weightMsgSubstationPlayerMigrate = 10 },
	)
	var weightMsgSubstationDelete int
	simState.AppParams.GetOrGenerate(OpWeightMsgSubstationDelete, &weightMsgSubstationDelete, nil,
		func(_ *rand.Rand) { weightMsgSubstationDelete = 5 },
	)
	// Address operations
	var weightMsgAddressRevoke int
	simState.AppParams.GetOrGenerate(OpWeightMsgAddressRevoke, &weightMsgAddressRevoke, nil,
		func(_ *rand.Rand) { weightMsgAddressRevoke = 5 },
	)
	// Player operations
	var weightMsgPlayerUpdatePrimaryAddress int
	simState.AppParams.GetOrGenerate(OpWeightMsgPlayerUpdatePrimaryAddress, &weightMsgPlayerUpdatePrimaryAddress, nil,
		func(_ *rand.Rand) { weightMsgPlayerUpdatePrimaryAddress = 5 },
	)
	var weightMsgPlayerResume int
	simState.AppParams.GetOrGenerate(OpWeightMsgPlayerResume, &weightMsgPlayerResume, nil,
		func(_ *rand.Rand) { weightMsgPlayerResume = 10 },
	)
	// Planet operations
	var weightMsgPlanetRaidComplete int
	simState.AppParams.GetOrGenerate(OpWeightMsgPlanetRaidComplete, &weightMsgPlanetRaidComplete, nil,
		func(_ *rand.Rand) { weightMsgPlanetRaidComplete = 10 },
	)
	// Reactor operations
	var weightMsgReactorDefuse int
	simState.AppParams.GetOrGenerate(OpWeightMsgReactorDefuse, &weightMsgReactorDefuse, nil,
		func(_ *rand.Rand) { weightMsgReactorDefuse = 15 },
	)
	var weightMsgReactorBeginMigration int
	simState.AppParams.GetOrGenerate(OpWeightMsgReactorBeginMigration, &weightMsgReactorBeginMigration, nil,
		func(_ *rand.Rand) { weightMsgReactorBeginMigration = 10 },
	)
	var weightMsgReactorCancelDefusion int
	simState.AppParams.GetOrGenerate(OpWeightMsgReactorCancelDefusion, &weightMsgReactorCancelDefusion, nil,
		func(_ *rand.Rand) { weightMsgReactorCancelDefusion = 5 },
	)
	// Guild membership operations
	var weightMsgGuildMembershipInvite int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildMembershipInvite, &weightMsgGuildMembershipInvite, nil,
		func(_ *rand.Rand) { weightMsgGuildMembershipInvite = 15 },
	)
	var weightMsgGuildMembershipInviteApprove int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildMembershipInviteApprove, &weightMsgGuildMembershipInviteApprove, nil,
		func(_ *rand.Rand) { weightMsgGuildMembershipInviteApprove = 10 },
	)
	var weightMsgGuildMembershipInviteDeny int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildMembershipInviteDeny, &weightMsgGuildMembershipInviteDeny, nil,
		func(_ *rand.Rand) { weightMsgGuildMembershipInviteDeny = 5 },
	)
	var weightMsgGuildMembershipInviteRevoke int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildMembershipInviteRevoke, &weightMsgGuildMembershipInviteRevoke, nil,
		func(_ *rand.Rand) { weightMsgGuildMembershipInviteRevoke = 5 },
	)
	var weightMsgGuildMembershipJoinProxy int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildMembershipJoinProxy, &weightMsgGuildMembershipJoinProxy, nil,
		func(_ *rand.Rand) { weightMsgGuildMembershipJoinProxy = 5 },
	)
	var weightMsgGuildMembershipKick int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildMembershipKick, &weightMsgGuildMembershipKick, nil,
		func(_ *rand.Rand) { weightMsgGuildMembershipKick = 5 },
	)
	var weightMsgGuildMembershipRequestApprove int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildMembershipRequestApprove, &weightMsgGuildMembershipRequestApprove, nil,
		func(_ *rand.Rand) { weightMsgGuildMembershipRequestApprove = 10 },
	)
	var weightMsgGuildMembershipRequestDeny int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildMembershipRequestDeny, &weightMsgGuildMembershipRequestDeny, nil,
		func(_ *rand.Rand) { weightMsgGuildMembershipRequestDeny = 5 },
	)
	var weightMsgGuildMembershipRequestRevoke int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildMembershipRequestRevoke, &weightMsgGuildMembershipRequestRevoke, nil,
		func(_ *rand.Rand) { weightMsgGuildMembershipRequestRevoke = 5 },
	)
	// Guild update operations
	var weightMsgGuildUpdateOwnerId int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildUpdateOwnerId, &weightMsgGuildUpdateOwnerId, nil,
		func(_ *rand.Rand) { weightMsgGuildUpdateOwnerId = 5 },
	)
	var weightMsgGuildUpdateEntrySubstationId int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildUpdateEntrySubstationId, &weightMsgGuildUpdateEntrySubstationId, nil,
		func(_ *rand.Rand) { weightMsgGuildUpdateEntrySubstationId = 5 },
	)
	var weightMsgGuildUpdateEndpoint int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildUpdateEndpoint, &weightMsgGuildUpdateEndpoint, nil,
		func(_ *rand.Rand) { weightMsgGuildUpdateEndpoint = 5 },
	)
	var weightMsgGuildUpdateJoinInfusionMin int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildUpdateJoinInfusionMin, &weightMsgGuildUpdateJoinInfusionMin, nil,
		func(_ *rand.Rand) { weightMsgGuildUpdateJoinInfusionMin = 5 },
	)
	var weightMsgGuildUpdateJoinInfusionBypassInvite int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildUpdateJoinInfusionBypassInvite, &weightMsgGuildUpdateJoinInfusionBypassInvite, nil,
		func(_ *rand.Rand) { weightMsgGuildUpdateJoinInfusionBypassInvite = 5 },
	)
	var weightMsgGuildUpdateJoinInfusionBypassRequest int
	simState.AppParams.GetOrGenerate(OpWeightMsgGuildUpdateJoinInfusionBypassRequest, &weightMsgGuildUpdateJoinInfusionBypassRequest, nil,
		func(_ *rand.Rand) { weightMsgGuildUpdateJoinInfusionBypassRequest = 5 },
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
		simulation.NewWeightedOperation(
			weightMsgAllocationCreate,
			structssimulation.SimulateMsgAllocationCreate(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgSubstationCreate,
			structssimulation.SimulateMsgSubstationCreate(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgProviderCreate,
			structssimulation.SimulateMsgProviderCreate(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		simulation.NewWeightedOperation(
			weightMsgAgreementOpen,
			structssimulation.SimulateMsgAgreementOpen(am.keeper, am.accountKeeper, am.bankKeeper),
		),
		// Agreement operations
		simulation.NewWeightedOperation(weightMsgAgreementClose, structssimulation.SimulateMsgAgreementClose(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgAgreementCapacityIncrease, structssimulation.SimulateMsgAgreementCapacityIncrease(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgAgreementCapacityDecrease, structssimulation.SimulateMsgAgreementCapacityDecrease(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgAgreementDurationIncrease, structssimulation.SimulateMsgAgreementDurationIncrease(am.keeper, am.accountKeeper, am.bankKeeper)),
		// Allocation operations
		simulation.NewWeightedOperation(weightMsgAllocationDelete, structssimulation.SimulateMsgAllocationDelete(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgAllocationUpdate, structssimulation.SimulateMsgAllocationUpdate(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgAllocationTransfer, structssimulation.SimulateMsgAllocationTransfer(am.keeper, am.accountKeeper, am.bankKeeper)),
		// Fleet operations
		simulation.NewWeightedOperation(weightMsgFleetMove, structssimulation.SimulateMsgFleetMove(am.keeper, am.accountKeeper, am.bankKeeper)),
		// Struct operations
		simulation.NewWeightedOperation(weightMsgStructBuildComplete, structssimulation.SimulateMsgStructBuildComplete(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgStructBuildCancel, structssimulation.SimulateMsgStructBuildCancel(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgStructActivate, structssimulation.SimulateMsgStructActivate(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgStructDeactivate, structssimulation.SimulateMsgStructDeactivate(am.keeper, am.accountKeeper, am.bankKeeper)),
		// Provider operations
		simulation.NewWeightedOperation(weightMsgProviderWithdrawBalance, structssimulation.SimulateMsgProviderWithdrawBalance(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgProviderUpdateCapacityMin, structssimulation.SimulateMsgProviderUpdateCapacityMinimum(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgProviderUpdateCapacityMax, structssimulation.SimulateMsgProviderUpdateCapacityMaximum(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgProviderUpdateDurationMin, structssimulation.SimulateMsgProviderUpdateDurationMinimum(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgProviderUpdateDurationMax, structssimulation.SimulateMsgProviderUpdateDurationMaximum(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgProviderUpdateAccessPolicy, structssimulation.SimulateMsgProviderUpdateAccessPolicy(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgProviderGuildGrant, structssimulation.SimulateMsgProviderGuildGrant(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgProviderGuildRevoke, structssimulation.SimulateMsgProviderGuildRevoke(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgProviderDelete, structssimulation.SimulateMsgProviderDelete(am.keeper, am.accountKeeper, am.bankKeeper)),
		// Substation operations
		simulation.NewWeightedOperation(weightMsgSubstationAllocationConnect, structssimulation.SimulateMsgSubstationAllocationConnect(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgSubstationAllocationDisconnect, structssimulation.SimulateMsgSubstationAllocationDisconnect(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgSubstationPlayerConnect, structssimulation.SimulateMsgSubstationPlayerConnect(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgSubstationPlayerDisconnect, structssimulation.SimulateMsgSubstationPlayerDisconnect(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgSubstationPlayerMigrate, structssimulation.SimulateMsgSubstationPlayerMigrate(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgSubstationDelete, structssimulation.SimulateMsgSubstationDelete(am.keeper, am.accountKeeper, am.bankKeeper)),
		// Address operations
		simulation.NewWeightedOperation(weightMsgAddressRevoke, structssimulation.SimulateMsgAddressRevoke(am.keeper, am.accountKeeper, am.bankKeeper)),
		// Player operations
		simulation.NewWeightedOperation(weightMsgPlayerUpdatePrimaryAddress, structssimulation.SimulateMsgPlayerUpdatePrimaryAddress(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgPlayerResume, structssimulation.SimulateMsgPlayerResume(am.keeper, am.accountKeeper, am.bankKeeper)),
		// Planet operations
		simulation.NewWeightedOperation(weightMsgPlanetRaidComplete, structssimulation.SimulateMsgPlanetRaidComplete(am.keeper, am.accountKeeper, am.bankKeeper)),
		// Reactor operations
		simulation.NewWeightedOperation(weightMsgReactorDefuse, structssimulation.SimulateMsgReactorDefuse(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgReactorBeginMigration, structssimulation.SimulateMsgReactorBeginMigration(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgReactorCancelDefusion, structssimulation.SimulateMsgReactorCancelDefusion(am.keeper, am.accountKeeper, am.bankKeeper)),
		// Guild membership operations
		simulation.NewWeightedOperation(weightMsgGuildMembershipInvite, structssimulation.SimulateMsgGuildMembershipInvite(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgGuildMembershipInviteApprove, structssimulation.SimulateMsgGuildMembershipInviteApprove(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgGuildMembershipInviteDeny, structssimulation.SimulateMsgGuildMembershipInviteDeny(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgGuildMembershipInviteRevoke, structssimulation.SimulateMsgGuildMembershipInviteRevoke(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgGuildMembershipJoinProxy, structssimulation.SimulateMsgGuildMembershipJoinProxy(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgGuildMembershipKick, structssimulation.SimulateMsgGuildMembershipKick(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgGuildMembershipRequestApprove, structssimulation.SimulateMsgGuildMembershipRequestApprove(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgGuildMembershipRequestDeny, structssimulation.SimulateMsgGuildMembershipRequestDeny(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgGuildMembershipRequestRevoke, structssimulation.SimulateMsgGuildMembershipRequestRevoke(am.keeper, am.accountKeeper, am.bankKeeper)),
		// Guild update operations
		simulation.NewWeightedOperation(weightMsgGuildUpdateOwnerId, structssimulation.SimulateMsgGuildUpdateOwnerId(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgGuildUpdateEntrySubstationId, structssimulation.SimulateMsgGuildUpdateEntrySubstationId(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgGuildUpdateEndpoint, structssimulation.SimulateMsgGuildUpdateEndpoint(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgGuildUpdateJoinInfusionMin, structssimulation.SimulateMsgGuildUpdateJoinInfusionMinimum(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgGuildUpdateJoinInfusionBypassInvite, structssimulation.SimulateMsgGuildUpdateJoinInfusionMinimumBypassByInvite(am.keeper, am.accountKeeper, am.bankKeeper)),
		simulation.NewWeightedOperation(weightMsgGuildUpdateJoinInfusionBypassRequest, structssimulation.SimulateMsgGuildUpdateJoinInfusionMinimumBypassByRequest(am.keeper, am.accountKeeper, am.bankKeeper)),
	)

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
