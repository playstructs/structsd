package app_test

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestPrimaryRotationRetainsMigratedOperatorSelfStake(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	validator := createRedelegationValidator(t, bApp, ctx, staking, denom, "retainedrotation")
	operator := sdk.AccAddress(validator)
	secondary := sdk.AccAddress(bytes.Repeat([]byte{0x46}, 20))
	index := bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, operator.String())
	require.NoError(t, bApp.StructsKeeper.SetPlayerIndexForAddress(ctx, secondary.String(), index))
	require.NoError(t, v22.MigrateVaultAssets(ctx, delegationUpgradeKeepers(bApp)))
	before, err := bApp.StakingKeeper.GetDelegation(ctx, operator, validator)
	require.NoError(t, err)
	server := keeper.NewMsgServerImpl(bApp.StructsKeeper)
	_, err = server.PlayerUpdatePrimaryAddress(ctx, &types.MsgPlayerUpdatePrimaryAddress{Creator: operator.String(), PrimaryAddress: secondary.String()})
	require.NoError(t, err)
	after, err := bApp.StakingKeeper.GetDelegation(ctx, operator, validator)
	require.NoError(t, err)
	require.Equal(t, before, after)
	_, err = bApp.StakingKeeper.GetDelegation(ctx, secondary, validator)
	require.Error(t, err)
	player, err := bApp.StructsKeeper.NewCurrentContext(ctx).GetPlayerByAddress(operator.String())
	require.NoError(t, err)
	require.Equal(t, secondary.String(), player.GetPrimaryAddress())
	require.Equal(t, index, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, operator.String()))
	_, err = bApp.DistrKeeper.WithdrawDelegationRewards(ctx, operator, validator)
	require.NoError(t, err)
	_, err = server.AddressRevoke(ctx, &types.MsgAddressRevoke{Creator: secondary.String(), Address: operator.String()})
	require.NoError(t, err, "self-stake must not prevent emergency key revocation")
	require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, operator.String()))
	retained, err := bApp.StakingKeeper.GetDelegation(ctx, operator, validator)
	require.NoError(t, err)
	require.Equal(t, before, retained, "revocation must not rekey native operator stake")
	reactorID := jailGateReactorId(t, bApp, ctx, validator)
	infusion, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, operator.String())
	if found {
		require.Zero(t, infusion.Power, "revoked stake must not keep crediting the player")
	}
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
