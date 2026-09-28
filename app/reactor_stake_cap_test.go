package app_test

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func activateTestStakeCap(t *testing.T, bApp *app.App, ctx sdk.Context) {
	t.Helper()
	p := bApp.StructsKeeper.GetParams(ctx).WithReactorDefaults()
	p.ReactorStakeCapMinValidators = 4
	require.NoError(t, bApp.StructsKeeper.SetParams(ctx, p))
	require.NoError(t, bApp.StructsKeeper.EvaluateReactorStakeCap(ctx))
	require.True(t, bApp.StructsKeeper.ReactorStakeCapActive(ctx))
}

func TestReactorStakeCapNativeAndVaultExactBoundary(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(map[bool]string{true: "native", false: "vault"}[native], func(t *testing.T) {
			bApp, ctx := setupJailGateAppWithValidators(t, 4)
			primary, _, owner := newTransferPlayer(t, bApp, ctx)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			address := primary
			if !native {
				address, err = types.VaultAddress(owner)
				require.NoError(t, err)
			}
			fundRedelegationAccount(t, bApp, ctx, address, sdk.NewCoins(sdk.NewInt64Coin(denom, 1000000)))
			activateTestStakeCap(t, bApp, ctx)
			reactor := jailGateReactorId(t, bApp, ctx, validator)
			query, err := bApp.StructsKeeper.ReactorStakeShare(ctx, &types.QueryReactorStakeShareRequest{ReactorId: reactor})
			require.NoError(t, err)
			require.Equal(t, "285714", query.ExternalDepositHeadroom)
			require.Equal(t, "200000", query.BondedRedelegationHeadroom)
			nativeServer := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			structsServer := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
			deposit := func(tx sdk.Context, amount int64) error {
				if native {
					_, err := nativeServer.Delegate(tx, stakingtypes.NewMsgDelegate(address.String(), validator.String(), sdk.NewInt64Coin(denom, amount)))
					return err
				}
				_, err := structsServer.ReactorInfuse(tx, &types.MsgReactorInfuse{Creator: primary.String(), ValidatorAddress: validator.String(), Amount: sdk.NewInt64Coin(denom, amount)})
				return err
			}
			before := applicationAssetStateRows(bApp, ctx)
			stakingBefore := bApp.StakingKeeper.ExportGenesis(ctx)
			tx, _ := ctx.CacheContext()
			require.ErrorIs(t, deposit(tx, 285715), types.ErrReactorStakeCapExceeded)
			require.Equal(t, before, applicationAssetStateRows(bApp, ctx))
			require.Equal(t, stakingBefore, bApp.StakingKeeper.ExportGenesis(ctx))
			require.NoError(t, deposit(ctx, 285714), "failed cached attempt cannot poison the next snapshot")
			tx, _ = ctx.CacheContext()
			require.ErrorIs(t, deposit(tx, 1), types.ErrReactorStakeCapExceeded)
			row, found := bApp.StructsKeeper.GetInfusion(ctx, reactor, address.String())
			require.True(t, found)
			require.Equal(t, uint64(285714), row.Power)
			requireStakingAndDistributionHealthy(t, bApp, ctx)
		})
	}
}

func TestReactorStakeCapRejectsCancellationButAllowsOutflowAndSlash(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 4)
	address, _, _ := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	delegate(t, bApp, ctx, server, denom, address, validator, math.NewInt(2000000))
	activateTestStakeCap(t, bApp, ctx)
	_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(address.String(), validator.String(), sdk.NewInt64Coin(denom, 1000000)))
	require.NoError(t, err, "over-cap stake may leave")
	pending, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, address, validator)
	require.NoError(t, err)
	tx, _ := ctx.CacheContext()
	_, err = server.CancelUnbondingDelegation(tx, &stakingtypes.MsgCancelUnbondingDelegation{DelegatorAddress: address.String(), ValidatorAddress: validator.String(), CreationHeight: pending.Entries[0].CreationHeight, Amount: sdk.NewInt64Coin(denom, 1)})
	require.ErrorIs(t, err, types.ErrReactorStakeCapExceeded)
	value, err := bApp.StakingKeeper.GetValidator(ctx, validator)
	require.NoError(t, err)
	consensus, err := value.GetConsAddr()
	require.NoError(t, err)
	_, err = bApp.StakingKeeper.Slash(ctx, consensus, ctx.BlockHeight(), value.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx)), math.LegacyNewDecWithPrec(5, 2))
	require.NoError(t, err)
	_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(address.String(), validator.String(), sdk.NewInt64Coin(denom, 950000)))
	require.NoError(t, err, "full removal consumes its before snapshot without requiring an after hook")
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

