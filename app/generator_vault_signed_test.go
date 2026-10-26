package app_test

import (
	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"math/rand"
	"structs/app"
	"testing"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	v22 "structs/app/upgrades/v0_22_0"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestSignedGeneratorVaultBurnAndBatchRollback(t *testing.T) {
	for _, scenario := range []struct {
		name                      string
		rollback, legacy, migrate bool
	}{
		{"success", false, false, false}, {"later_message_failure", true, false, false},
		{"restricted_key", false, false, false},
		{"legacy_success", false, true, false}, {"legacy_later_message_failure", true, true, false},
		{"migrated_legacy_success", false, true, true}, {"migrated_legacy_later_message_failure", true, true, true},
	} {
		name, rollback := scenario.name, scenario.rollback
		t.Run(name, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			key := secp256k1.GenPrivKey()
			address := sdk.AccAddress(key.PubKey().Address())
			actor := appendAppPlayer(t, bApp.StructsKeeper, ctx, address)
			vault, err := types.VaultAddress(actor.Id)
			require.NoError(t, err)
			// Bank funding creates the auth account, but not a Structs vault association.
			fundRedelegationAccount(t, bApp, ctx, vault, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100)))
			fundRedelegationAccount(t, bApp, ctx, address, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 73)))
			_, found := bApp.StructsKeeper.GetVaultOwner(ctx, vault)
			require.False(t, found)
			planet := types.Planet{Id: "2-991", Owner: actor.Id, Creator: address.String()}
			bApp.StructsKeeper.SetPlanet(ctx, planet)
			generatorType, found := bApp.StructsKeeper.GetStructType(ctx, 20)
			require.True(t, found)
			require.Equal(t, uint64(2), generatorType.GeneratingRate)
			bApp.StructsKeeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, actor.Id), generatorType.PassiveDraw)
			bApp.StructsKeeper.SetStructAttribute(ctx, keeperlib.GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, actor.Id, 20), 1)
			generator := types.Struct{Id: "5-991", Owner: actor.Id, Creator: address.String(), Type: 20, LocationId: planet.Id, LocationType: types.ObjectType_planet}
			bApp.StructsKeeper.SetStruct(ctx, generator)
			statusID := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, generator.Id)
			bApp.StructsKeeper.SetStructAttribute(ctx, statusID, uint64(types.StructStateMaterialized|types.StructStateBuilt|types.StructStateOnline))
			var legacyBefore types.Infusion
			if scenario.legacy {
				// Seed already-burned historical fuel at the former primary key.
				// This is a state fixture, not a second mint or a migration invocation.
				cc := bApp.StructsKeeper.NewCurrentContext(ctx)
				position := cc.UpsertInfusion(types.ObjectType_struct, generator.Id, address.String(), actor.Id)
				position.SetRatio(2)
				position.AddFuel(7)
				cc.CommitAll()
				legacyBefore, found = bApp.StructsKeeper.GetInfusion(ctx, generator.Id, address.String())
				require.True(t, found)
				require.Equal(t, uint64(14), legacyBefore.Power)
			}
			vaultBefore, primaryBefore := int64(100), int64(73)
			if scenario.migrate {
				require.Empty(t, bApp.StructsKeeper.GetGridCascadeQueue(ctx, false))
				handler := v22.CreateUpgradeHandler(bApp.ModuleManager, bApp.Configurator(), delegationUpgradeKeepers(bApp))
				_, err := handler(ctx, upgradetypes.Plan{Name: v22.UpgradeName}, bApp.ModuleManager.GetVersionMap())
				require.NoError(t, err)
				vaultBefore, primaryBefore = 173, 0
				require.Equal(t, vaultBefore, bApp.BankKeeper.GetBalance(ctx, vault, "ualpha").Amount.Int64())
				require.True(t, bApp.BankKeeper.GetBalance(ctx, address, "ualpha").IsZero())
				legacyAfter, found := bApp.StructsKeeper.GetInfusion(ctx, generator.Id, address.String())
				require.True(t, found)
				require.Equal(t, legacyBefore, legacyAfter)
				_, found = bApp.StructsKeeper.GetInfusion(ctx, generator.Id, vault.String())
				require.False(t, found, "migration must not duplicate already-burned generator fuel")
				require.Equal(t, uint64(14), redelegationCapacity(t, bApp, ctx, actor.Id))
			}
			if name == "restricted_key" {
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), types.PermAll&^types.PermTokenInfuse)
			}
			messages := []sdk.Msg{&types.MsgStructGeneratorInfuse{Creator: address.String(), StructId: generator.Id, InfuseAmount: "10ualpha"}}
			if rollback {
				messages = append(messages, &types.MsgStructGeneratorInfuse{Creator: address.String(), StructId: generator.Id, InfuseAmount: "1uguild.0-1"})
			}
			// This app fixture has no mint module: only this transaction can change supply.
			supplyBefore := bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount
			moduleAddress := bApp.AccountKeeper.GetModuleAddress(types.ModuleName)
			moduleBefore := bApp.BankKeeper.GetBalance(ctx, moduleAddress, "ualpha").Amount
			capacityBefore := redelegationCapacity(t, bApp, ctx, actor.Id)
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
			finalCtx := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
			require.Equal(t, primaryBefore, bApp.BankKeeper.GetBalance(finalCtx, address, "ualpha").Amount.Int64())
			infusion, infusionFound := bApp.StructsKeeper.GetInfusion(finalCtx, generator.Id, vault.String())
			owner, associated := bApp.StructsKeeper.GetVaultOwner(finalCtx, vault)
			if scenario.legacy {
				legacyAfter, found := bApp.StructsKeeper.GetInfusion(finalCtx, generator.Id, address.String())
				require.True(t, found)
				require.Equal(t, legacyBefore, legacyAfter, "new deposits must not reprice or copy old generator fuel")
			}
			require.Equal(t, moduleBefore, bApp.BankKeeper.GetBalance(finalCtx, moduleAddress, "ualpha").Amount)
			require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(finalCtx, vault.String()))
			if rollback || name == "restricted_key" {
				require.Equal(t, supplyBefore, bApp.BankKeeper.GetSupply(finalCtx, "ualpha").Amount)
				require.Equal(t, capacityBefore, redelegationCapacity(t, bApp, finalCtx, actor.Id))
				require.NotZero(t, result.Code)
				if rollback {
					require.Contains(t, result.Log, "message index: 1")
					require.Contains(t, result.Log, "is not a fuel")
				} else {
					require.Contains(t, result.Log, "permission")
				}
				require.Equal(t, vaultBefore, bApp.BankKeeper.GetBalance(finalCtx, vault, "ualpha").Amount.Int64())
				require.False(t, infusionFound)
				require.Equal(t, scenario.migrate, associated, "only pre-transaction associations survive a failed batch")
				for _, event := range result.Events {
					require.NotContains(t, []string{"coin_spent", "coin_received", "burn", "structs.structs.EventVaultAddress", "structs.structs.EventAlphaInfuse"}, event.Type)
				}
			} else {
				require.Zero(t, result.Code, result.Log)
				require.Equal(t, supplyBefore.SubRaw(10), bApp.BankKeeper.GetSupply(finalCtx, "ualpha").Amount)
				require.Equal(t, capacityBefore+20, redelegationCapacity(t, bApp, finalCtx, actor.Id))
				require.Equal(t, vaultBefore-10, bApp.BankKeeper.GetBalance(finalCtx, vault, "ualpha").Amount.Int64())
				require.True(t, infusionFound)
				require.Equal(t, uint64(10), infusion.Fuel)
				require.Equal(t, uint64(20), infusion.Power)
				require.Equal(t, actor.Id, infusion.AssetOwnerID())
				require.True(t, associated)
				require.Equal(t, actor.Id, owner)
				count := 0
				for _, event := range result.Events {
					if event.Type != "structs.structs.EventAlphaInfuse" {
						continue
					}
					parsed, err := sdk.ParseTypedEvent(event)
					require.NoError(t, err)
					detail := parsed.(*types.EventAlphaInfuse).EventAlphaInfuseDetail
					require.Equal(t, vault.String(), detail.VaultAddress)
					require.Equal(t, address.String(), detail.PrimaryAddress)
					require.Equal(t, uint64(10), detail.Amount)
					count++
				}
				require.Equal(t, 1, count)
				if scenario.migrate {
					testGeneratorExportRestart(t, bApp, finalCtx, actor.Id, generator.Id, address, vault, legacyBefore, infusion)
					return
				}
				if scenario.legacy {
					// Exercise cache teardown against real app state after delivery.
					cc := bApp.StructsKeeper.NewCurrentContext(finalCtx)
					cc.GetStruct(generator.Id).DestroyAndCommit()
					cc.CommitAll()
					_, err = bApp.StructsKeeper.EndBlocker(finalCtx)
					require.NoError(t, err)
					for _, source := range []string{address.String(), vault.String()} {
						_, found := bApp.StructsKeeper.GetInfusion(finalCtx, generator.Id, source)
						require.False(t, found)
					}
					require.Zero(t, redelegationCapacity(t, bApp, finalCtx, actor.Id))
					require.Equal(t, vaultBefore-10, bApp.BankKeeper.GetBalance(finalCtx, vault, "ualpha").Amount.Int64())
					require.Equal(t, supplyBefore.SubRaw(10), bApp.BankKeeper.GetSupply(finalCtx, "ualpha").Amount)
				}
			}
		})
	}
}

