package ante

import (
	"errors"

	circuitante "cosmossdk.io/x/circuit/ante"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	txsigning "cosmossdk.io/x/tx/signing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

type HandlerOptions struct {
	AccountKeeper   authkeeper.AccountKeeper
	BankKeeper      bankkeeper.Keeper
	FeegrantKeeper  feegrantkeeper.Keeper
	SignModeHandler *txsigning.HandlerMap
	CircuitKeeper   circuitante.CircuitBreaker
	StructsKeeper   StructsAnteKeeper

	MaxFreeTxSize int
	MaxMsgCount   int
	FreeGasCap    uint64
	PlayerMsgCap  uint64
}

func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	if options.SignModeHandler == nil {
		return nil, errors.New("sign mode handler is required for ante builder")
	}
	if options.StructsKeeper == nil {
		return nil, errors.New("structs keeper is required for ante builder")
	}

	anteDecorators := []sdk.AnteDecorator{
		// 1-2: Cheap pre-checks (no state reads, no gas meter needed)
		NewTxSizeDecorator(options.MaxFreeTxSize),
		NewMsgCountDecorator(options.MaxMsgCount),

		// 3: SDK SetUpContext creates the initial gas meter from block gas limit
		ante.NewSetUpContextDecorator(),

		// 4: Replace gas meter with free meter for pure-Structs txs (MUST be after SetUpContext)
		NewGasRouterDecorator(options.FreeGasCap),

		// 5: Circuit breaker (governance can disable message types)
		circuitante.NewCircuitBreakerDecorator(options.CircuitKeeper),

		// 6-9: Standard SDK tx validation
		ante.NewExtensionOptionsDecorator(nil),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),

		// 10: Gas for tx size (counts toward free meter cap)
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),

		// 11: Conditional fee deduction (skip for free Structs txs)
		NewConditionalFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper),

		// 12-15: Signature handling
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, ante.DefaultSigVerificationGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),

		// 16: Nonce increment (prevents replay attacks)
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),

		// 17: PubKey derivation check for signature-bearing Structs messages
		NewPubKeyDerivationDecorator(),

		// 18: Structs-specific checks (player lookup, permissions, charge, msg cap)
		NewStructsDecorator(options.StructsKeeper, options.PlayerMsgCap),

		// 19: Per-object throttles (proof, fleet, explore, register)
		NewThrottleDecorator(options.StructsKeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
