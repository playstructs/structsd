package app_test

import (
	"bytes"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// A paid guild may choose somebody else's reactor. Stake already on that
// primary must satisfy the join minimum without a same-validator redelegation
// or a token-migration permission that the operation does not exercise.
func TestGuildJoinCountsUncontrolledPrimaryWithoutRedelegation(t *testing.T) {
	for _, scenario := range []struct {
		name             string
		permission       types.Permission
		minimum          uint64
		differentPrimary bool
		expectedError    string
		slash            bool
	}{
		{"all_permissions", types.PermAll, 1000000, false, "", false},
		{"membership_only", types.PermGuildMembership, 1000000, false, "", false},
		{"primary_below_minimum", types.PermGuildMembership, 1000001, false, "join infusion minimum not met", false},
		{"outside_primary_requires_migration", types.PermGuildMembership, 1000000, true, "token_migrate", false},
		{"pending_slash_above_backing", types.PermGuildMembership, 975000, false, "join infusion minimum not met", true},
		{"pending_slash_at_backing", types.PermGuildMembership, 950000, false, "", true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			validator := createRedelegationValidator(t, bApp, ctx, staking, bondDenom, "independent-primary")
			_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
			require.NoError(t, err)
			reactorID := jailGateReactorId(t, bApp, ctx, validator)
			primaryID := reactorID
			if scenario.differentPrimary {
				otherValidator := createRedelegationValidator(t, bApp, ctx, staking, bondDenom, "other-primary")
				_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
				require.NoError(t, err)
				primaryID = jailGateReactorId(t, bApp, ctx, otherValidator)
			}
			reactor, found := bApp.StructsKeeper.GetReactor(ctx, reactorID)
			require.True(t, found)
			foreign := types.CreateEmptyGuild()
			foreign.Id, foreign.Index, foreign.Owner = "0-986", 986, reactor.Owner
			bApp.StructsKeeper.SetGuild(ctx, foreign)
			reactor.GuildId = foreign.Id
			bApp.StructsKeeper.SetReactor(ctx, reactor)
			founder := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x6d}, 20)))
			founderVaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(founder.Id)
			require.NoError(t, err)
			fundRedelegationAccount(t, bApp, ctx, founderVaults[0], sdk.NewCoins(sdk.NewInt64Coin("ualpha", 2000000)))
			server := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
			created, err := server.GuildCreate(ctx, &types.MsgGuildCreate{Creator: founder.Creator, ReactorId: primaryID})
			require.NoError(t, err)
			guild, found := bApp.StructsKeeper.GetGuild(ctx, created.GuildId)
			require.True(t, found)
			guild.JoinInfusionMinimum = scenario.minimum
			bApp.StructsKeeper.SetGuild(ctx, guild)
			actor := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()))
			vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(actor.Id)
			require.NoError(t, err)
			vault := vaults[0]
			stake := sdk.NewInt64Coin(bondDenom, 1000000)
			fundRedelegationAccount(t, bApp, ctx, vault, sdk.NewCoins(stake))
			_, err = staking.Delegate(ctx, stakingtypes.NewMsgDelegate(vault.String(), validator.String(), stake))
			require.NoError(t, err)
			infusion, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, vault.String())
			require.True(t, found)
			require.Equal(t, actor.Id, infusion.PlayerId)
			require.Equal(t, uint64(1000000), infusion.Fuel)
			if scenario.slash {


				value, err := bApp.StakingKeeper.GetValidator(ctx, validator)
				require.NoError(t, err)
				consensus, err := value.GetConsAddr()
				require.NoError(t, err)
				_, err = bApp.StakingKeeper.Slash(ctx, consensus, ctx.BlockHeight(), value.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx)), math.LegacyNewDecWithPrec(5, 2))
				require.NoError(t, err)
				value, err = bApp.StakingKeeper.GetValidator(ctx, validator)
				require.NoError(t, err)
				live, err := bApp.StakingKeeper.GetDelegation(ctx, vault, validator)
				require.NoError(t, err)
				require.Equal(t, int64(950000), value.TokensFromShares(live.Shares).TruncateInt().Int64())
				infusion, found = bApp.StructsKeeper.GetInfusion(ctx, reactorID, vault.String())
				require.True(t, found)
				require.Equal(t, uint64(1000000), infusion.Fuel, "bounded reconciliation has not reached the slashed position")
				reactor, found = bApp.StructsKeeper.GetReactor(ctx, reactorID)
				require.True(t, found)
			}
			delegation, err := bApp.StakingKeeper.GetDelegation(ctx, vault, validator)
			require.NoError(t, err)
			capacity := redelegationCapacity(t, bApp, ctx, actor.Id)
			bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(actor.Creator), scenario.permission)
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			_, err = server.GuildMembershipJoin(ctx, &types.MsgGuildMembershipJoin{Creator: actor.Creator, PlayerId: actor.Id, GuildId: guild.Id, InfusionId: []string{reactorID + "-" + vault.String()}})
			if scenario.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, scenario.expectedError)
			}
			joined, found := bApp.StructsKeeper.GetPlayer(ctx, actor.Id)
			require.True(t, found)
			if scenario.expectedError == "" {
				require.Equal(t, guild.Id, joined.GuildId)
			} else {
				require.Empty(t, joined.GuildId)
			}
			after, err := bApp.StakingKeeper.GetDelegation(ctx, vault, validator)
			require.NoError(t, err)
			require.Equal(t, delegation, after)
			afterInfusion, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, vault.String())
			require.True(t, found)
			require.Equal(t, infusion, afterInfusion, "membership changes cannot reprice an unchanged infusion")
			require.Equal(t, capacity, redelegationCapacity(t, bApp, ctx, actor.Id))
			afterReactor, found := bApp.StructsKeeper.GetReactor(ctx, reactorID)
			require.True(t, found)
			require.Equal(t, reactor, afterReactor, "selecting a primary and admitting a member cannot take over its reactor")
			for _, event := range ctx.EventManager().Events() {
				require.NotContains(t, []string{stakingtypes.EventTypeRedelegate, stakingtypes.EventTypeDelegate, stakingtypes.EventTypeUnbond, "transfer", "coin_spent", "coin_received"}, event.Type)
			}
		})
	}
}
