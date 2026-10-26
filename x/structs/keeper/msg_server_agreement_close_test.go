package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgAgreementClose(t *testing.T) {
	testAgreementMutationAuthority(t, "close")
}

// Open a funded agreement through its handler. Fabricated agreement/allocation
// IDs can exercise a cache while proving nothing about its backing or settlement.
// This fixture covers the legacy checkpoint path; retained service accounting
// has separate real-app tests with history capture enabled.
func testAgreementMutationAuthority(t *testing.T, action string) {
	t.Helper()
	for _, guildOwned := range []bool{false, true} {
		ownership := "player owned"
		if guildOwned {
			ownership = "guild owned"
		}
		t.Run(ownership, func(t *testing.T) {
			for _, name := range []string{"owner with exact key bit", "explicit delegate", "missing agreement", "missing agreement with grant", "unregistered signer", "unprivileged player", "owner key lacks update", "delegate key lacks update"} {
				t.Run(name, func(t *testing.T) {
					f := setupTeardownFixture(t, 10, "0", "0")
					ownerID := f.consumer.Id
					if guildOwned {
						guild := types.CreateEmptyGuild()
						guild.Id, guild.Index, guild.Owner = "0-900", 900, f.consumer.Id
						f.k.SetGuild(f.ctx, guild)
						ownerID = guild.Id
						vaults, err := f.k.NewCurrentContext(f.ctx).EnsureVaults(ownerID)
						require.NoError(t, err)
						f.consumerAcc = vaults[0]
					}
					f.fund(t, f.consumerAcc, 50000)
					_, err := f.ms.AgreementOpen(f.ctx, &types.MsgAgreementOpen{Creator: f.consumer.Creator, OwnerId: ownerID, ProviderId: f.provider.Id, Capacity: 100, Duration: 50})
					require.NoError(t, err)
					ids := f.k.GetAllAgreementIdByProviderIndex(f.ctx, f.provider.Id)
					require.Len(t, ids, 1)
					agreement, found := f.k.GetAgreement(f.ctx, ids[0])
					require.True(t, found)
					require.Equal(t, ownerID, agreement.Owner)
					allocation, found := f.k.GetAllocation(f.ctx, agreement.AllocationId)
					require.True(t, found)
					require.Equal(t, keeperlib.GetObjectID(types.ObjectType_agreement, allocation.Index), agreement.Id)
					require.Equal(t, ownerID, allocation.Controller)
					advanceBlocks(f, 10)
					ctx := sdk.UnwrapSDKContext(f.ctx)
					address := sdk.AccAddress("agreement_other_actor")
					other := testAppendPlayer(f.k, ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
					creator, agreementID, wantError := f.consumer.Creator, agreement.Id, ""
					f.k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermUpdate)
					f.k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(other.Creator), types.PermUpdate)
					switch name {
					case "explicit delegate", "delegate key lacks update":
						creator = other.Creator
						f.k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(agreement.Id, other.Id), types.PermUpdate)
						if name == "delegate key lacks update" {
							f.k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermAll &^ types.PermUpdate)
							wantError = "permission"
						}
					case "missing agreement", "missing agreement with grant":
						agreementID, wantError = "11-999999", "not found"
						if name == "missing agreement with grant" {
							f.k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(agreementID, f.consumer.Id), types.PermUpdate)
						}
					case "unregistered signer":
						creator, wantError = sdk.AccAddress("agreement_unknown_key").String(), "player"
					case "unprivileged player":
						creator, wantError = other.Creator, "permission"
					case "owner key lacks update":
						f.k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(creator), types.PermAll &^ types.PermUpdate)
						wantError = "permission"
					}
					bank := f.k.BankKeeper().(*keepertest.MockBankKeeper)
					bankBefore := bank.Snapshot()
					gridBefore := f.k.GetAllGridExport(ctx)
					permissionsBefore := f.k.GetAllPermissionExport(ctx)
					ctx = ctx.WithEventManager(sdk.NewEventManager())
					var response *types.MsgAgreementResponse
					switch action {
					case "close":
						response, err = f.ms.AgreementClose(ctx, &types.MsgAgreementClose{Creator: creator, AgreementId: agreementID})
					case "increase":
						response, err = f.ms.AgreementCapacityIncrease(ctx, &types.MsgAgreementCapacityIncrease{Creator: creator, AgreementId: agreementID, CapacityIncrease: 100})
					case "decrease":
						response, err = f.ms.AgreementCapacityDecrease(ctx, &types.MsgAgreementCapacityDecrease{Creator: creator, AgreementId: agreementID, CapacityDecrease: 50})
					default:
						t.Fatalf("unknown agreement action %q", action)
					}
					require.NotNil(t, response)
					if wantError != "" {
						require.Error(t, err)
						require.ErrorContains(t, err, wantError)
						actual, exists := f.k.GetAgreement(ctx, agreement.Id)
						require.True(t, exists)
						require.Equal(t, agreement, actual)
						actualAllocation, exists := f.k.GetAllocation(ctx, agreement.AllocationId)
						require.True(t, exists)
						require.Equal(t, allocation, actualAllocation)
						require.Equal(t, bankBefore, bank.Snapshot())
						require.Equal(t, gridBefore, f.k.GetAllGridExport(ctx))
						require.Equal(t, permissionsBefore, f.k.GetAllPermissionExport(ctx))
						require.Equal(t, ids, f.k.GetAllAgreementIdByProviderIndex(ctx, f.provider.Id))
						require.Contains(t, f.k.GetAllAgreementIdByExpirationIndex(ctx, agreement.EndBlock), agreement.Id)
						for _, event := range ctx.EventManager().ABCIEvents() {
							require.NotEqual(t, "structs.structs.EventAgreement", event.Type)
							require.NotEqual(t, "structs.structs.EventDelete", event.Type)
						}
						return
					}
					require.NoError(t, err)
					require.Equal(t, math.NewInt(10000), f.balance(f.earningsAcc), "ten blocks at capacity 100 and rate 10 are earned before changing service")
					if action == "close" {
						f.requireSettledOnce(t, agreement)
						require.Equal(t, math.NewInt(40000), f.balance(f.consumerAcc))
						require.True(t, f.balance(f.collateralAcc).IsZero())
						require.Zero(t, f.agreementLoad())
						beforeRepeat := bank.Snapshot()
						_, err = f.ms.AgreementClose(ctx, &types.MsgAgreementClose{Creator: creator, AgreementId: agreementID})
						require.Error(t, err)
						require.Equal(t, beforeRepeat, bank.Snapshot())
					} else {
						wantCapacity, wantDuration := uint64(200), uint64(20)
						if action == "decrease" {
							wantCapacity, wantDuration = 50, 80
						}
						actual, exists := f.k.GetAgreement(ctx, agreement.Id)
						require.True(t, exists)
						expected := agreement
						expected.Capacity, expected.StartBlock, expected.EndBlock = wantCapacity, uint64(ctx.BlockHeight()), uint64(ctx.BlockHeight())+wantDuration
						require.Equal(t, expected, actual)
						require.Equal(t, wantCapacity, f.agreementLoad())
						currentAllocation, exists := f.k.NewCurrentContext(ctx).GetAllocation(agreement.AllocationId)
						require.True(t, exists)
						require.Equal(t, wantCapacity, currentAllocation.GetPower())
						require.NotContains(t, f.k.GetAllAgreementIdByExpirationIndex(ctx, agreement.EndBlock), agreement.Id)
						require.Contains(t, f.k.GetAllAgreementIdByExpirationIndex(ctx, expected.EndBlock), agreement.Id)
						require.Equal(t, math.NewInt(40000), f.balance(f.collateralAcc))
						require.True(t, f.balance(f.consumerAcc).IsZero())
					}
				})
			}
		})
	}
}

func TestExistingAgreementRejectsWarmCacheAfterTeardown(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")
	agreement, _ := f.openAgreement(t, 100, 50)
	cc := f.k.NewCurrentContext(f.ctx)
	loaded, err := cc.GetExistingAgreement(agreement.Id)
	require.NoError(t, err)
	require.Same(t, loaded, cc.GetAgreement(agreement.Id))
	require.NoError(t, loaded.PrematureCloseByConsumer())
	_, present := f.k.GetAgreement(f.ctx, agreement.Id)
	require.False(t, present, "teardown removes the stored row immediately")
	require.True(t, loaded.AgreementLoaded, "the old record is still held in the shared cache")
	_, err = cc.GetExistingAgreement(agreement.Id)
	require.ErrorIs(t, err, types.ErrObjectNotFound, "a warm cache cannot revive a removed agreement")
	cc.CommitAll()
	_, err = f.k.NewCurrentContext(f.ctx).GetExistingAgreement(agreement.Id)
	require.ErrorIs(t, err, types.ErrObjectNotFound)
}