// Restart the delivered composed-migration fixture through actual genesis export
// and InitChain; test both height-preserving and zero-height forms.
func testGeneratorExportRestart(t *testing.T, original *app.App, ctx sdk.Context, ownerID, generatorID string, primary, vault sdk.AccAddress, legacy, current types.Infusion) {
	t.Helper()
	vaultAmount := original.BankKeeper.GetBalance(ctx, vault, "ualpha").Amount
	supply := original.BankKeeper.GetSupply(ctx, "ualpha").Amount
	accountNumber := original.AccountKeeper.GetAccount(ctx, vault).GetAccountNumber()
	_, err := original.Commit()
	require.NoError(t, err)
	for _, zeroHeight := range []bool{false, true} {
		name := "preserve-height"
		if zeroHeight {
			name = "zero-height"
		}
		t.Run(name, func(t *testing.T) {
			exported, err := original.ExportAppStateAndValidators(zeroHeight, nil, nil)
			require.NoError(t, err)
			options := simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}
			restarted, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, options, baseapp.SetChainID(jailGateChainID))
			require.NoError(t, err)
			restartTime := ctx.BlockTime().Add(time.Second)
			_, err = restarted.InitChain(&abci.RequestInitChain{AppStateBytes: exported.AppState, ChainId: jailGateChainID, InitialHeight: exported.Height, Time: restartTime, ConsensusParams: &exported.ConsensusParams})
			require.NoError(t, err)
			height := exported.Height
			if height == 0 {
				height = 1
			}
			restoredCtx := restarted.NewContextLegacy(false, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: restartTime})
			for _, expected := range []types.Infusion{legacy, current} {
				actual, found := restarted.StructsKeeper.GetInfusion(restoredCtx, generatorID, expected.Address)
				require.True(t, found)
				require.Equal(t, expected, actual)
			}
			require.Equal(t, uint64(34), redelegationCapacity(t, restarted, restoredCtx, ownerID))
			require.Equal(t, vaultAmount, restarted.BankKeeper.GetBalance(restoredCtx, vault, "ualpha").Amount)
			require.True(t, restarted.BankKeeper.GetBalance(restoredCtx, primary, "ualpha").IsZero())
			require.Equal(t, supply, restarted.BankKeeper.GetSupply(restoredCtx, "ualpha").Amount)
			require.Equal(t, accountNumber, restarted.AccountKeeper.GetAccount(restoredCtx, vault).GetAccountNumber())
			owner, found := restarted.StructsKeeper.GetVaultOwner(restoredCtx, vault)
			require.True(t, found)
			require.Equal(t, ownerID, owner)
			require.Zero(t, restarted.StructsKeeper.GetPlayerIndexFromAddress(restoredCtx, vault.String()))
			cc := restarted.StructsKeeper.NewCurrentContext(restoredCtx)
			structure := cc.GetStruct(generatorID)
			require.NoError(t, structure.CheckStruct())
			require.False(t, structure.IsDestroyed())
			require.Equal(t, uint64(20), structure.GetTypeId())
			require.True(t, structure.GetStructType().HasPowerGenerationSystem())
			require.Len(t, cc.GetAllInfusionByDestination(generatorID), 2)
			structure.DestroyAndCommit()
			cc.CommitAll()
			_, queued, queueErr := restarted.StructsKeeper.GetGeneratorPowerSnapshot(restoredCtx, generatorID)
			require.NoError(t, queueErr)
			require.True(t, queued)
			restarted.StructsKeeper.ProcessGeneratorPowerQueue(restoredCtx)
			for _, source := range []string{primary.String(), vault.String()} {
				_, found := restarted.StructsKeeper.GetInfusion(restoredCtx, generatorID, source)
				require.False(t, found)
			}
			require.Zero(t, redelegationCapacity(t, restarted, restoredCtx, ownerID))
			require.Equal(t, vaultAmount, restarted.BankKeeper.GetBalance(restoredCtx, vault, "ualpha").Amount)
			require.Equal(t, supply, restarted.BankKeeper.GetSupply(restoredCtx, "ualpha").Amount)
		})
	}
}
