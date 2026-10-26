package v0_22_0

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	"structs/app/upgrades"
)

// MigrateICAHostPolicy disables unused inbound account control. Clearing the
// wildcard as well means re-enabling hosting alone cannot reopen every handler.
// Account/channel records, ICA controller and ICS-20 transfers are unchanged.
func MigrateICAHostPolicy(ctx context.Context, keepers *upgrades.Keepers) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	keepers.ICAHostKeeper.SetParams(sdkCtx, icahosttypes.NewParams(false, []string{}))
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent("ica_host_policy",
		sdk.NewAttribute("host_enabled", "false"),
		sdk.NewAttribute("allow_messages", "[]"),
		sdk.NewAttribute("upgrade", UpgradeName),
	))
}
