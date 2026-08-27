package types

import (
	"context"
	"time"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	//auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// StakingKeeper defines the expected interface for the Staking module.
type StakingKeeper interface {
	ConsensusAddressCodec() address.Codec
	ValidatorByConsAddr(context.Context, sdk.ConsAddress) (staking.ValidatorI, error)
	// Methods imported from account should be defined here

	GetValidator(context.Context, sdk.ValAddress) (staking.Validator, error)
	GetAllValidators(context.Context) ([]staking.Validator, error)
	GetValidators(context.Context, uint32) ([]staking.Validator, error)

	GetValidatorDelegations(context.Context, sdk.ValAddress) ([]staking.Delegation, error)

	GetDelegation(context.Context, sdk.AccAddress, sdk.ValAddress) (staking.Delegation, error)

	GetUnbondingDelegation(context.Context, sdk.AccAddress, sdk.ValAddress) (staking.UnbondingDelegation, error)
	GetUnbondingDelegationByUnbondingID(context.Context, uint64) (staking.UnbondingDelegation, error)

	GetDelegatorDelegations(ctx context.Context, delegator sdk.AccAddress, maxRetrieve uint16) ([]staking.Delegation, error)
	SetDelegation(ctx context.Context, delegation staking.Delegation) error
	RemoveDelegation(ctx context.Context, delegation staking.Delegation) error

	// Needed to refuse a delegation transfer out of an address that is the
	// destination of an in-flight redelegation. SlashRedelegation resolves the
	// delegation to slash through the redelegation record's own delegator
	// address and silently skips when it is missing, so rekeying out from
	// under one makes the stake unslashable.
	HasReceivingRedelegation(ctx context.Context, delAddr sdk.AccAddress, valDstAddr sdk.ValAddress) (bool, error)

	// Needed for the Join Migration
	ValidateUnbondAmount(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) (shares math.LegacyDec, err error)
	BeginRedelegation(ctx context.Context, delAddr sdk.AccAddress, valSrcAddr, valDstAddr sdk.ValAddress, sharesAmount math.LegacyDec) (completionTime time.Time, err error)

	BondDenom(ctx context.Context) (string, error)
	Delegate(ctx context.Context, delAddr sdk.AccAddress, bondAmt math.Int, tokenSrc staking.BondStatus, validator staking.Validator, subtractAccount bool) (newShares math.LegacyDec, err error)
	Undelegate(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, sharesAmount math.LegacyDec) (time.Time, math.Int, error)
	RemoveUnbondingDelegation(ctx context.Context, ubd staking.UnbondingDelegation) error
	SetUnbondingDelegation(ctx context.Context, ubd staking.UnbondingDelegation) error
}

// DistributionKeeper is the read side of x/distribution that a delegation
// transfer needs: whether a (validator, delegator) pair still has the starting
// info that prices its rewards.
type DistributionKeeper interface {
	HasDelegatorStartingInfo(ctx context.Context, val sdk.ValAddress, del sdk.AccAddress) (bool, error)
}

// DistributionHooks is the write side, and the only public route to
// initializeDelegation and withdrawDelegationRewards. Moving a delegation
// between addresses has to drive that lifecycle by hand: staking's
// SetDelegation is a bare store write that fires nothing, so without this the
// destination pair has no starting info and can never withdraw, undelegate or
// redelegate again.
//
// Calling distribution's hooks directly rather than staking's multi-hook is
// deliberate. Slashing registers no delegation hooks, and structs reconciles
// its own infusions explicitly, so the multi-hook would only re-enter us.
type DistributionHooks interface {
	BeforeDelegationCreated(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error
	BeforeDelegationSharesModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error
	AfterDelegationModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error
}

// StakingHooks event hooks for staking validator object (noalias)
type StakingHooks interface {
	AfterValidatorCreated(ctx context.Context, valAddr sdk.ValAddress) error                           // Must be called when a validator is created
	BeforeValidatorModified(ctx context.Context, valAddr sdk.ValAddress) error                         // Must be called when a validator's state changes
	AfterValidatorRemoved(ctx context.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) error // Must be called when a validator is deleted

	AfterValidatorBonded(ctx context.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) error         // Must be called when a validator is bonded
	AfterValidatorBeginUnbonding(ctx context.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) error // Must be called when a validator begins unbonding

	BeforeDelegationCreated(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error        // Must be called when a delegation is created
	BeforeDelegationSharesModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error // Must be called when a delegation's shares are modified
	BeforeDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error        // Must be called when a delegation is removed
	AfterDelegationModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error
	BeforeValidatorSlashed(ctx context.Context, valAddr sdk.ValAddress, fraction math.LegacyDec) error
}

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	// Methods imported from account should be defined here
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI
	GetModuleAddress(string) sdk.AccAddress
	NewAccountWithAddress(context.Context, sdk.AccAddress) sdk.AccountI
	SetAccount(context.Context, sdk.AccountI)
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	// Methods imported from bank should be defined here
	SetDenomMetaData(context.Context, banktypes.Metadata)
	GetDenomMetaData(context.Context, string) (banktypes.Metadata, bool)
	GetSupply(context.Context, string) sdk.Coin
	HasBalance(context.Context, sdk.AccAddress, sdk.Coin) bool
	GetAllBalances(context.Context, sdk.AccAddress) sdk.Coins
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	SpendableCoin(context.Context, sdk.AccAddress, string) sdk.Coin
	SendCoins(context.Context, sdk.AccAddress, sdk.AccAddress, sdk.Coins) error
	// IsSendEnabledCoins and BlockedAddr are the two policies the bank's own
	// MsgServer applies before SendCoins and that SendCoins itself does not:
	// a governance send-disable on a denom, and the module accounts the app
	// declared unreachable. Any handler that moves coins to a destination a
	// message chose has to ask them itself.
	IsSendEnabledCoins(context.Context, ...sdk.Coin) error
	BlockedAddr(sdk.AccAddress) bool
	SendCoinsFromModuleToModule(context.Context, string, string, sdk.Coins) error
	SendCoinsFromAccountToModule(context.Context, sdk.AccAddress, string, sdk.Coins) error
	SendCoinsFromModuleToAccount(context.Context, string, sdk.AccAddress, sdk.Coins) error
	MintCoins(context.Context, string, sdk.Coins) error
	BurnCoins(context.Context, string, sdk.Coins) error
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}
