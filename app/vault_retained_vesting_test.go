package app_test

import (
	"bytes"
	"encoding/hex"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Exercise the native bank/staking/distribution hooks, including an account
// whose only remaining self-stake reference is a held redelegation record.
func TestVaultMigrationRetainsWholeNativeVestingAccount(t *testing.T) {
	for _, kind := range []string{"continuous", "delayed", "periodic", "permanent"} {
		for _, position := range []string{"bonded", "self_unbonding", "redelegation_with_self_unbonding", "held_redelegation_only"} {
			t.Run(kind+"/"+position, func(t *testing.T) {
				bApp, ctx := setupJailGateApp(t)
				denom, err := bApp.StakingKeeper.BondDenom(ctx)
				require.NoError(t, err)
				coins := func(n int64) sdk.Coins { return sdk.NewCoins(sdk.NewInt64Coin(denom, n)) }
				validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
				require.NoError(t, err)
				other, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
				require.NoError(t, err)
				operatorKey := secp256k1.GenPrivKey()
				operator := sdk.AccAddress(operatorKey.PubKey().Address())
				self := sdk.ValAddress(operator)
				fundRedelegationAccount(t, bApp, ctx, operator, coins(20000000))
				base := bApp.AccountKeeper.GetAccount(ctx, operator).(*authtypes.BaseAccount)
				start := ctx.BlockTime().Unix()
				end := ctx.BlockTime().Add(365 * 24 * time.Hour).Unix()
				var account vestingexported.VestingAccount
				switch kind {
				case "continuous":
					account, err = vestingtypes.NewContinuousVestingAccount(base, coins(20000000), start, end)
				case "delayed":
					account, err = vestingtypes.NewDelayedVestingAccount(base, coins(20000000), end)
				case "periodic":
					account, err = vestingtypes.NewPeriodicVestingAccount(base, coins(20000000), start, vestingtypes.Periods{{Length: end - start, Amount: coins(20000000)}})
				case "permanent":
					account, err = vestingtypes.NewPermanentLockedAccount(base, coins(20000000))
				}
				require.NoError(t, err)
				bApp.AccountKeeper.SetAccount(ctx, account)
				server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
				msg, err := stakingtypes.NewMsgCreateValidator(self.String(), ed25519.GenPrivKey().PubKey(), coins(10000000)[0],
					stakingtypes.NewDescription("retained-vesting", "", "", "", ""),
					stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(1, 1), math.LegacyNewDecWithPrec(2, 1), math.LegacyNewDecWithPrec(1, 2)), math.OneInt())
				require.NoError(t, err)
				_, err = server.CreateValidator(ctx, msg)
				require.NoError(t, err)
				_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
				require.NoError(t, err)
				delegate(t, bApp, ctx, server, denom, operator, other, math.NewInt(2000000))
				// The other signing address belongs to this same player but must
				// still migrate, even though its operator account is retained.
				secondary := sdk.AccAddress(bytes.Repeat([]byte{0x67}, 20))
				index := bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, operator.String())
				require.NotZero(t, index)
				require.NoError(t, bApp.StructsKeeper.SetPlayerIndexForAddress(ctx, secondary.String(), index))
				delegate(t, bApp, ctx, server, denom, secondary, other, math.NewInt(3000000))
				fundRedelegationAccount(t, bApp, ctx, secondary, coins(123))
				fundRedelegationAccount(t, bApp, ctx, operator, sdk.NewCoins(sdk.NewInt64Coin("utest", 77)))
				withdraw := sdk.AccAddress(bytes.Repeat([]byte{0x68}, 20))
				require.NoError(t, bApp.DistrKeeper.SetDelegatorWithdrawAddr(ctx, operator, withdraw))
				var red stakingtypes.Redelegation
				var ubd stakingtypes.UnbondingDelegation
				if position == "held_redelegation_only" || position == "redelegation_with_self_unbonding" {
					_, err = server.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(operator.String(), other.String(), self.String(), coins(1000000)[0]))
					require.NoError(t, err)
					red, err = bApp.StakingKeeper.GetRedelegation(ctx, operator, other, self)
					require.NoError(t, err)
					require.NoError(t, bApp.StakingKeeper.PutUnbondingOnHold(ctx, red.Entries[0].UnbondingId))
					red, err = bApp.StakingKeeper.GetRedelegation(ctx, operator, other, self)
					require.NoError(t, err)
				}
				if position != "bonded" {
					d, err := bApp.StakingKeeper.GetDelegation(ctx, operator, self)
					require.NoError(t, err)
					_, _, err = bApp.StakingKeeper.Undelegate(ctx, operator, self, d.Shares)
					require.NoError(t, err)
					ubd, err = bApp.StakingKeeper.GetUnbondingDelegation(ctx, operator, self)
					require.NoError(t, err)
					if position == "held_redelegation_only" {
						ctx = ctx.WithBlockTime(ubd.Entries[0].CompletionTime.Add(time.Second))
						_, err = bApp.StakingKeeper.CompleteUnbonding(ctx, operator, self)
						require.NoError(t, err)
						_, err = bApp.StakingKeeper.GetUnbondingDelegation(ctx, operator, self)
						require.ErrorIs(t, err, stakingtypes.ErrNoUnbondingDelegation)
					}
					_, err = bApp.StakingKeeper.GetDelegation(ctx, operator, self)
					require.ErrorIs(t, err, stakingtypes.ErrNoDelegation)
				}
				beforeAccount := bApp.AccountKeeper.GetAccount(ctx, operator)
				var beforeSelf stakingtypes.Delegation
				if position == "bonded" {
					beforeSelf, err = bApp.StakingKeeper.GetDelegation(ctx, operator, self)
					require.NoError(t, err)
				}
				beforeBalances := bApp.BankKeeper.GetAllBalances(ctx, operator)
				beforeSpendable := bApp.BankKeeper.SpendableCoins(ctx, operator)
				beforeStake, err := bApp.StakingKeeper.GetDelegation(ctx, operator, other)
				require.NoError(t, err)
				beforeInfo, err := bApp.DistrKeeper.GetDelegatorStartingInfo(ctx, other, operator)
				require.NoError(t, err)
				secondaryStake, err := bApp.StakingKeeper.GetDelegation(ctx, secondary, other)
				require.NoError(t, err)
				player, err := bApp.StructsKeeper.NewCurrentContext(ctx).GetPlayerByAddress(operator.String())
				require.NoError(t, err)
				vault, err := types.VaultAddress(player.ID())
				require.NoError(t, err)
				keepers := delegationUpgradeKeepers(bApp)
				movements, err := v22.BuildVaultAddressMigrationPlan(ctx, keepers)
				require.NoError(t, err)
				for _, move := range movements {
					require.NotEqual(t, operator.String(), move.Source)
				}
				ctx = ctx.WithEventManager(sdk.NewEventManager())
				require.NoError(t, v22.MigrateVaultAssets(ctx, keepers))
				require.Equal(t, beforeAccount, bApp.AccountKeeper.GetAccount(ctx, operator))
				require.Equal(t, beforeBalances, bApp.BankKeeper.GetAllBalances(ctx, operator))
				require.Equal(t, beforeSpendable, bApp.BankKeeper.SpendableCoins(ctx, operator))
				if position == "bonded" {
					afterSelf, err := bApp.StakingKeeper.GetDelegation(ctx, operator, self)
					require.NoError(t, err)
					require.Equal(t, beforeSelf, afterSelf)
				}
				require.Equal(t, index, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, operator.String()))
				afterStake, err := bApp.StakingKeeper.GetDelegation(ctx, operator, other)
				require.NoError(t, err)
				require.Equal(t, beforeStake, afterStake, "non-self stake must remain with the same vesting counters")
				afterInfo, err := bApp.DistrKeeper.GetDelegatorStartingInfo(ctx, other, operator)
				require.NoError(t, err)
				require.Equal(t, beforeInfo, afterInfo)
				afterWithdraw, err := bApp.DistrKeeper.GetDelegatorWithdrawAddr(ctx, operator)
				require.NoError(t, err)
				require.Equal(t, withdraw, afterWithdraw)
				_, err = bApp.StakingKeeper.GetDelegation(ctx, vault, other)
				require.ErrorIs(t, err, stakingtypes.ErrNoDelegation)
				moved, err := bApp.StakingKeeper.GetDelegation(ctx, secondary, other)
				require.NoError(t, err)
				require.Equal(t, secondaryStake.Shares, moved.Shares)
				require.True(t, bApp.BankKeeper.GetAllBalances(ctx, secondary).IsZero())
				require.Equal(t, coins(123), bApp.BankKeeper.GetAllBalances(ctx, vault))
				for _, event := range migrationEvents[*types.EventVaultDelegationMigration](t, ctx, "EventVaultDelegationMigration") {
					require.NotEqual(t, operator.String(), event.SourceAddress)
				}
				for _, event := range migrationEvents[*types.EventVaultVestingMigration](t, ctx, "EventVaultVestingMigration") {
					require.NotContains(t, event.SourceAddresses, operator.String())
				}
				for _, event := range ctx.EventManager().Events() {
					if event.Type == "transfer" {
						for _, attribute := range event.Attributes {
							if attribute.Key == "sender" {
								require.NotEqual(t, operator.String(), attribute.Value, "no bank movement may be reported from a retained account")
							}
						}
					}
				}
				if position == "self_unbonding" || position == "redelegation_with_self_unbonding" {
					after, err := bApp.StakingKeeper.GetUnbondingDelegationByUnbondingID(ctx, ubd.Entries[0].UnbondingId)
					require.NoError(t, err)
					require.Equal(t, ubd, after)
					queue, err := bApp.StakingKeeper.GetUBDQueueTimeSlice(ctx, ubd.Entries[0].CompletionTime)
					require.NoError(t, err)
					require.Contains(t, queue, stakingtypes.DVPair{DelegatorAddress: operator.String(), ValidatorAddress: self.String()})
				}
				if position == "held_redelegation_only" || position == "redelegation_with_self_unbonding" {
					after, err := bApp.StakingKeeper.GetRedelegationByUnbondingID(ctx, red.Entries[0].UnbondingId)
					require.NoError(t, err)
					require.Equal(t, red, after)
					queue, err := bApp.StakingKeeper.GetRedelegationQueueTimeSlice(ctx, red.Entries[0].CompletionTime)
					require.NoError(t, err)
					require.Contains(t, queue, stakingtypes.DVVTriplet{DelegatorAddress: operator.String(), ValidatorSrcAddress: other.String(), ValidatorDstAddress: self.String()})
				}
				if position == "redelegation_with_self_unbonding" {
					// Slashing must still find pending self-stake at the record's
					// native address, without rewriting the account's counters.
					preview, _ := ctx.CacheContext()
					validator, err := bApp.StakingKeeper.GetValidator(preview, other)
					require.NoError(t, err)
					supply := bApp.BankKeeper.GetSupply(preview, denom).Amount
					slashed, err := bApp.StakingKeeper.SlashRedelegation(preview, validator, red, red.Entries[0].CreationHeight, math.LegacyNewDecWithPrec(1, 1))
					require.NoError(t, err)
					require.Equal(t, math.NewInt(100000), slashed)
					pending, err := bApp.StakingKeeper.GetUnbondingDelegationByUnbondingID(preview, ubd.Entries[0].UnbondingId)
					require.NoError(t, err)
					require.Equal(t, ubd.Entries[0].Balance.Sub(slashed), pending.Entries[0].Balance)
					require.Equal(t, slashed, supply.Sub(bApp.BankKeeper.GetSupply(preview, denom).Amount))
					require.Equal(t, beforeAccount, bApp.AccountKeeper.GetAccount(preview, operator))
					requireStakingAndDistributionHealthy(t, bApp, preview)
				}
				// Re-running the migration must not flatten the retained account.
				require.NoError(t, v22.MigrateVaultAssets(ctx, keepers))
				require.Equal(t, beforeAccount, bApp.AccountKeeper.GetAccount(ctx, operator))
				_, err = bApp.DistrKeeper.WithdrawDelegationRewards(ctx, operator, other)
				require.NoError(t, err)
				// No pending operation protects this non-self delegation yet.
				// Rotation must still preserve its vesting counters and shares.
				rotation, _ := ctx.CacheContext()
				_, err = keeper.NewMsgServerImpl(bApp.StructsKeeper).PlayerUpdatePrimaryAddress(rotation,
					&types.MsgPlayerUpdatePrimaryAddress{Creator: operator.String(), PrimaryAddress: secondary.String()})
				require.NoError(t, err)
				rotationStake, err := bApp.StakingKeeper.GetDelegation(rotation, operator, other)
				require.NoError(t, err)
				require.Equal(t, afterStake, rotationStake)
				require.Equal(t, bApp.AccountKeeper.GetAccount(ctx, operator), bApp.AccountKeeper.GetAccount(rotation, operator))
				_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(operator.String(), other.String(), coins(100000)[0]))
				require.NoError(t, err, "retained native vesting stake must remain accessible")
				pending, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, operator, other)
				require.NoError(t, err)
				preview, _ := ctx.CacheContext()
				preview = preview.WithBlockTime(pending.Entries[0].CompletionTime.Add(time.Second))
				old := bApp.AccountKeeper.GetAccount(preview, operator).(vestingexported.VestingAccount)
				old.TrackUndelegation(coins(100000))
				paid, err := bApp.StakingKeeper.CompleteUnbonding(preview, operator, other)
				require.NoError(t, err)
				require.Equal(t, coins(100000), paid)
				afterRefund := bApp.AccountKeeper.GetAccount(preview, operator).(vestingexported.VestingAccount)
				require.Equal(t, old.GetDelegatedVesting(), afterRefund.GetDelegatedVesting())
				require.Equal(t, old.GetDelegatedFree(), afterRefund.GetDelegatedFree())
				require.Equal(t, old.LockedCoins(preview.BlockTime()), afterRefund.LockedCoins(preview.BlockTime()))
				requireStakingAndDistributionHealthy(t, bApp, preview)

				// Native obligations stay at this account through identity changes.
				// Rotation cannot rekey its non-self stake, and pending operations
				// cannot prevent emergency revocation of its player authority.
				identity := bApp.AccountKeeper.GetAccount(ctx, operator)
				otherStake, err := bApp.StakingKeeper.GetDelegation(ctx, operator, other)
				require.NoError(t, err)
				messages := keeper.NewMsgServerImpl(bApp.StructsKeeper)
				_, err = messages.PlayerUpdatePrimaryAddress(ctx, &types.MsgPlayerUpdatePrimaryAddress{Creator: operator.String(), PrimaryAddress: secondary.String()})
				require.NoError(t, err)
				require.Equal(t, identity, bApp.AccountKeeper.GetAccount(ctx, operator))
				_, err = messages.AddressRevoke(ctx, &types.MsgAddressRevoke{Creator: secondary.String(), Address: operator.String()})
				require.NoError(t, err)
				require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, operator.String()))
				retainedStake, err := bApp.StakingKeeper.GetDelegation(ctx, operator, other)
				require.NoError(t, err)
				require.Equal(t, otherStake, retainedStake)
				require.Equal(t, identity, bApp.AccountKeeper.GetAccount(ctx, operator))
				otherReactor := jailGateReactorId(t, bApp, ctx, other)
				infusion, found := bApp.StructsKeeper.GetInfusion(ctx, otherReactor, operator.String())
				if found {
					require.Zero(t, infusion.Power, "revocation cannot leave capacity credited to the old owner")
				}
				newPrimary, _, newOwner := newTransferPlayer(t, bApp, ctx)
				nonce := bApp.StructsKeeper.NewCurrentContext(ctx).GetAddressProofNonce(operator.String())
				proof, err := operatorKey.Sign([]byte(types.AddressRegisterProofInput(ctx.ChainID(), newOwner, operator.String(), nonce)))
				require.NoError(t, err)
				_, err = messages.AddressRegister(ctx, &types.MsgAddressRegister{Creator: newPrimary.String(), PlayerId: newOwner, Address: operator.String(),
					ProofPubKey: hex.EncodeToString(operatorKey.PubKey().Bytes()), ProofSignature: hex.EncodeToString(proof), Permissions: uint64(types.PermAll)})
				require.NoError(t, err)
				require.Equal(t, identity, bApp.AccountKeeper.GetAccount(ctx, operator), "registration must not transfer vesting obligations")
				retainedStake, err = bApp.StakingKeeper.GetDelegation(ctx, operator, other)
				require.NoError(t, err)
				require.Equal(t, otherStake, retainedStake)
				infusion, found = bApp.StructsKeeper.GetInfusion(ctx, otherReactor, operator.String())
				require.True(t, found)
				require.Equal(t, newOwner, infusion.PlayerId)
				requireStakingAndDistributionHealthy(t, bApp, ctx)
			})
		}
	}
}
