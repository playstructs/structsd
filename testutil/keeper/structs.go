package keeper

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/core/address"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
	"github.com/stretchr/testify/require"

	"structs/x/structs/keeper"
	"structs/x/structs/types"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// MockAccountKeeper is a mock implementation of the AccountKeeper interface
type MockAccountKeeper struct {
	accounts map[string]sdk.AccountI
}

func NewMockAccountKeeper() *MockAccountKeeper {
	return &MockAccountKeeper{
		accounts: make(map[string]sdk.AccountI),
	}
}

func (m *MockAccountKeeper) GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI {
	return m.accounts[addr.String()]
}

func (m *MockAccountKeeper) SetAccount(ctx context.Context, acc sdk.AccountI) {
	m.accounts[acc.GetAddress().String()] = acc
}

func (m *MockAccountKeeper) NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI {
	acc := authtypes.NewBaseAccount(addr, nil, 0, 0)
	m.accounts[addr.String()] = acc
	return acc
}

func (m *MockAccountKeeper) GetModuleAddress(module string) sdk.AccAddress {
	return authtypes.NewModuleAddress(module)
}

// MockBankKeeper is a mock implementation of the BankKeeper interface
type MockBankKeeper struct {
	balances map[string]sdk.Coins
	metadata map[string]banktypes.Metadata
	supply   map[string]math.Int // Track supply per denom
}

func NewMockBankKeeper() *MockBankKeeper {
	return &MockBankKeeper{
		balances: make(map[string]sdk.Coins),
		metadata: make(map[string]banktypes.Metadata),
		supply:   make(map[string]math.Int),
	}
}

func (m *MockBankKeeper) SetDenomMetaData(ctx context.Context, metadata banktypes.Metadata) {
	m.metadata[metadata.Base] = metadata
}

func (m *MockBankKeeper) GetDenomMetaData(ctx context.Context, denom string) (banktypes.Metadata, bool) {
	metadata, found := m.metadata[denom]
	return metadata, found
}

func (m *MockBankKeeper) GetSupply(ctx context.Context, denom string) sdk.Coin {
	// Return tracked supply or zero if not tracked
	supply, exists := m.supply[denom]
	if !exists {
		supply = math.ZeroInt()
	}
	return sdk.NewCoin(denom, supply)
}

func (m *MockBankKeeper) HasBalance(ctx context.Context, addr sdk.AccAddress, coin sdk.Coin) bool {
	return true
}

func (m *MockBankKeeper) SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	coins, ok := m.balances[addr.String()]
	if !ok {
		return sdk.Coins{}
	}
	return coins
}

func (m *MockBankKeeper) SpendableCoin(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	coins := m.SpendableCoins(ctx, addr)
	amount := coins.AmountOf(denom)
	return sdk.NewCoin(denom, amount)
}

func (m *MockBankKeeper) SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	fromBal, exists := m.balances[fromAddr.String()]
	if !exists {
		fromBal = sdk.Coins{}
	}
	if fromBal.IsAllLT(amt) {
		// Use fmt.Errorf instead of sdkerrors.New to avoid error code registration issues
		// when running multiple tests together
		return fmt.Errorf("insufficient funds")
	}
	m.balances[fromAddr.String()] = fromBal.Sub(amt...)
	toBal, exists := m.balances[toAddr.String()]
	if !exists {
		toBal = sdk.Coins{}
	}
	m.balances[toAddr.String()] = toBal.Add(amt...)
	return nil
}

func (m *MockBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	// For testing, we'll just track module balances in a special way
	// This is a simplified implementation
	return nil
}

func (m *MockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	fromBal := m.balances[senderAddr.String()]
	if fromBal.IsAllLT(amt) {
		// Use fmt.Errorf instead of sdkerrors.New to avoid error code registration issues
		// when running multiple tests together
		return fmt.Errorf("insufficient funds")
	}
	m.balances[senderAddr.String()] = fromBal.Sub(amt...)
	return nil
}

