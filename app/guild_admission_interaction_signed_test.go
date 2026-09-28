package app_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	"structs/x/structs/types"
)

// Existing consent routes remain usable with no guild tokens. Purchase admission
// is independent and opt-in, and consuming an old proposal cannot leave consent
// that re-admits a player after a later kick or resets a purchased rank.
func TestSignedRankAdmissionAndExistingConsentRoutes(t *testing.T) {
	for _, route := range []string{"invite", "request"} {
		for _, admission := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/purchase=%t", route, admission), func(t *testing.T) {
				bApp, initial := setupReactorAppGenesis(t, 1, true)
				k := bApp.StructsKeeper
				ownerKey, buyerKey := secp256k1.GenPrivKey(), secp256k1.GenPrivKey()
				ownerAddress, buyerAddress := sdk.AccAddress(ownerKey.PubKey().Address()), sdk.AccAddress(buyerKey.PubKey().Address())
				owner := appendAppPlayer(t, k, initial, ownerAddress)
				buyer := appendAppPlayer(t, k, initial, buyerAddress)
				validators, err := bApp.StakingKeeper.GetLastValidators(initial)
				require.NoError(t, err)
				validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
				require.NoError(t, err)
				guild := types.CreateEmptyGuild()
				guild.Id, guild.Index, guild.Owner = "0-988", 988, owner.Id
				guild.PrimaryReactorId = jailGateReactorId(t, bApp, initial, validator)
				guild.EntrySubstationId, guild.EntryRank = "4-988", 5
				guild.JoinInfusionMinimum = 1000
				guild.JoinInfusionMinimumBypassByInvite = types.GuildJoinBypassLevel_permissioned
				guild.JoinInfusionMinimumBypassByRequest = types.GuildJoinBypassLevel_permissioned
				guild.RankPurchaseMinimumRank, guild.RankPurchaseBasePrice = 1, 1
				guild.JoinByRankPurchaseEnabled = admission
				k.SetGuild(initial, guild)
				k.SetSubstation(initial, types.Substation{Id: guild.EntrySubstationId, Owner: guild.Id})
				vaults, err := k.NewCurrentContext(initial).EnsureVaults(buyer.Id, guild.Id)
				require.NoError(t, err)
				buyerVault, treasury := vaults[0], vaults[1]
				denom := "uguild." + guild.Id
				fundRedelegationAccount(t, bApp, initial, treasury, sdk.NewCoins(sdk.NewInt64Coin(denom, 10)))
				ctx, height := initial, int64(0)
				config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
				deliver := func(key cryptotypes.PrivKey, message sdk.Msg) *abci.ExecTxResult {
					t.Helper()
					account := bApp.AccountKeeper.GetAccount(ctx, sdk.AccAddress(key.PubKey().Address()))
					if height > 0 {
						_, err := bApp.Commit()
						require.NoError(t, err)
					}
					height++
					tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(height)), config, []sdk.Msg{message}, nil,
						ante.DefaultFreeGasCap, jailGateChainID, []uint64{account.GetAccountNumber()}, []uint64{account.GetSequence()}, key)
					require.NoError(t, err)
					wire, err := config.TxEncoder()(tx)
					require.NoError(t, err)
					when := initial.BlockTime().Add(time.Duration(height-1) * time.Second)
					block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: when, Txs: [][]byte{wire}})
					require.NoError(t, err)
					require.Len(t, block.TxResults, 1)
					ctx = bApp.NewContext(false).WithBlockTime(when)
					result := block.TxResults[0]
					require.NotContains(t, result.Log, "panic")
					require.Equal(t, int64(10), bApp.BankKeeper.GetSupply(ctx, denom).Amount.Int64())
					return result
				}
				refused := func(result *abci.ExecTxResult, reason string) {
					t.Helper()
					require.NotZero(t, result.Code)
					require.Contains(t, result.Log, reason)
					for _, event := range result.Events {
						require.False(t, strings.HasPrefix(event.Type, "structs.structs."))
						require.NotContains(t, []string{"transfer", "coin_spent", "coin_received"}, event.Type)
					}
				}
				purchase := &types.MsgGuildRankPurchase{Creator: buyerAddress.String(), GuildId: guild.Id, ExpectedRank: 5, TargetRank: 3, MaxPrice: 2}
				join := &types.MsgGuildMembershipJoinByRankPurchase{Creator: buyerAddress.String(), GuildId: guild.Id, TargetRank: 3, MaxPrice: 2}
				refused(deliver(buyerKey, purchase), "not a member")
				refused(deliver(buyerKey, &types.MsgGuildMembershipJoin{Creator: buyerAddress.String(), PlayerId: buyer.Id, GuildId: guild.Id}), "join infusion minimum not met")
				if !admission {
					refused(deliver(buyerKey, join), "rank_purchase_admission_disabled")
				}
				var propose, approve, wrong sdk.Msg
				var proposeKey, approveKey, wrongKey cryptotypes.PrivKey
				if route == "invite" {
					propose = &types.MsgGuildMembershipInvite{Creator: ownerAddress.String(), GuildId: guild.Id, PlayerId: buyer.Id}
					approve = &types.MsgGuildMembershipInviteApprove{Creator: buyerAddress.String(), GuildId: guild.Id, PlayerId: buyer.Id}
					wrong = &types.MsgGuildMembershipRequestApprove{Creator: ownerAddress.String(), GuildId: guild.Id, PlayerId: buyer.Id}
					proposeKey, approveKey, wrongKey = ownerKey, buyerKey, ownerKey
				} else {
					propose = &types.MsgGuildMembershipRequest{Creator: buyerAddress.String(), GuildId: guild.Id, PlayerId: buyer.Id}
					approve = &types.MsgGuildMembershipRequestApprove{Creator: ownerAddress.String(), GuildId: guild.Id, PlayerId: buyer.Id}
					wrong = &types.MsgGuildMembershipInviteApprove{Creator: buyerAddress.String(), GuildId: guild.Id, PlayerId: buyer.Id}
					proposeKey, approveKey, wrongKey = buyerKey, ownerKey, buyerKey
				}
				result := deliver(proposeKey, propose)
				require.Zero(t, result.Code, result.Log)
				proposal, found := k.GetGuildMembershipApplication(ctx, guild.Id, buyer.Id)
				require.True(t, found)
				refused(deliver(buyerKey, purchase), "not a member")
				refused(deliver(wrongKey, wrong), "join_type_mismatch")
				unchanged, found := k.GetGuildMembershipApplication(ctx, guild.Id, buyer.Id)
				require.True(t, found)
				require.Equal(t, proposal, unchanged)
				assertApplicationConsumed := func(result *abci.ExecTxResult) {
					t.Helper()
					_, found := k.GetGuildMembershipApplication(ctx, guild.Id, buyer.Id)
					require.False(t, found)
					expected := proposal
					expected.RegistrationStatus = types.RegistrationStatus_approved
					count := 0
					for _, event := range result.Events {
						if event.Type == "structs.structs.EventGuildMembershipApplication" {
							decoded, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							require.Equal(t, expected, *decoded.(*types.EventGuildMembershipApplication).GuildMembershipApplication)
							count++
						}
					}
					require.Equal(t, 1, count)
				}
				if !admission {
					result = deliver(approveKey, approve)
					require.Zero(t, result.Code, result.Log)
					assertApplicationConsumed(result)
					member, found := k.GetPlayer(ctx, buyer.Id)
					require.True(t, found)
					require.Equal(t, guild.Id, member.GuildId)
					require.Equal(t, guild.EntryRank, member.GuildRank)
					require.True(t, bApp.BankKeeper.GetBalance(ctx, buyerVault, denom).IsZero(), "legacy consent requires no guild tokens")
					require.Equal(t, int64(10), bApp.BankKeeper.GetBalance(ctx, treasury, denom).Amount.Int64())
				}
				result = deliver(ownerKey, &types.MsgVaultSend{Creator: ownerAddress.String(), OwnerId: guild.Id, RecipientOwnerId: buyer.Id, Amount: sdk.NewCoins(sdk.NewInt64Coin(denom, 10))})
				require.Zero(t, result.Code, result.Log)
				if admission {
					result = deliver(buyerKey, join)
					require.Zero(t, result.Code, result.Log)
					assertApplicationConsumed(result)
				} else {
					result = deliver(buyerKey, purchase)
					require.Zero(t, result.Code, result.Log)
				}
				purchaseCount := 0
				for _, event := range result.Events {
					if event.Type == "structs.structs.EventGuildRankPurchase" {
						decoded, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						previous := uint64(5)
						if admission {
							previous = buyer.GuildRank
						}
						require.Equal(t, &types.EventGuildRankPurchase{GuildId: guild.Id, PlayerId: buyer.Id, PreviousRank: previous, Rank: 3, Price: 2, Denom: denom, Joined: admission}, decoded)
						purchaseCount++
					}
				}
				require.Equal(t, 1, purchaseCount)
				refused(deliver(approveKey, approve), "already a member")
				member, found := k.GetPlayer(ctx, buyer.Id)
				require.True(t, found)
				require.Equal(t, uint64(3), member.GuildRank, "replayed approval cannot reset a purchased rank")
				require.Equal(t, guild.EntrySubstationId, member.SubstationId)
				result = deliver(ownerKey, &types.MsgGuildMembershipKick{Creator: ownerAddress.String(), GuildId: guild.Id, PlayerId: buyer.Id})
				require.Zero(t, result.Code, result.Log)
				refused(deliver(approveKey, approve), "no application on file")
				member, found = k.GetPlayer(ctx, buyer.Id)
				require.True(t, found)
				require.Empty(t, member.GuildId, "consumed consent cannot re-admit a kicked player")
				require.Equal(t, int64(8), bApp.BankKeeper.GetBalance(ctx, buyerVault, denom).Amount.Int64())
				require.Equal(t, int64(2), bApp.BankKeeper.GetBalance(ctx, treasury, denom).Amount.Int64(), "departure does not refund purchased ranks")
				_, err = bApp.Commit()
				require.NoError(t, err)
			})
		}
	}
}
