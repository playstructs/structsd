package keeper

import (
	"context"
	"encoding/binary"
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

	// nextAccountNumber mirrors the real keeper's global monotonic sequence.
	// Without it every account would be numbered 0 and the mock would hide the
	// very ordering bug that map-order commits used to cause.
	nextAccountNumber uint64

	// creationOrder records the addresses passed to NewAccountWithAddress, in
	// call order, so a test can assert the order state was written in and not
	// just the numbers that came out of it.
	creationOrder []string
}

func NewMockAccountKeeper() *MockAccountKeeper {
	return &MockAccountKeeper{
		accounts: make(map[string]sdk.AccountI),
	}
}

// AccountCreationOrder returns the addresses given account numbers, in the
// order they were assigned.
func (m *MockAccountKeeper) AccountCreationOrder() []string {
	return append([]string(nil), m.creationOrder...)
}

func (m *MockAccountKeeper) GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI {
	return m.accounts[addr.String()]
}

func (m *MockAccountKeeper) SetAccount(ctx context.Context, acc sdk.AccountI) {
	m.accounts[acc.GetAddress().String()] = acc
}

func (m *MockAccountKeeper) NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI {
	accountNumber := m.nextAccountNumber
	m.nextAccountNumber++
	m.creationOrder = append(m.creationOrder, addr.String())

	acc := authtypes.NewBaseAccount(addr, nil, accountNumber, 0)
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

// MockBankState is a deep copy of everything MockBankKeeper tracks.
type MockBankState struct {
	balances map[string]sdk.Coins
	metadata map[string]banktypes.Metadata
	supply   map[string]math.Int
}

// Snapshot copies the mock's current state so Restore can put it back.
//
// The mock keeps balances and supply in plain Go maps and ignores the ctx it is
// handed, so sdk.Context.CacheContext() does not isolate bank state the way it
// isolates KV state — a branch that mints or sends is writing to the same maps
// the parent reads. A test that needs several independent bank scenarios must
// snapshot before each and restore after, or the second scenario starts on top
// of the first one's balances.
func (m *MockBankKeeper) Snapshot() MockBankState {
	snapshot := MockBankState{
		balances: make(map[string]sdk.Coins, len(m.balances)),
		metadata: make(map[string]banktypes.Metadata, len(m.metadata)),
		supply:   make(map[string]math.Int, len(m.supply)),
	}
	for addr, coins := range m.balances {
		// sdk.Coins is a slice; copy it so later Add/Sub cannot write through.
		snapshot.balances[addr] = append(sdk.Coins{}, coins...)
	}
	for denom, meta := range m.metadata {
		snapshot.metadata[denom] = meta
	}
	for denom, amount := range m.supply {
		snapshot.supply[denom] = amount
	}
	return snapshot
}

// Restore returns the mock to a state captured by Snapshot.
func (m *MockBankKeeper) Restore(snapshot MockBankState) {
	m.balances = make(map[string]sdk.Coins, len(snapshot.balances))
	m.metadata = make(map[string]banktypes.Metadata, len(snapshot.metadata))
	m.supply = make(map[string]math.Int, len(snapshot.supply))
	for addr, coins := range snapshot.balances {
		m.balances[addr] = append(sdk.Coins{}, coins...)
	}
	for denom, meta := range snapshot.metadata {
		m.metadata[denom] = meta
	}
	for denom, amount := range snapshot.supply {
		m.supply[denom] = amount
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

// JailValidator marks an existing validator as jailed (and unbonded).
// Used by tests for the guild primary-reactor recovery handler.
func (m *MockStakingKeeper) JailValidator(operatorAddr sdk.ValAddress) {
	val, ok := m.validators[operatorAddr.String()]
	if !ok {
		return
	}
	val.Jailed = true
	val.Status = stakingtypes.Unbonded
	m.validators[operatorAddr.String()] = val
}

// UnjailValidator clears the jail flag but leaves the bond status alone, which
// is exactly what MsgUnjail does: rebonding is a separate decision staking makes
// in EndBlock, and only if the validator is inside the active-set cutoff. Pair
// with BondValidator to model the full recovery. Used by tests for the reactor
// energy recovery path.
func (m *MockStakingKeeper) UnjailValidator(operatorAddr sdk.ValAddress) {
	val, ok := m.validators[operatorAddr.String()]
	if !ok {
		return
	}
	val.Jailed = false
	m.validators[operatorAddr.String()] = val
}

// BondValidator returns a validator to the bonded set, modelling the rebond that
// staking performs in EndBlock for an unjailed validator inside the active set.
func (m *MockStakingKeeper) BondValidator(operatorAddr sdk.ValAddress) {
	val, ok := m.validators[operatorAddr.String()]
	if !ok {
		return
	}
	val.Status = stakingtypes.Bonded
	m.validators[operatorAddr.String()] = val
}

// SlashValidatorTokens reduces a validator's token pool while leaving its
// delegator shares alone, which is how Cosmos slashing devalues each share.
// Used by tests that check energy returns proportionally lower after a slash.
func (m *MockStakingKeeper) SlashValidatorTokens(operatorAddr sdk.ValAddress, remaining math.Int) {
	val, ok := m.validators[operatorAddr.String()]
	if !ok {
		return
	}
	val.Tokens = remaining
	m.validators[operatorAddr.String()] = val
}

// RemoveValidator deletes a validator from the mock. Used by tests that
// simulate a permanently retired validator (the recovery scenario for
// MsgGuildUpdatePrimaryReactor).
func (m *MockStakingKeeper) RemoveValidator(operatorAddr sdk.ValAddress) {
	delete(m.validators, operatorAddr.String())
}

// MatureUnbondingDelegation removes the UBD record for a (delegator, validator)
// pair to simulate Cosmos SDK's silent EndBlocker maturity completion. Used
// by tests for the structs-side maturity-sweep mechanism.
func (m *MockStakingKeeper) MatureUnbondingDelegation(delAddr sdk.AccAddress, valAddr sdk.ValAddress) {
	dk := delegationKey{Delegator: delAddr.String(), Validator: valAddr.String()}
	delete(m.unbondings, dk)
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
	transientStoreKey := storetypes.NewTransientStoreKey(types.TStoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	stateStore.MountStoreWithDB(transientStoreKey, storetypes.StoreTypeTransient, nil)
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
		runtime.NewTransientStoreService(transientStoreKey),
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

	// Stash the IAVL store key on the context so downstream test helpers
	// (e.g. WriteRawGridAttribute) can punch raw KV writes through the
	// keeper's exported surface to simulate corrupted on-chain state.
	ctx = ctx.WithValue(testStoreKeyCtx{}, storeKey)
	ctx = ctx.WithValue(testAccountKeeperCtx{}, mockAccountKeeper)

	return k, ctx
}

// testStoreKeyCtx is the unexported context key used to stash the IAVL store
// key produced by StructsKeeper so test helpers can reach the underlying KV
// store without changing the public StructsKeeper return signature.
type testStoreKeyCtx struct{}

// testAccountKeeperCtx stashes the MockAccountKeeper for the same reason.
type testAccountKeeperCtx struct{}

// AccountKeeperFrom returns the MockAccountKeeper behind a context produced by
// StructsKeeper, so a test can inspect how auth accounts were numbered.
func AccountKeeperFrom(t testing.TB, ctx sdk.Context) *MockAccountKeeper {
	t.Helper()
	accountKeeper, ok := ctx.Value(testAccountKeeperCtx{}).(*MockAccountKeeper)
	require.True(t, ok, "AccountKeeperFrom: ctx not produced by keepertest.StructsKeeper")
	return accountKeeper
}

// WriteRawGridAttribute plants a raw GridAttribute KV row, bypassing
// Keeper.SetGridAttribute. This exists only so tests can simulate the
// pre-v0.17.0 testnet "2-" orphan documented in
// docs/incident-2026-05-grid-orphan.md (which the SetGridAttribute backstop
// would otherwise refuse to write). Production code must never use this.
func WriteRawGridAttribute(t testing.TB, _ keeper.Keeper, ctx sdk.Context, gridAttributeId string, value uint64) {
	t.Helper()
	storeKey, ok := ctx.Value(testStoreKeyCtx{}).(*storetypes.KVStoreKey)
	require.True(t, ok, "WriteRawGridAttribute: ctx not produced by keepertest.StructsKeeper")

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, value)

	rawStore := ctx.KVStore(storeKey)
	rawStore.Set(append([]byte(types.GridAttributeKey), []byte(gridAttributeId)...), bz)
}

// WriteRawGuild plants protobuf bytes directly under a guild key. It is used to
// exercise migrations and queries against records encoded by pre-upgrade
// schemas that cannot be produced through the current keeper.
func WriteRawGuild(t testing.TB, ctx sdk.Context, guildID string, bz []byte) {
	t.Helper()
	storeKey, ok := ctx.Value(testStoreKeyCtx{}).(*storetypes.KVStoreKey)
	require.True(t, ok, "WriteRawGuild: ctx not produced by keepertest.StructsKeeper")

	rawStore := ctx.KVStore(storeKey)
	rawStore.Set(append([]byte(types.GuildKey), []byte(guildID)...), bz)
}
