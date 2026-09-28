package app_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vesting "github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	"github.com/stretchr/testify/require"
	"structs/app/upgrades"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

func TestVaultMigrationAcceptsAdversarialVestingDestination(t *testing.T) {
	for _, kind := range []string{"permanent", "continuous", "delayed", "periodic"} {
		for _, sourceVesting := range []bool{false, true} {
			t.Run(kind+"/"+map[bool]string{false: "ordinary-source", true: "vesting-source"}[sourceVesting], func(t *testing.T) {
				bApp, ctx := setupJailGateApp(t)
				primary, secondary, playerID := newTransferPlayer(t, bApp, ctx)
				fundRedelegationAccount(t, bApp, ctx, primary, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1000)))
				fundRedelegationAccount(t, bApp, ctx, secondary, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1000)))
				vault, err := types.VaultAddress(playerID)
				require.NoError(t, err)
				require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, vault))
				server := vesting.NewMsgServerImpl(bApp.AccountKeeper, bApp.BankKeeper)

				switch kind {
				case "permanent":
					_, err = server.CreatePermanentLockedAccount(ctx, &vestingtypes.MsgCreatePermanentLockedAccount{FromAddress: secondary.String(), ToAddress: vault.String(), Amount: sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1))})
				case "continuous", "delayed":
					_, err = server.CreateVestingAccount(ctx, &vestingtypes.MsgCreateVestingAccount{FromAddress: secondary.String(), ToAddress: vault.String(), Amount: sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1)), EndTime: ctx.BlockTime().Add(time.Hour).Unix(), Delayed: kind == "delayed"})
				case "periodic":
					_, err = server.CreatePeriodicVestingAccount(ctx, &vestingtypes.MsgCreatePeriodicVestingAccount{FromAddress: secondary.String(), ToAddress: vault.String(), StartTime: ctx.BlockTime().Unix(), VestingPeriods: []vestingtypes.Period{{Length: 3600, Amount: sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1))}}})
				}
				require.NoError(t, err, "this is an ordinary SDK operation available before the upgrade")
				locked := int64(1)
				if sourceVesting {
					base := bApp.AccountKeeper.GetAccount(ctx, primary).(*authtypes.BaseAccount)
					source, err := vestingtypes.NewDelayedVestingAccount(base, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100)), ctx.BlockTime().Add(time.Hour).Unix())
					require.NoError(t, err)
					bApp.AccountKeeper.SetAccount(ctx, source)
				}
				before := bApp.BankKeeper.GetBalance(ctx, primary, "ualpha").Amount.
					Add(bApp.BankKeeper.GetBalance(ctx, secondary, "ualpha").Amount).
					Add(bApp.BankKeeper.GetBalance(ctx, vault, "ualpha").Amount)
				if sourceVesting { before = before.Sub(bApp.BankKeeper.GetBalance(ctx, primary, "ualpha").Amount) }
				keepers := &upgrades.Keepers{StructsKeeper: bApp.StructsKeeper, StakingKeeper: bApp.StakingKeeper, AccountKeeper: bApp.AccountKeeper, DistrKeeper: bApp.DistrKeeper}
				require.NoError(t, v22.MigrateVaultAssets(ctx, keepers))
				account := bApp.AccountKeeper.GetAccount(ctx, vault).(vestingexported.VestingAccount)
				require.Equal(t, locked, account.LockedCoins(ctx.BlockTime()).AmountOf("ualpha").Int64())
				require.Equal(t, before, bApp.BankKeeper.GetBalance(ctx, vault, "ualpha").Amount)
				require.Equal(t, before.SubRaw(locked), bApp.BankKeeper.SpendableCoin(ctx, vault, "ualpha").Amount)
				_, err = server.CreatePermanentLockedAccount(ctx, &vestingtypes.MsgCreatePermanentLockedAccount{
					FromAddress: secondary.String(), ToAddress: vault.String(), Amount: sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1)),
				})
				require.ErrorContains(t, err, "already exists", "provisioned vaults cannot be replaced by native account creation")
			})
		}
	}
}
