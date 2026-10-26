package app_test

import (
	"encoding/hex"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestOperatorRegistrationRehomesRetainedInfusionWithoutMovingStake(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	oldPrimary, _, oldPlayerID := newTransferPlayer(t, bApp, ctx)
	key := secp256k1.GenPrivKey()
	operator := sdk.AccAddress(key.PubKey().Address())
	index := bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, oldPrimary.String())
	require.NoError(t, bApp.StructsKeeper.SetPlayerIndexForAddress(ctx, operator.String(), index))
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	stake := sdk.NewInt64Coin(denom, 10000000)
	fundRedelegationAccount(t, bApp, ctx, operator, sdk.NewCoins(stake))
	validator := sdk.ValAddress(operator)
	create, err := stakingtypes.NewMsgCreateValidator(validator.String(), ed25519.GenPrivKey().PubKey(), stake,
		stakingtypes.NewDescription("registeredoperator", "", "", "", ""),
		stakingtypes.NewCommissionRates(math.LegacyZeroDec(), math.LegacyOneDec(), math.LegacyOneDec()), math.OneInt())
	require.NoError(t, err)
	staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	_, err = staking.CreateValidator(ctx, create)
	require.NoError(t, err)
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	require.NoError(t, v22.MigrateVaultAssets(ctx, delegationUpgradeKeepers(bApp)))
	before, err := bApp.StakingKeeper.GetDelegation(ctx, operator, validator)
	require.NoError(t, err)
	server := keeper.NewMsgServerImpl(bApp.StructsKeeper)
	beforeOwnershipCapacity := redelegationCapacity(t, bApp, ctx, oldPlayerID)
	_, err = server.AddressRevoke(ctx, &types.MsgAddressRevoke{Creator: oldPrimary.String(), Address: operator.String()})
	require.NoError(t, err)
	oldCapacity := redelegationCapacity(t, bApp, ctx, oldPlayerID)
	transferredCapacity := beforeOwnershipCapacity - oldCapacity
	require.Positive(t, transferredCapacity)
	primary, _, playerID := newTransferPlayer(t, bApp, ctx)
	newCapacity := redelegationCapacity(t, bApp, ctx, playerID)
	nonce := bApp.StructsKeeper.GetAddressProofNonce(ctx, operator.String())
	proof := types.AddressRegisterProofInput(ctx.ChainID(), playerID, operator.String(), nonce)
	signature, err := key.Sign([]byte(proof))
	require.NoError(t, err)
	_, err = server.AddressRegister(ctx, &types.MsgAddressRegister{Creator: primary.String(), PlayerId: playerID,
		Address: operator.String(), Permissions: uint64(types.PermAll), ProofPubKey: hex.EncodeToString(key.PubKey().Bytes()), ProofSignature: hex.EncodeToString(signature)})
	require.NoError(t, err)
	after, err := bApp.StakingKeeper.GetDelegation(ctx, operator, validator)
	require.NoError(t, err)
	require.Equal(t, before, after)
	reactorID := jailGateReactorId(t, bApp, ctx, validator)
	infusion, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, operator.String())
	require.True(t, found)
	require.Equal(t, playerID, infusion.PlayerId)
	require.Equal(t, uint64(10000000), infusion.Fuel)
	require.Equal(t, uint64(10000000), infusion.Power)
	require.Equal(t, newCapacity+transferredCapacity, redelegationCapacity(t, bApp, ctx, playerID))
	require.Equal(t, oldCapacity, redelegationCapacity(t, bApp, ctx, oldPlayerID))
	entry := keeper.InfusionOwnerEntry{DestinationId: reactorID, Address: operator.String()}
	oldEntries, oldMore, _ := bApp.StructsKeeper.GetInfusionOwnerEntriesUpTo(ctx, oldPlayerID, 10)
	require.False(t, oldMore)
	require.NotContains(t, oldEntries, entry, "registration must not leave the previous owner's inventory pointing at the new owner's stake")
	newEntries, newMore, _ := bApp.StructsKeeper.GetInfusionOwnerEntriesUpTo(ctx, playerID, 10)
	require.False(t, newMore)
	require.Contains(t, newEntries, entry, "the canonical owner, owner index and both capacities must agree")
	_, found = bApp.StructsKeeper.GetReactorDiversityPosition(ctx, oldPlayerID, reactorID)
	require.False(t, found, "revoking retained self-stake removes the old owner's diversity position")
	position, found := bApp.StructsKeeper.GetReactorDiversityPosition(ctx, playerID, reactorID)
	require.True(t, found)
	_, _, share := infusion.GetPowerDistribution()
	require.Equal(t, math.NewIntFromUint64(share).String(), position.Power)
	require.NoError(t, bApp.StructsKeeper.CheckReactorDiversityAccounting(ctx))
	_, err = bApp.StakingKeeper.GetDelegation(ctx, primary, validator)
	require.Error(t, err, "registration cannot rekey self-stake onto the new primary")
	_, err = bApp.DistrKeeper.WithdrawDelegationRewards(ctx, operator, validator)
	require.NoError(t, err)
	require.Equal(t, nonce+1, bApp.StructsKeeper.GetAddressProofNonce(ctx, operator.String()))
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