func TestReactorStakeCapGuildJoinRollsBackEarlierSuccessfulMigration(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 4)
	primary, _, owner := newTransferPlayer(t, bApp, ctx)
	vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(owner)
	require.NoError(t, err)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	addresses := make([]sdk.ValAddress, 3)
	for i := range addresses {
		addresses[i], err = sdk.ValAddressFromBech32(validators[i].OperatorAddress)
		require.NoError(t, err)
	}
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	delegate(t, bApp, ctx, server, denom, vaults[0], addresses[0], math.NewInt(100000))
	delegate(t, bApp, ctx, server, denom, vaults[0], addresses[1], math.NewInt(300000))
	activateTestStakeCap(t, bApp, ctx)
	target := jailGateReactorId(t, bApp, ctx, addresses[2])
	guild := types.CreateEmptyGuild()
	guild.Id, guild.Index, guild.Owner, guild.Creator = "0-913", 913, owner, primary.String()
	guild.PrimaryReactorId, guild.JoinInfusionMinimum = target, 400000
	bApp.StructsKeeper.SetGuild(ctx, guild)
	msg := &types.MsgGuildMembershipJoin{Creator: primary.String(), PlayerId: owner, GuildId: guild.Id,
		InfusionId: []string{jailGateReactorId(t, bApp, ctx, addresses[0]) + "-" + vaults[0].String(), jailGateReactorId(t, bApp, ctx, addresses[1]) + "-" + vaults[0].String()}}
	before, nativeBefore := applicationAssetStateRows(bApp, ctx), bApp.StakingKeeper.ExportGenesis(ctx)
	events := len(ctx.EventManager().Events())
	tx, _ := ctx.CacheContext()
	_, err = keeperlib.NewMsgServerImpl(bApp.StructsKeeper).GuildMembershipJoin(tx, msg)
	require.ErrorIs(t, err, types.ErrReactorStakeCapExceeded)
	migrated, err := bApp.StakingKeeper.GetRedelegation(tx, vaults[0], addresses[0], addresses[2])
	require.NoError(t, err, "first migration must have succeeded inside the discarded message")
	require.Len(t, migrated.Entries, 1)
	require.Equal(t, before, applicationAssetStateRows(bApp, ctx))
	require.Equal(t, nativeBefore, bApp.StakingKeeper.ExportGenesis(ctx))
	require.Len(t, ctx.EventManager().Events(), events)
	p := bApp.StructsKeeper.GetParams(ctx)
	p.ReactorStakeCapBps = 4000
	require.NoError(t, bApp.StructsKeeper.SetParams(ctx, p))
	_, err = keeperlib.NewMsgServerImpl(bApp.StructsKeeper).GuildMembershipJoin(ctx, msg)
	require.NoError(t, err)
	player, found := bApp.StructsKeeper.GetPlayer(ctx, owner)
	require.True(t, found)
	require.Equal(t, guild.Id, player.GuildId)
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

func TestReactorStakeCapSnapshotIsConsumedAndCannotBeOmitted(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 4)
	primary, _, _ := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	activateTestStakeCap(t, bApp, ctx)
	hooks := bApp.StructsKeeper.Hooks()
	require.ErrorContains(t, hooks.AfterDelegationModified(ctx, primary, validator), "stake snapshot")
	require.NoError(t, hooks.BeforeDelegationCreated(ctx, primary, validator))
	require.NoError(t, hooks.AfterDelegationModified(ctx, primary, validator))
	require.ErrorContains(t, hooks.AfterDelegationModified(ctx, primary, validator), "stake snapshot")
}

func newCapValidatorMessage(t *testing.T, bApp *app.App, ctx sdk.Context, amount int64) (*stakingtypes.MsgCreateValidator, sdk.ValAddress) {
	t.Helper()
	operator := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	fundRedelegationAccount(t, bApp, ctx, operator, sdk.NewCoins(sdk.NewInt64Coin(denom, amount)))
	msg, err := stakingtypes.NewMsgCreateValidator(sdk.ValAddress(operator).String(), ed25519.GenPrivKey().PubKey(), sdk.NewInt64Coin(denom, amount), stakingtypes.NewDescription("cap-validator", "", "", "", ""), stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(1, 1), math.LegacyNewDecWithPrec(2, 1), math.LegacyNewDecWithPrec(1, 2)), math.OneInt())
	require.NoError(t, err)
	return msg, sdk.ValAddress(operator)
}

