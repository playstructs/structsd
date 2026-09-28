package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgAgreementDurationIncrease(t *testing.T) {
	for _, scenario := range []string{"success", "missing_agreement", "no_update_permission", "unregistered_signer", "no_transfer_permission", "insufficient_funds", "zero", "above_maximum", "overflow"} {
		t.Run(scenario, func(t *testing.T) {
			f := setupTeardownFixture(t, 100, "0", "0")
			f.provider.DurationMaximum = 10
			f.k.SetProvider(f.ctx, f.provider)
			agreement, _ := f.openAgreement(t, 100, 5)
			// Capacity exceeds maximum duration deliberately: swapping those
			// arguments would reject the successful three-block extension.
			if scenario != "insufficient_funds" {
				f.fund(t, f.consumerAcc, 30000)
			}
			primary, err := sdk.AccAddressFromBech32(f.consumer.PrimaryAddress)
			require.NoError(t, err)
			f.fund(t, primary, 777)
			msg := &types.MsgAgreementDurationIncrease{Creator: f.consumer.Creator, AgreementId: agreement.Id, DurationIncrease: 3}
			switch scenario {
			case "missing_agreement":
				msg.AgreementId = "10-999999"
			case "no_update_permission":
				msg.Creator = f.provider.Creator
			case "unregistered_signer":
				msg.Creator = sdk.AccAddress("duration-unregistered").String()
			case "no_transfer_permission":
				testPermissionRemove(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(msg.Creator), types.PermTokenTransfer)
			case "zero":
				msg.DurationIncrease = 0
			case "above_maximum":
				msg.DurationIncrease = 6
			case "overflow":
				agreement.EndBlock = ^uint64(0)
				_, err := f.k.SetAgreement(f.ctx, agreement)
				require.NoError(t, err)
				msg.DurationIncrease = 1
			}
			vaultBefore, collateralBefore := f.balance(f.consumerAcc), f.balance(f.collateralAcc)
			loadBefore := f.agreementLoad()
			_, err = f.ms.AgreementDurationIncrease(f.ctx, msg)
			stored, found := f.k.GetAgreement(f.ctx, agreement.Id)
			require.True(t, found)
			if scenario == "success" {
				require.NoError(t, err)
				require.Equal(t, agreement.EndBlock+3, stored.EndBlock)
				require.True(t, vaultBefore.SubRaw(30000).Equal(f.balance(f.consumerAcc)))
				require.True(t, collateralBefore.AddRaw(30000).Equal(f.balance(f.collateralAcc)))
				agreement.EndBlock = stored.EndBlock
			} else {
				require.Error(t, err)
				require.Equal(t, vaultBefore, f.balance(f.consumerAcc))
				require.Equal(t, collateralBefore, f.balance(f.collateralAcc))
			}
			require.Equal(t, agreement, stored, "extension may only change the end block")
			require.Equal(t, loadBefore, f.agreementLoad())
			require.Equal(t, math.NewInt(777), f.balance(primary), "extension must not spend signing-key funds")
		})
	}
}
