package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

// The provider and agreement handlers used to discard the error from the creator
// lookup and then call a method on the nil PlayerCache it returns. The ante
// registration check made that hard to reach, but a fee-paying mixed tx skipped
// the ante entirely, so an unregistered creator could panic the handler instead
// of being rejected. Every one of them must now return a typed error.
func TestUnregisteredCreatorIsRejectedNotPanicked(t *testing.T) {
	unregistered := sdk.AccAddress("unregistered_padding_address_1234567890").String()

	cases := []struct {
		name string
		call func(ms types.MsgServer, wctx sdk.Context) error
	}{
		{"provider create", func(ms types.MsgServer, wctx sdk.Context) error {
			_, err := ms.ProviderCreate(wctx, &types.MsgProviderCreate{Creator: unregistered, SubstationId: "4-0"})
			return err
		}},
		{"provider delete", func(ms types.MsgServer, wctx sdk.Context) error {
			_, err := ms.ProviderDelete(wctx, &types.MsgProviderDelete{Creator: unregistered, ProviderId: "9-0"})
			return err
		}},
		{"provider withdraw balance", func(ms types.MsgServer, wctx sdk.Context) error {
			_, err := ms.ProviderWithdrawBalance(wctx, &types.MsgProviderWithdrawBalance{Creator: unregistered, ProviderId: "9-0"})
			return err
		}},
		{"provider update access policy", func(ms types.MsgServer, wctx sdk.Context) error {
			_, err := ms.ProviderUpdateAccessPolicy(wctx, &types.MsgProviderUpdateAccessPolicy{Creator: unregistered, ProviderId: "9-0"})
			return err
		}},
		{"provider update capacity maximum", func(ms types.MsgServer, wctx sdk.Context) error {
			_, err := ms.ProviderUpdateCapacityMaximum(wctx, &types.MsgProviderUpdateCapacityMaximum{Creator: unregistered, ProviderId: "9-0"})
			return err
		}},
		{"provider update capacity minimum", func(ms types.MsgServer, wctx sdk.Context) error {
			_, err := ms.ProviderUpdateCapacityMinimum(wctx, &types.MsgProviderUpdateCapacityMinimum{Creator: unregistered, ProviderId: "9-0"})
			return err
		}},
		{"provider update duration maximum", func(ms types.MsgServer, wctx sdk.Context) error {
			_, err := ms.ProviderUpdateDurationMaximum(wctx, &types.MsgProviderUpdateDurationMaximum{Creator: unregistered, ProviderId: "9-0"})
			return err
		}},
		{"provider update duration minimum", func(ms types.MsgServer, wctx sdk.Context) error {
			_, err := ms.ProviderUpdateDurationMinimum(wctx, &types.MsgProviderUpdateDurationMinimum{Creator: unregistered, ProviderId: "9-0"})
			return err
		}},
		{"agreement close", func(ms types.MsgServer, wctx sdk.Context) error {
			_, err := ms.AgreementClose(wctx, &types.MsgAgreementClose{Creator: unregistered, AgreementId: "10-0"})
			return err
		}},
		{"agreement capacity increase", func(ms types.MsgServer, wctx sdk.Context) error {
			_, err := ms.AgreementCapacityIncrease(wctx, &types.MsgAgreementCapacityIncrease{Creator: unregistered, AgreementId: "10-0"})
			return err
		}},
		{"agreement capacity decrease", func(ms types.MsgServer, wctx sdk.Context) error {
			_, err := ms.AgreementCapacityDecrease(wctx, &types.MsgAgreementCapacityDecrease{Creator: unregistered, AgreementId: "10-0"})
			return err
		}},
		{"agreement duration increase", func(ms types.MsgServer, wctx sdk.Context) error {
			_, err := ms.AgreementDurationIncrease(wctx, &types.MsgAgreementDurationIncrease{Creator: unregistered, AgreementId: "10-0"})
			return err
		}},
		{"permission grant on address", func(ms types.MsgServer, wctx sdk.Context) error {
			_, err := ms.PermissionGrantOnAddress(wctx, &types.MsgPermissionGrantOnAddress{
				Creator:     unregistered,
				Address:     unregistered,
				Permissions: uint64(types.PermPlay),
			})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ms, ctx := setupMsgServer(t)
			wctx := sdk.UnwrapSDKContext(ctx)

			require.NotPanics(t, func() {
				err := tc.call(ms, wctx)
				require.Error(t, err)
				require.Contains(t, err.Error(), "not associated")
			})
		})
	}
}