func (m *MockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	toBal, exists := m.balances[recipientAddr.String()]
	if !exists {
		toBal = sdk.Coins{}
	}
	m.balances[recipientAddr.String()] = toBal.Add(amt...)
	return nil
}

func (m *MockBankKeeper) MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	// Track supply for each denom
	for _, coin := range amt {
		currentSupply, exists := m.supply[coin.Denom]
		if !exists {
			currentSupply = math.ZeroInt()
		}
		m.supply[coin.Denom] = currentSupply.Add(coin.Amount)
	}
	return nil
}

func (m *MockBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	// Decrease supply for each denom
	for _, coin := range amt {
		currentSupply, exists := m.supply[coin.Denom]
		if !exists {
			currentSupply = math.ZeroInt()
		}
		newSupply := currentSupply.Sub(coin.Amount)
		if newSupply.IsNegative() {
			newSupply = math.ZeroInt()
		}
		m.supply[coin.Denom] = newSupply
	}
	return nil
}

// delegationKey uniquely identifies a delegation by delegator+validator.
type delegationKey struct {
	Delegator string
	Validator string
}

// MockStakingKeeper is a stateful mock of the StakingKeeper interface.
// It tracks validators, delegations, and unbonding delegations so that
// reactor infuse/defuse/migrate/cancel tests can exercise the full handler path.
type MockStakingKeeper struct {
	validators   map[string]stakingtypes.Validator
	delegations  map[delegationKey]stakingtypes.Delegation
	unbondings   map[delegationKey]stakingtypes.UnbondingDelegation
	nextUnbondID uint64
}

func NewMockStakingKeeper() *MockStakingKeeper {
	return &MockStakingKeeper{
		validators:   make(map[string]stakingtypes.Validator),
		delegations:  make(map[delegationKey]stakingtypes.Delegation),
		unbondings:   make(map[delegationKey]stakingtypes.UnbondingDelegation),
		nextUnbondID: 1,
	}
}

// AddValidator registers a bonded validator with 1:1 token-to-share ratio.
func (m *MockStakingKeeper) AddValidator(operatorAddr sdk.ValAddress, tokens math.Int) {
	val := stakingtypes.Validator{
		OperatorAddress: operatorAddr.String(),
		Status:          stakingtypes.Bonded,
		Tokens:          tokens,
		DelegatorShares: math.LegacyNewDecFromInt(tokens),
	}
	m.validators[operatorAddr.String()] = val
}

func (m *MockStakingKeeper) ConsensusAddressCodec() address.Codec {
	return nil
}

func (m *MockStakingKeeper) ValidatorByConsAddr(ctx context.Context, consAddr sdk.ConsAddress) (stakingtypes.ValidatorI, error) {
	return nil, nil
}

func (m *MockStakingKeeper) GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error) {
	val, ok := m.validators[addr.String()]
	if !ok {
		return stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound
	}
	return val, nil
}

func (m *MockStakingKeeper) GetAllValidators(ctx context.Context) ([]stakingtypes.Validator, error) {
	vals := make([]stakingtypes.Validator, 0, len(m.validators))
	for _, v := range m.validators {
		vals = append(vals, v)
	}
	return vals, nil
}

func (m *MockStakingKeeper) GetValidators(ctx context.Context, maxRetrieve uint32) ([]stakingtypes.Validator, error) {
	return m.GetAllValidators(ctx)
}

