package cmd

import (
	cmtcfg "github.com/cometbft/cometbft/config"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/app"
)

// InitSDKConfig initializes the SDK configuration with the correct bech32 prefix.
// This must be called before any modules are initialized to ensure addresses use the correct prefix.
func InitSDKConfig() {
	// Set prefixes
	accountPubKeyPrefix := app.AccountAddressPrefix + "pub"
	validatorAddressPrefix := app.AccountAddressPrefix + "valoper"
	validatorPubKeyPrefix := app.AccountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := app.AccountAddressPrefix + "valcons"
	consNodePubKeyPrefix := app.AccountAddressPrefix + "valconspub"

	// Set and seal config
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(app.AccountAddressPrefix, accountPubKeyPrefix)
	config.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	config.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
	config.Seal()
}

// initCometBFTConfig helps to override default CometBFT Config values.
// return cmtcfg.DefaultConfig if no custom configuration is required for the application.
func initCometBFTConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()

	// these values put a higher strain on node memory
	// cfg.P2P.MaxNumInboundPeers = 100
	// cfg.P2P.MaxNumOutboundPeers = 40

	return cfg
}

// StructsAnteConfig holds node-local tuning knobs for the custom Structs ante
// handler chain.
type StructsAnteConfig struct {
	CheckTxAddrCap uint64 `mapstructure:"checktx-addr-cap"`
}

func defaultStructsAnteConfig() StructsAnteConfig {
	return StructsAnteConfig{
		CheckTxAddrCap: 5,
	}
}

// initAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	type CustomAppConfig struct {
		serverconfig.Config `mapstructure:",squash"`
		StructsAnte         StructsAnteConfig `mapstructure:"structs-ante"`
	}

	srvCfg := serverconfig.DefaultConfig()

	customAppConfig := CustomAppConfig{
		Config:      *srvCfg,
		StructsAnte: defaultStructsAnteConfig(),
	}

	customAppTemplate := serverconfig.DefaultConfigTemplate + `
###############################################################################
###                       Structs Ante Handler                              ###
###############################################################################

[structs-ante]

# Maximum free Structs transactions a single address can submit per block
# during CheckTx (mempool admission). Node-local, not consensus.
checktx-addr-cap = 5

# Deprecated/ignored in this binary: max-free-tx-size, max-msg-count,
# free-gas-cap, and player-msg-cap. Those limits are deterministic defaults.
`

	return customAppTemplate, customAppConfig
}
