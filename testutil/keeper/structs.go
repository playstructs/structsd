package keeper

import (
	"context"
	"testing"

	"cosmossdk.io/core/address"
	"cosmossdk.io/log"
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
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	portkeeper "github.com/cosmos/ibc-go/v8/modules/core/05-port/keeper"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
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
	return sdk.AccAddress{}
}

// MockBankKeeper is a mock implementation of the BankKeeper interface
type MockBankKeeper struct {
	balances map[string]sdk.Coins
	metadata map[string]banktypes.Metadata
}

func NewMockBankKeeper() *MockBankKeeper {
	return &MockBankKeeper{
		balances: make(map[string]sdk.Coins),
		metadata: make(map[string]banktypes.Metadata),
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
	return sdk.Coin{}
}

func (m *MockBankKeeper) HasBalance(ctx context.Context, addr sdk.AccAddress, coin sdk.Coin) bool {
	return true
}

func (m *MockBankKeeper) SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	return sdk.Coins{}
}

func (m *MockBankKeeper) SpendableCoin(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return sdk.Coin{}
}

func (m *MockBankKeeper) SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (m *MockBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	return nil
}

func (m *MockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return nil
}

func (m *MockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (m *MockBankKeeper) MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

func (m *MockBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

// MockStakingKeeper is a mock implementation of the StakingKeeper interface
type MockStakingKeeper struct{}

func NewMockStakingKeeper() *MockStakingKeeper {
	return &MockStakingKeeper{}
}

func (m *MockStakingKeeper) ConsensusAddressCodec() address.Codec {
	return nil
}

func (m *MockStakingKeeper) ValidatorByConsAddr(ctx context.Context, consAddr sdk.ConsAddress) (stakingtypes.ValidatorI, error) {
	return nil, nil
}

func (m *MockStakingKeeper) GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error) {
	return stakingtypes.Validator{}, nil
}

func (m *MockStakingKeeper) GetAllValidators(ctx context.Context) ([]stakingtypes.Validator, error) {
	return nil, nil
}

func (m *MockStakingKeeper) GetValidators(ctx context.Context, maxRetrieve uint32) ([]stakingtypes.Validator, error) {
	return nil, nil
}

func (m *MockStakingKeeper) GetValidatorDelegations(ctx context.Context, valAddr sdk.ValAddress) ([]stakingtypes.Delegation, error) {
	return nil, nil
}

func (m *MockStakingKeeper) GetDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (stakingtypes.Delegation, error) {
	return stakingtypes.Delegation{}, nil
}

func (m *MockStakingKeeper) GetUnbondingDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (stakingtypes.UnbondingDelegation, error) {
	return stakingtypes.UnbondingDelegation{}, nil
}

func (m *MockStakingKeeper) GetUnbondingDelegationByUnbondingID(ctx context.Context, id uint64) (stakingtypes.UnbondingDelegation, error) {
	return stakingtypes.UnbondingDelegation{}, nil
}

func (m *MockStakingKeeper) GetDelegatorDelegations(ctx context.Context, delegator sdk.AccAddress, maxRetrieve uint16) ([]stakingtypes.Delegation, error) {
	return nil, nil
}

func (m *MockStakingKeeper) SetDelegation(ctx context.Context, delegation stakingtypes.Delegation) error {
	return nil
}

func (m *MockStakingKeeper) RemoveDelegation(ctx context.Context, delegation stakingtypes.Delegation) error {
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
	capabilityKeeper := capabilitykeeper.NewKeeper(appCodec, storeKey, memStoreKey)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	scopedKeeper := capabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	portKeeper := portkeeper.NewKeeper(scopedKeeper)
	scopeModule := capabilityKeeper.ScopeToModule(types.ModuleName)

	// Create mock keepers
	mockAccountKeeper := NewMockAccountKeeper()
	mockBankKeeper := NewMockBankKeeper()
	mockStakingKeeper := NewMockStakingKeeper()

	k := keeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		func() *ibckeeper.Keeper {
			return &ibckeeper.Keeper{
				PortKeeper: &portKeeper,
			}
		},
		func(string) capabilitykeeper.ScopedKeeper {
			return scopeModule
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