func (m *MockStakingKeeper) GetValidatorDelegations(ctx context.Context, valAddr sdk.ValAddress) ([]stakingtypes.Delegation, error) {
	var result []stakingtypes.Delegation
	for k, d := range m.delegations {
		if k.Validator == valAddr.String() {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *MockStakingKeeper) GetDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (stakingtypes.Delegation, error) {
	dk := delegationKey{Delegator: delAddr.String(), Validator: valAddr.String()}
	del, ok := m.delegations[dk]
	if !ok {
		return stakingtypes.Delegation{}, stakingtypes.ErrNoDelegation
	}
	return del, nil
}

func (m *MockStakingKeeper) GetUnbondingDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (stakingtypes.UnbondingDelegation, error) {
	dk := delegationKey{Delegator: delAddr.String(), Validator: valAddr.String()}
	ubd, ok := m.unbondings[dk]
	if !ok {
		return stakingtypes.UnbondingDelegation{}, stakingtypes.ErrNoUnbondingDelegation
	}
	return ubd, nil
}

func (m *MockStakingKeeper) GetUnbondingDelegationByUnbondingID(ctx context.Context, id uint64) (stakingtypes.UnbondingDelegation, error) {
	for _, ubd := range m.unbondings {
		for _, entry := range ubd.Entries {
			if entry.UnbondingId == id {
				return ubd, nil
			}
		}
	}
	return stakingtypes.UnbondingDelegation{}, stakingtypes.ErrNoUnbondingDelegation
}

func (m *MockStakingKeeper) GetDelegatorDelegations(ctx context.Context, delegator sdk.AccAddress, maxRetrieve uint16) ([]stakingtypes.Delegation, error) {
	var result []stakingtypes.Delegation
	for k, d := range m.delegations {
		if k.Delegator == delegator.String() {
			result = append(result, d)
			if uint16(len(result)) >= maxRetrieve {
				break
			}
		}
	}
	return result, nil
}

func (m *MockStakingKeeper) SetDelegation(ctx context.Context, delegation stakingtypes.Delegation) error {
	dk := delegationKey{Delegator: delegation.DelegatorAddress, Validator: delegation.ValidatorAddress}
	m.delegations[dk] = delegation
	return nil
}

func (m *MockStakingKeeper) RemoveDelegation(ctx context.Context, delegation stakingtypes.Delegation) error {
	dk := delegationKey{Delegator: delegation.DelegatorAddress, Validator: delegation.ValidatorAddress}
	delete(m.delegations, dk)
	return nil
}

// ValidateUnbondAmount returns the shares corresponding to amt using a 1:1 ratio.
// Returns an error if no delegation exists or the amount exceeds delegated shares.
func (m *MockStakingKeeper) ValidateUnbondAmount(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) (math.LegacyDec, error) {
	dk := delegationKey{Delegator: delAddr.String(), Validator: valAddr.String()}
	del, ok := m.delegations[dk]
	if !ok {
		return math.LegacyZeroDec(), stakingtypes.ErrNoDelegation
	}
	shares := math.LegacyNewDecFromInt(amt)
	if shares.GT(del.Shares) {
		return math.LegacyZeroDec(), stakingtypes.ErrNotEnoughDelegationShares
	}
	return shares, nil
}

func (m *MockStakingKeeper) BeginRedelegation(ctx context.Context, delAddr sdk.AccAddress, valSrcAddr, valDstAddr sdk.ValAddress, sharesAmount math.LegacyDec) (time.Time, error) {
	srcKey := delegationKey{Delegator: delAddr.String(), Validator: valSrcAddr.String()}
	srcDel, ok := m.delegations[srcKey]
	if !ok {
		return time.Time{}, stakingtypes.ErrNoDelegation
	}
	if sharesAmount.GT(srcDel.Shares) {
		return time.Time{}, stakingtypes.ErrNotEnoughDelegationShares
	}
	srcDel.Shares = srcDel.Shares.Sub(sharesAmount)
	if srcDel.Shares.IsZero() {
		delete(m.delegations, srcKey)
	} else {
		m.delegations[srcKey] = srcDel
	}
	dstKey := delegationKey{Delegator: delAddr.String(), Validator: valDstAddr.String()}
	dstDel, exists := m.delegations[dstKey]
	if !exists {
		dstDel = stakingtypes.Delegation{
			DelegatorAddress: delAddr.String(),
			ValidatorAddress: valDstAddr.String(),
			Shares:           math.LegacyZeroDec(),
		}
	}
	dstDel.Shares = dstDel.Shares.Add(sharesAmount)
	m.delegations[dstKey] = dstDel
	return time.Now().Add(21 * 24 * time.Hour), nil
}

func (m *MockStakingKeeper) BondDenom(ctx context.Context) (string, error) {
	return "stake", nil
}

// Delegate creates or adds to a delegation using a 1:1 token-to-share ratio.
func (m *MockStakingKeeper) Delegate(ctx context.Context, delAddr sdk.AccAddress, bondAmt math.Int, tokenSrc stakingtypes.BondStatus, validator stakingtypes.Validator, subtractAccount bool) (math.LegacyDec, error) {
	dk := delegationKey{Delegator: delAddr.String(), Validator: validator.OperatorAddress}
	del, exists := m.delegations[dk]
	if !exists {
		del = stakingtypes.Delegation{
			DelegatorAddress: delAddr.String(),
			ValidatorAddress: validator.OperatorAddress,
			Shares:           math.LegacyZeroDec(),
		}
	}
	newShares := math.LegacyNewDecFromInt(bondAmt)
	del.Shares = del.Shares.Add(newShares)
	m.delegations[dk] = del

	if val, ok := m.validators[validator.OperatorAddress]; ok {
		val.Tokens = val.Tokens.Add(bondAmt)
		val.DelegatorShares = val.DelegatorShares.Add(newShares)
		m.validators[validator.OperatorAddress] = val
	}
	return newShares, nil
}

// Undelegate removes shares from a delegation and creates an unbonding entry.
func (m *MockStakingKeeper) Undelegate(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, sharesAmount math.LegacyDec) (time.Time, math.Int, error) {
	dk := delegationKey{Delegator: delAddr.String(), Validator: valAddr.String()}
	del, ok := m.delegations[dk]
	if !ok {
		return time.Time{}, math.ZeroInt(), stakingtypes.ErrNoDelegation
	}
	if sharesAmount.GT(del.Shares) {
		return time.Time{}, math.ZeroInt(), stakingtypes.ErrNotEnoughDelegationShares
	}
	del.Shares = del.Shares.Sub(sharesAmount)
	if del.Shares.IsZero() {
		delete(m.delegations, dk)
	} else {
		m.delegations[dk] = del
	}

	returnAmount := sharesAmount.TruncateInt()
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	completionTime := sdkCtx.BlockTime().Add(21 * 24 * time.Hour)

	ubd, exists := m.unbondings[dk]
	if !exists {
		ubd = stakingtypes.UnbondingDelegation{
			DelegatorAddress: delAddr.String(),
			ValidatorAddress: valAddr.String(),
		}
	}
	id := m.nextUnbondID
	m.nextUnbondID++
	ubd.Entries = append(ubd.Entries, stakingtypes.UnbondingDelegationEntry{
		CreationHeight:          sdkCtx.BlockHeight(),
		CompletionTime:          completionTime,
		InitialBalance:          returnAmount,
		Balance:                 returnAmount,
		UnbondingId:             id,
		UnbondingOnHoldRefCount: 0,
	})
	m.unbondings[dk] = ubd

	return completionTime, returnAmount, nil
}

func (m *MockStakingKeeper) RemoveUnbondingDelegation(ctx context.Context, ubd stakingtypes.UnbondingDelegation) error {
	dk := delegationKey{Delegator: ubd.DelegatorAddress, Validator: ubd.ValidatorAddress}
	delete(m.unbondings, dk)
	return nil
}

func (m *MockStakingKeeper) SetUnbondingDelegation(ctx context.Context, ubd stakingtypes.UnbondingDelegation) error {
	dk := delegationKey{Delegator: ubd.DelegatorAddress, Validator: ubd.ValidatorAddress}
	m.unbondings[dk] = ubd
	return nil
}

func StructsKeeper(t testing.TB) (keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	appCodec := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Create mock keepers
	mockAccountKeeper := NewMockAccountKeeper()
	mockBankKeeper := NewMockBankKeeper()
	mockStakingKeeper := NewMockStakingKeeper()

	// IBC v10 - no capability keeper needed
	k := keeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		func() *ibckeeper.Keeper {
			return &ibckeeper.Keeper{}
		},
		mockBankKeeper,
		mockStakingKeeper,
		mockAccountKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}
