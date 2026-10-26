package app_test

import (
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

// CometBFT verifies the evidence before presenting Misbehavior to the app.
// This exercises its ABCI delivery, the production upgrade, real slashing
// hooks and public block events; it does not test CometBFT signature checking.
func TestRetainedStakeSourceEvidenceSlashesOnceThroughFinalizeBlock(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	start := ctx.BlockTime()
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	source := createRedelegationValidator(t, bApp, ctx, server, denom, "evidence-source")
	destination := createRedelegationValidator(t, bApp, ctx, server, denom, "evidence-destination")
	primary, _, _ := newTransferPlayer(t, bApp, ctx)
	delegate(t, bApp, ctx, server, denom, primary, source, math.NewInt(2000000))
	params, err := bApp.SlashingKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.SlashFractionDoubleSign = math.LegacyNewDecWithPrec(15, 2)
	require.NoError(t, bApp.SlashingKeeper.SetParams(ctx, params))
	const upgradeName = "test-vault-evidence-upgrade"
	require.NoError(t, bApp.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{Name: upgradeName, Height: 4}))
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: start})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: start.Add(time.Second)})
	require.NoError(t, err)
	ctx = bApp.NewContextLegacy(false, cmtproto.Header{Height: 2, ChainID: jailGateChainID, Time: start.Add(time.Second)})
	historical, err := bApp.StakingKeeper.GetValidator(ctx, source)
	require.NoError(t, err)
	require.True(t, historical.IsBonded())
	consensus, err := historical.GetConsAddr()
	require.NoError(t, err)
	require.True(t, bApp.SlashingKeeper.HasValidatorSigningInfo(ctx, consensus))
	historicalPower := historical.GetConsensusPower(bApp.StakingKeeper.PowerReduction(ctx))
	_, err = server.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(primary.String(), source.String(), destination.String(), sdk.NewInt64Coin(denom, 1000000)))
	require.NoError(t, err)
	_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(primary.String(), destination.String(), sdk.NewInt64Coin(denom, 100000)))
	require.NoError(t, err)
	// Seed the legacy pending operations through real keepers after this
	// fixture block's FinalizeBlock, then persist that branch before Commit.
	bApp.SimWriteState()
	_, err = bApp.Commit()
	require.NoError(t, err)
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 3, Time: start.Add(2 * time.Second)})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	bApp.UpgradeKeeper.SetUpgradeHandler(upgradeName, v22.CreateUpgradeHandler(bApp.ModuleManager, bApp.Configurator(), delegationUpgradeKeepers(bApp)))
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 4, Time: start.Add(3 * time.Second)})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	ctx = bApp.NewContextLegacy(true, cmtproto.Header{Height: 5, ChainID: jailGateChainID, Time: start.Add(4 * time.Second)})
	vault := primary // Legacy stake remains at the original address.
	redelegation, err := bApp.StakingKeeper.GetRedelegation(ctx, vault, source, destination)
	require.NoError(t, err)
	require.Len(t, redelegation.Entries, 1)
	require.Equal(t, int64(2), redelegation.Entries[0].CreationHeight)
	beforeSupply := bApp.BankKeeper.GetSupply(ctx, denom).Amount
	beforeLiquid := bApp.BankKeeper.GetAllBalances(ctx, vault)
	beforeSource, err := bApp.StakingKeeper.GetValidator(ctx, source)
	require.NoError(t, err)
	beforeDestination, err := bApp.StakingKeeper.GetValidator(ctx, destination)
	require.NoError(t, err)
	// The validator set published at block 1 signs block 3. SDK evidence
	// processing subtracts ValidatorUpdateDelay, reaching the block-2 move.
	evidence := []abci.Misbehavior{{
		Type:      abci.MisbehaviorType_DUPLICATE_VOTE,
		Validator: abci.Validator{Address: consensus, Power: historicalPower},
		Height:    3, Time: start.Add(2 * time.Second), TotalVotingPower: historicalPower + 11,
	}}
	result, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 5, Time: ctx.BlockTime(), Misbehavior: evidence})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	ctx = bApp.NewContextLegacy(true, cmtproto.Header{Height: 5, ChainID: jailGateChainID, Time: ctx.BlockTime()})
	wantTotal := params.SlashFractionDoubleSign.MulInt(bApp.StakingKeeper.TokensFromConsensusPower(ctx, historicalPower)).TruncateInt()
	// Upstream v0.53.8 burns the full 150,000 active penalty after taking
	// 100,000 from pending withdrawals. Preserve event/accounting coverage of
	// that accepted upstream defect; the opt-in reproduction asserts its fix.
	wantBurn := wantTotal.AddRaw(100000)
	require.True(t, beforeSupply.Sub(bApp.BankKeeper.GetSupply(ctx, denom).Amount).Equal(wantBurn))
	require.Equal(t, beforeLiquid, bApp.BankKeeper.GetAllBalances(ctx, vault))
	afterSource, err := bApp.StakingKeeper.GetValidator(ctx, source)
	require.NoError(t, err)
	require.True(t, afterSource.IsJailed())
	require.True(t, bApp.SlashingKeeper.IsTombstoned(ctx, consensus))
	require.True(t, beforeSource.Tokens.Sub(afterSource.Tokens).Equal(wantTotal.SubRaw(150000)))
	afterDestination, err := bApp.StakingKeeper.GetValidator(ctx, destination)
	require.NoError(t, err)
	require.Equal(t, int64(150000), beforeDestination.Tokens.Sub(afterDestination.Tokens).Int64(), "upstream burns the full active fraction despite partial pending payment")
	pending, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, vault, destination)
	require.NoError(t, err)
	require.Len(t, pending.Entries, 1)
	require.True(t, pending.Entries[0].Balance.IsZero())
	_, err = bApp.StakingKeeper.GetRedelegation(ctx, primary, source, destination)
	require.NoError(t, err, "the evidence must reach the retained source record")
	wantInfusions := map[string]types.Infusion{}
	for _, validator := range []sdk.ValAddress{source, destination} {
		reactor := jailGateReactorId(t, bApp, ctx, validator)
		infusion, found := bApp.StructsKeeper.GetInfusion(ctx, reactor, vault.String())
		require.True(t, found)
		wantedFuel := uint64(850000)
		if validator.Equals(destination) {
			wantedFuel = 750000 // Includes the upstream excess active burn.
		}
		require.Equal(t, wantedFuel, infusion.Fuel)
		require.Zero(t, infusion.Defusing)
		wantInfusions[reactor] = infusion
		if validator.Equals(source) {
			require.Zero(t, infusion.Power)
		} else {
			require.NotZero(t, infusion.Power)
		}
	}
	emittedBurn := math.ZeroInt()
	slashEvents := 0
	emittedInfusions := map[string]types.Infusion{}
	for _, event := range result.Events {
		if event.Type == "structs.structs.EventInfusion" {
			// BaseApp's raw mode=BeginBlock/EndBlock attribute is an
			// envelope label, not part of the protobuf JSON payload.
			payload := abci.Event{Type: event.Type}
			for _, attribute := range event.Attributes {
				if attribute.Key != "mode" {
					payload.Attributes = append(payload.Attributes, attribute)
				}
			}
			decoded, err := sdk.ParseTypedEvent(payload)
			require.NoError(t, err)
			infusion := decoded.(*types.EventInfusion).Infusion
			if infusion.Address == vault.String() {
				emittedInfusions[infusion.DestinationId] = *infusion
			}
		}
		attributes := map[string]string{}
		for _, attribute := range event.Attributes {
			attributes[attribute.Key] = attribute.Value
		}
		if event.Type == banktypes.EventTypeCoinBurn {
			coins, err := sdk.ParseCoinsNormalized(attributes[sdk.AttributeKeyAmount])
			require.NoError(t, err)
			emittedBurn = emittedBurn.Add(coins.AmountOf(denom))
		}
		if event.Type == slashingtypes.EventTypeSlash && attributes[slashingtypes.AttributeKeyReason] == slashingtypes.AttributeValueDoubleSign {
			slashEvents++
			require.Equal(t, sdk.ConsAddress(consensus).String(), attributes[slashingtypes.AttributeKeyAddress])
			require.Equal(t, strconv.FormatInt(historicalPower, 10), attributes[slashingtypes.AttributeKeyPower])
			// The SDK slash summary reports the source-validator burn only.
			// Bank burn events also include pending/active destination stake.
			require.Equal(t, wantTotal.SubRaw(150000).String(), attributes[slashingtypes.AttributeKeyBurnedCoins])
		}
	}
	require.True(t, emittedBurn.Equal(wantBurn), "public burn events must equal the full source and destination supply reduction")
	require.Equal(t, 1, slashEvents)
	requireInfusionEventsEqual(t, wantInfusions, emittedInfusions, "the last infusion event for each moved position must match committed state")
	requireStakingAndDistributionHealthy(t, bApp, ctx)
	// Repeated evidence for an already tombstoned validator must not charge
	// the moved stake again, nor publish another slash or burn.
	again, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 6, Time: start.Add(5 * time.Second), Misbehavior: evidence})
	require.NoError(t, err)
	for _, event := range again.Events {
		require.NotEqual(t, banktypes.EventTypeCoinBurn, event.Type)
		require.NotEqual(t, slashingtypes.EventTypeSlash, event.Type)
	}
	_, err = bApp.Commit()
	require.NoError(t, err)
	ctx = bApp.NewContextLegacy(true, cmtproto.Header{Height: 6, ChainID: jailGateChainID, Time: start.Add(5 * time.Second)})
	require.True(t, beforeSupply.Sub(bApp.BankKeeper.GetSupply(ctx, denom).Amount).Equal(wantBurn))
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