func TestReactorStakeCapCoversNewValidatorAndAuthz(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 4)
	activateTestStakeCap(t, bApp, ctx)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	for _, amount := range []int64{2000000, 500000} {
		msg, address := newCapValidatorMessage(t, bApp, ctx, amount)
		tx, write := ctx.CacheContext()
		_, err := server.CreateValidator(tx, msg)
		if amount == 2000000 {
			require.ErrorIs(t, err, types.ErrReactorStakeCapExceeded)
			_, err = bApp.StakingKeeper.GetValidator(ctx, address)
			require.ErrorIs(t, err, stakingtypes.ErrNoValidatorFound)
		} else {
			require.NoError(t, err)
			write()
		}
	}
	primary, _, _ := newTransferPlayer(t, bApp, ctx)
	grantee, _, _ := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	fundRedelegationAccount(t, bApp, ctx, primary, sdk.NewCoins(sdk.NewInt64Coin(denom, 1000000)))
	require.NoError(t, bApp.AuthzKeeper.SaveGrant(ctx, grantee, primary, authz.NewGenericAuthorization(sdk.MsgTypeURL(&stakingtypes.MsgDelegate{})), nil))
	msg := authz.NewMsgExec(grantee, []sdk.Msg{stakingtypes.NewMsgDelegate(primary.String(), validator.String(), sdk.NewInt64Coin(denom, 1000000))})
	before, nativeBefore := applicationAssetStateRows(bApp, ctx), bApp.StakingKeeper.ExportGenesis(ctx)
	tx, _ := ctx.CacheContext()
	_, err = bApp.AuthzKeeper.Exec(tx, &msg)
	require.ErrorIs(t, err, types.ErrReactorStakeCapExceeded)
	require.Equal(t, before, applicationAssetStateRows(bApp, ctx))
	require.Equal(t, nativeBefore, bApp.StakingKeeper.ExportGenesis(ctx))
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

func TestReactorStakeCapRedelegationPoolCombinations(t *testing.T) {
	for _, sourceBonded := range []bool{false, true} {
		for _, targetBonded := range []bool{false, true} {
			t.Run(fmt.Sprintf("source-bonded=%t/target-bonded=%t", sourceBonded, targetBonded), func(t *testing.T) {
				bApp, ctx := setupJailGateAppWithValidators(t, 4)
				primary, _, _ := newTransferPlayer(t, bApp, ctx)
				validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
				require.NoError(t, err)
				source, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
				require.NoError(t, err)
				target, err := sdk.ValAddressFromBech32(validators[1].OperatorAddress)
				require.NoError(t, err)
				server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
				if !sourceBonded {
					msg, address := newCapValidatorMessage(t, bApp, ctx, 1000000)
					_, err = server.CreateValidator(ctx, msg)
					require.NoError(t, err)
					source = address
				}
				if !targetBonded {
					msg, address := newCapValidatorMessage(t, bApp, ctx, 1000000)
					_, err = server.CreateValidator(ctx, msg)
					require.NoError(t, err)
					target = address
				}
				denom, err := bApp.StakingKeeper.BondDenom(ctx)
				require.NoError(t, err)
				delegate(t, bApp, ctx, server, denom, primary, source, math.NewInt(1000000))
				activateTestStakeCap(t, bApp, ctx)
				before, nativeBefore := applicationAssetStateRows(bApp, ctx), bApp.StakingKeeper.ExportGenesis(ctx)
				for _, amount := range []int64{900000, 1000000} {
					tx, _ := ctx.CacheContext()
					_, err = server.BeginRedelegate(tx, stakingtypes.NewMsgBeginRedelegate(primary.String(), source.String(), target.String(), sdk.NewInt64Coin(denom, amount)))
					require.ErrorIs(t, err, types.ErrReactorStakeCapExceeded)
					require.Equal(t, before, applicationAssetStateRows(bApp, ctx))
					require.Equal(t, nativeBefore, bApp.StakingKeeper.ExportGenesis(ctx))
				}
				_, err = server.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(primary.String(), source.String(), target.String(), sdk.NewInt64Coin(denom, 100000)))
				require.NoError(t, err)
				requireStakingAndDistributionHealthy(t, bApp, ctx)
			})
		}
	}
}

func TestReactorStakeCapUsesProductionGenesisThreshold(t *testing.T) {
	for _, count := range []int{9, 10} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			bApp, ctx := setupJailGateAppWithValidators(t, count)
			require.Equal(t, count >= 10, bApp.StructsKeeper.ReactorStakeCapActive(ctx))
			p := bApp.StructsKeeper.GetParams(ctx)
			require.Equal(t, uint64(3000), p.ReactorStakeCapBps)
			require.Equal(t, uint64(10), p.ReactorStakeCapMinValidators)
		})
	}
}
