package types

import (
	"context"
    "time"
    "cosmossdk.io/math"
	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
    //auth "github.com/cosmos/cosmos-sdk/x/auth/types"
    staking "github.com/cosmos/cosmos-sdk/x/staking/types"
    banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
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

    // Needed for the Join Migration
    ValidateUnbondAmount(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) (shares math.LegacyDec, err error)
    BeginRedelegation(ctx context.Context, delAddr sdk.AccAddress, valSrcAddr, valDstAddr sdk.ValAddress, sharesAmount math.LegacyDec) (completionTime time.Time, err error)

    BondDenom(ctx context.Context) (string, error)
    Delegate(ctx context.Context, delAddr sdk.AccAddress, bondAmt math.Int, tokenSrc staking.BondStatus, validator staking.Validator, subtractAccount bool) (newShares math.LegacyDec, err error)
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
    SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
    SpendableCoin(context.Context, sdk.AccAddress, string) sdk.Coin
    SendCoins(context.Context, sdk.AccAddress, sdk.AccAddress, sdk.Coins) error
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

