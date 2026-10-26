package structs_test

import (
	"bytes"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	structs "structs/x/structs/module"
	"structs/x/structs/types"
)

func TestGenesisPreservesVaultReactorInfusions(t *testing.T) {
	for _, mode := range []string{"active", "guild", "registered-key", "jailed", "inactive", "unbonding-only"} {
		t.Run(mode, func(t *testing.T) {
			state := vaultGenesisFixture()

			vault, err := types.VaultAddress("1-1")
			require.NoError(t, err)
			ownerID := "1-1"
			if mode == "guild" {
				ownerID = "0-1"
				vault, err = types.VaultAddress(ownerID)
				require.NoError(t, err)
			}
			if mode == "registered-key" {
				vault = sdk.AccAddress(bytes.Repeat([]byte{0x46}, 20))
				state.AddressList = []*types.AddressRecord{{Address: vault.String(), PlayerIndex: 1}}
			}
			validator := sdk.ValAddress(bytes.Repeat([]byte{0x45}, 20))
			reactor := types.Reactor{Id: "3-1", Validator: validator.String(), RawAddress: validator, DefaultCommission: math.LegacyMustNewDecFromStr("0.04")}
			state.ReactorList = []types.Reactor{reactor}
			record := types.Infusion{DestinationType: types.ObjectType_reactor, DestinationId: reactor.Id, PlayerId: "1-1", Address: vault.String(), Fuel: 100, Power: 100, Ratio: 1, Commission: reactor.DefaultCommission, Defusing: 20}
			if mode == "guild" { record.PlayerId, record.OwnerId = "", ownerID }
			if mode == "unbonding-only" {
				record.Fuel, record.Power, record.Ratio = 0, 0, 0
			}
			state.InfusionList = []types.Infusion{record}
			for restart := 0; restart < 2; restart++ {
				k, ctx := keepertest.StructsKeeper(t)
				mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)
				mock.AddValidator(validator, math.NewInt(1000))
				if mode == "jailed" || mode == "inactive" {
					mock.JailValidator(validator)
					if mode == "inactive" { mock.UnjailValidator(validator) }
				}
				if mode != "unbonding-only" {
					require.NoError(t, mock.SetDelegation(ctx, stakingtypes.Delegation{DelegatorAddress: vault.String(), ValidatorAddress: validator.String(), Shares: math.LegacyNewDec(100)}))
				}
				require.NoError(t, mock.SetUnbondingDelegation(ctx, stakingtypes.UnbondingDelegation{DelegatorAddress: vault.String(), ValidatorAddress: validator.String(), Entries: []stakingtypes.UnbondingDelegationEntry{{Balance: math.NewInt(20), InitialBalance: math.NewInt(20), CompletionTime: ctx.BlockTime().Add(time.Hour)}}}))
				structs.InitGenesis(ctx, k, *state)
				actual, found := k.GetInfusion(ctx, reactor.Id, vault.String())
				require.True(t, found, "zero-power positions must retain weight and unbonding balance")

				require.Equal(t, record.Fuel, actual.Fuel)
				require.Equal(t, uint64(20), actual.Defusing)
				require.Equal(t, ownerID, actual.AssetOwnerID())
				wantPower, wantPlayer := uint64(100), uint64(96)
				if mode == "jailed" || mode == "inactive" || mode == "unbonding-only" {
					wantPower, wantPlayer = 0, 0
				}
				require.Equal(t, wantPower, actual.Power)
				require.Equal(t, wantPlayer, k.GetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, ownerID)))
				if mode != "registered-key" {
					require.Zero(t, k.GetPlayerIndexFromAddress(ctx, vault.String()))
				}
				require.Len(t, k.GetInfusionMaturitySweepQueueExport(ctx), 1, "staking unbondings rebuild their maturity appointment")
				require.NoError(t, actual.ValidateAssetOwner())
				state = structs.ExportGenesis(ctx, k)
			}
		})
	}
}

func TestGenesisReactorInfusionRejectsBadAssociationBeforeWrites(t *testing.T) {
	state := vaultGenesisFixture()
	vault, err := types.VaultAddress("1-1")
	require.NoError(t, err)
	state.InfusionList = []types.Infusion{{DestinationType: types.ObjectType_reactor, DestinationId: "3-1", PlayerId: "1-2", Address: vault.String(), Commission: math.LegacyZeroDec()}}
	k, ctx := keepertest.StructsKeeper(t)
	require.Panics(t, func() { structs.InitGenesis(ctx, k, *state) })
	require.Empty(t, k.GetAllPlayer(ctx))
	require.Nil(t, k.AccountKeeper().GetAccount(ctx, vault))
}

func TestGenesisRebuildsReactorBackingAndPricesTopUps(t *testing.T) {
	for _, inactive := range []bool{false, true} {
		name := "bonded"
		if inactive { name = "inactive" }
		t.Run(name, func(t *testing.T) {
			state := vaultGenesisFixture()

			address := sdk.AccAddress(bytes.Repeat([]byte{0x47}, 20))
			validator := sdk.ValAddress(bytes.Repeat([]byte{0x48}, 20))
			state.AddressList = []*types.AddressRecord{{Address: address.String(), PlayerIndex: 1}}
			reactor := types.Reactor{Id: "3-1", Validator: validator.String(), RawAddress: validator, DefaultCommission: math.LegacyZeroDec()}
			state.ReactorList = []types.Reactor{reactor}
			// Legacy genesis carries no weight. Its overstated fuel must be
			// rebuilt from staking before granting the historical entitlement.
			state.InfusionList = []types.Infusion{{DestinationType: types.ObjectType_reactor, DestinationId: reactor.Id, PlayerId: "1-1", Address: address.String(), Fuel: 200, Power: 200, Ratio: 1, Commission: math.LegacyZeroDec()}}
			for restart := 0; restart < 2; restart++ {
				k, ctx := keepertest.StructsKeeper(t)
				mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)
				mock.AddValidator(validator, math.NewInt(1000))
				if inactive { mock.JailValidator(validator); mock.UnjailValidator(validator) }
				require.NoError(t, mock.SetDelegation(ctx, stakingtypes.Delegation{DelegatorAddress: address.String(), ValidatorAddress: validator.String(), Shares: math.LegacyNewDec(100)}))
				structs.InitGenesis(ctx, k, *state)
				actual, found := k.GetInfusion(ctx, reactor.Id, address.String())
				require.True(t, found)
				require.Equal(t, uint64(100), actual.Fuel)

				power := uint64(100)
				if inactive { power = 0 }
				require.Equal(t, power, actual.Power)
				require.Equal(t, power, k.GetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, "1-1")))
				state = structs.ExportGenesis(ctx, k)
				// A later hook sees only the increment as newly deposited fuel,
				// even when no validator snapshot was supplied in this genesis.
				require.NoError(t, mock.SetDelegation(ctx, stakingtypes.Delegation{DelegatorAddress: address.String(), ValidatorAddress: validator.String(), Shares: math.LegacyNewDec(200)}))
				k.ReconcileInfusionForDelegation(ctx, address, validator)
				actual, found = k.GetInfusion(ctx, reactor.Id, address.String())
				require.True(t, found)
				require.Equal(t, uint64(200), actual.Fuel)

				if inactive { require.Zero(t, actual.Power) } else { require.Equal(t, uint64(200), actual.Power) }
			}
		})
	}
}
