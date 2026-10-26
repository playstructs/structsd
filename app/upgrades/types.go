package upgrades

import (
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	icahostkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/keeper"

	structsmodulekeeper "structs/x/structs/keeper"
)

// Keepers holds references to keepers that upgrade handlers may need.
type Keepers struct {
	UpgradeKeeper  *upgradekeeper.Keeper
	StructsKeeper  structsmodulekeeper.Keeper
	AccountKeeper  authkeeper.AccountKeeper
	BankKeeper     bankkeeper.Keeper
	StakingKeeper  *stakingkeeper.Keeper
	DistrKeeper    distrkeeper.Keeper
	ICAHostKeeper  icahostkeeper.Keeper
}

// Upgrade defines an on-chain upgrade with its name, handler factory,
// and any store key additions/removals needed at the upgrade height.
type Upgrade struct {
	UpgradeName          string
	CreateUpgradeHandler func(mm *module.Manager, configurator module.Configurator, keepers *Keepers) upgradetypes.UpgradeHandler
	StoreUpgrades        storetypes.StoreUpgrades
}
