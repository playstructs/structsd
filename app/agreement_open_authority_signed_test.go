package app_test

import (
	"fmt"
	"math/rand"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stretchr/testify/require"

	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Access to a provider and authority to spend a selected owner's vault are
// independent. Exercise both gates through signed execution with real bank
// balances and ordinary collateral settlement, including rollback after a first open.
func TestSignedAgreementOpenSeparatesMarketAndTreasuryAuthority(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		for _, scenario := range []string{"owner_open", "delegate_open", "market_only", "treasury_only", "both_grants", "no_transfer_bit", "no_provider_bit", "closed_market", "unknown_policy", "missing_owner", "later_failure", "insufficient_funds"} {
			t.Run(fmt.Sprintf("guild=%t/%s", guildOwned, scenario), func(t *testing.T) {
				bApp, ctx, providerID, _, _ := setupFundedAgreements(t, 0)
				ownerKey, delegateKey := secp256k1.GenPrivKey(), secp256k1.GenPrivKey()
				ownerAddress := sdk.AccAddress(ownerKey.PubKey().Address())
				delegateAddress := sdk.AccAddress(delegateKey.PubKey().Address())
				owner := appendAppPlayer(t, bApp.StructsKeeper, ctx, ownerAddress)
				delegate := appendAppPlayer(t, bApp.StructsKeeper, ctx, delegateAddress)
				ownerID := owner.Id
				if guildOwned {
					guild := types.CreateEmptyGuild()
					guild.Id, guild.Index, guild.Owner, guild.Creator = "0-951", 951, owner.Id, owner.Creator
					bApp.StructsKeeper.SetGuild(ctx, guild)
					ownerID = guild.Id
				}
				vault, err := types.VaultAddress(ownerID)
				require.NoError(t, err)
				initial := int64(2100) // capacity 10 * duration 70 * rate 3
				if scenario == "insufficient_funds" {
					initial--
				}
				fundRedelegationAccount(t, bApp, ctx, vault, sdk.NewCoins(sdk.NewInt64Coin("ualpha", initial)))
				for _, address := range []sdk.AccAddress{ownerAddress, delegateAddress} {
					fundRedelegationAccount(t, bApp, ctx, address, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 73)))
				}
				_, associated := bApp.StructsKeeper.GetVaultOwner(ctx, vault)
				require.False(t, associated)
				key, address := ownerKey, ownerAddress
				bits := types.PermTokenTransfer
				wantError := ""
				provider, found := bApp.StructsKeeper.GetProvider(ctx, providerID)
				require.True(t, found)
				switch scenario {
				case "delegate_open":
					key, address = delegateKey, delegateAddress
					bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(ownerID, delegate.Id), types.PermTokenTransfer)
				case "market_only", "treasury_only", "both_grants", "no_transfer_bit", "no_provider_bit":
					key, address = delegateKey, delegateAddress
					bits = types.PermTokenTransfer | types.PermProviderOpen
					provider.AccessPolicy = types.ProviderAccessPolicy_guildMarket
					if scenario != "treasury_only" {
						bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(providerID, delegate.Id), types.PermProviderOpen)
					}
					if scenario != "market_only" {
						bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(ownerID, delegate.Id), types.PermTokenTransfer)
					}
					if scenario != "both_grants" {
						wantError = "permission"
					}
					if scenario == "no_transfer_bit" {
						bits = types.PermProviderOpen
					}
					if scenario == "no_provider_bit" {
						bits = types.PermTokenTransfer
					}
				case "closed_market":
					provider.AccessPolicy, wantError = types.ProviderAccessPolicy_closedMarket, "closed_market"
				case "unknown_policy":
					// Simulate an existing invalid enum row, bypassing the setter's
					// input validation to exercise the read-side deny default.
					provider.AccessPolicy, wantError = types.ProviderAccessPolicy(99), "unknown"
				case "missing_owner":
					wantError = "not found"
				case "later_failure":
					wantError = "message index: 1"
				case "insufficient_funds":
					wantError = "afford"
				}
				_, err = bApp.StructsKeeper.SetProvider(ctx, provider)
				require.NoError(t, err)
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), bits)
				msg := &types.MsgAgreementOpen{Creator: address.String(), OwnerId: ownerID, ProviderId: providerID, Capacity: 10, Duration: 70}
				if scenario == "missing_owner" {
					msg.OwnerId = "0-999999"
				}
				messages := []sdk.Msg{msg}
				if scenario == "later_failure" {
					messages = append(messages, &types.MsgAgreementOpen{Creator: address.String(), OwnerId: ownerID, ProviderId: providerID, Capacity: 10, Duration: 70})
				}
				before := bApp.StructsKeeper.GetAllAgreement(ctx)
				allocationCount := bApp.StructsKeeper.GetAllocationCount(ctx)
				permissionsBefore := bApp.StructsKeeper.GetAllPermissionExport(ctx)
				signer := bApp.AccountKeeper.GetAccount(ctx, address)
				config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
				tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages, nil,
					ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
				require.NoError(t, err)
				wire, err := config.TxEncoder()(tx)
				require.NoError(t, err)
				block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
				require.NoError(t, err)
				require.Len(t, block.TxResults, 1)
				result := block.TxResults[0]
				current := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
				ids := bApp.StructsKeeper.GetAllAgreementIdByProviderIndex(current, providerID)
				associatedOwner, associated := bApp.StructsKeeper.GetVaultOwner(current, vault)
				pool := keeperlib.GetProviderCollateralPoolLocation(providerID)
				for _, signingAddress := range []sdk.AccAddress{ownerAddress, delegateAddress} {
					require.Equal(t, "73", bApp.BankKeeper.GetBalance(current, signingAddress, "ualpha").Amount.String(), "only the selected vault may fund collateral")
				}
				if wantError != "" {
					require.NotZero(t, result.Code)
					require.Contains(t, result.Log, wantError)
					require.NotContains(t, result.Log, "panic")
					require.Empty(t, ids)
					require.Equal(t, before, bApp.StructsKeeper.GetAllAgreement(current))
					require.Equal(t, allocationCount, bApp.StructsKeeper.GetAllocationCount(current))
					require.Equal(t, permissionsBefore, bApp.StructsKeeper.GetAllPermissionExport(current))
					require.False(t, associated)
					require.Equal(t, initial, bApp.BankKeeper.GetBalance(current, vault, "ualpha").Amount.Int64())
					require.True(t, bApp.BankKeeper.GetBalance(current, pool, "ualpha").IsZero())
					require.Zero(t, bApp.StructsKeeper.NewCurrentContext(current).GetProvider(providerID).GetAgreementLoad())
					for _, event := range result.Events {
						require.NotContains(t, []string{"coin_spent", "coin_received", "transfer", "structs.structs.EventAgreement", "structs.structs.EventServiceObligation", "structs.structs.EventVaultAddress"}, event.Type)
					}
				} else {
					require.Zero(t, result.Code, result.Log)
					require.Less(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
					t.Logf("signed AgreementOpen: %d gas (cap %d)", result.GasUsed, ante.DefaultFreeGasCap)
					require.Len(t, ids, 1)
					require.True(t, associated)
					require.Equal(t, ownerID, associatedOwner)
					require.True(t, bApp.BankKeeper.GetBalance(current, vault, "ualpha").IsZero())
					require.Equal(t, "2100", bApp.BankKeeper.GetBalance(current, pool, "ualpha").Amount.String())
					cc := bApp.StructsKeeper.NewCurrentContext(current)
					agreement, exists := bApp.StructsKeeper.GetAgreement(current, ids[0])
					require.True(t, exists)
					require.Equal(t, ownerID, agreement.Owner)
					require.Equal(t, address.String(), agreement.Creator)
					require.Equal(t, uint64(1), agreement.StartBlock)
					require.Equal(t, uint64(71), agreement.EndBlock)
					require.Contains(t, bApp.StructsKeeper.GetAllAgreementIdByExpirationIndex(current, 71), agreement.Id)
					allocation, exists := cc.GetAllocation(agreement.AllocationId)
					require.True(t, exists)
					require.Equal(t, ownerID, allocation.GetOwnerId())
					require.Equal(t, uint64(10), allocation.GetPower())
					for _, subject := range []string{owner.Id, delegate.Id, ownerID} {
						wantAgreement, wantAllocation := types.Permissionless, types.Permissionless
						if !guildOwned && subject == ownerID {
							wantAgreement, wantAllocation = types.PermAgreementAll, types.PermAllocationConnection
						}
						require.Equal(t, wantAgreement, cc.GetPermissions(keeperlib.GetObjectPermissionIDBytes(agreement.Id, subject)))
						require.Equal(t, wantAllocation, cc.GetPermissions(keeperlib.GetObjectPermissionIDBytes(allocation.ID(), subject)))
					}
					require.Equal(t, uint64(10), cc.GetProvider(providerID).GetAgreementLoad())
					transfers := 0
					for _, event := range result.Events {
						if event.Type == "transfer" {
							attributes := map[string]string{}
							for _, attribute := range event.Attributes {
								attributes[attribute.Key] = attribute.Value
							}
							require.Equal(t, vault.String(), attributes["sender"])
							require.Equal(t, pool.String(), attributes["recipient"])
							require.Equal(t, "2100ualpha", attributes["amount"])
							transfers++
						}
					}
					require.Equal(t, 1, transfers)
				}
				message, broken := keeperlib.ProviderCollateralSolvencyInvariant(bApp.StructsKeeper)(current)
				require.False(t, broken, message)
			})
		}
	}
}
