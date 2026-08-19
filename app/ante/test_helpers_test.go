package ante_test

import (
	"context"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"cosmossdk.io/log"

	"structs/x/structs/types"
)

var _ = types.ModuleName // ensure import

// newTestCtx returns a properly initialized sdk.Context for decorator tests.
func newTestCtx() sdk.Context {
	return sdk.NewContext(nil, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
}

// mockAnteKeeper implements StructsAnteKeeper for testing.
type mockAnteKeeper struct {
	playerIndexes map[string]uint64
	permissions   map[string]types.Permission
	gridAttrs     map[string]uint64
	msgCounts     map[string]uint64
	throttleKeys  map[string]bool
	// throttleAuthDenied is a deny-list keyed by "<creator>|<targetId>", so the
	// default is authorized and a test only has to name the pairs it wants
	// refused. The real keeper answers from object ownership and delegation,
	// which is covered against a live keeper in x/structs/keeper.
	throttleAuthDenied map[string]bool
	// throttleAuthDenyFn refuses a whole class of targets at once, for tests
	// that generate the ids they are checking rather than naming them.
	throttleAuthDenyFn func(creator, targetId string) bool
	hasTransientStore  bool
	// incrementOrder records the order IncrementPlayerMsgCount was called in,
	// so a test can assert the cap loop iterates players deterministically.
	incrementOrder []string
}

func newMockAnteKeeper() *mockAnteKeeper {
	return &mockAnteKeeper{
		playerIndexes:      make(map[string]uint64),
		permissions:        make(map[string]types.Permission),
		gridAttrs:          make(map[string]uint64),
		msgCounts:          make(map[string]uint64),
		throttleKeys:       make(map[string]bool),
		throttleAuthDenied: make(map[string]bool),
		hasTransientStore:  true,
	}
}

// denyThrottleTarget makes the keeper refuse creator's authorization over
// targetId, the way a real keeper refuses somebody else's object.
func (m *mockAnteKeeper) denyThrottleTarget(creator, targetId string) {
	m.throttleAuthDenied[creator+"|"+targetId] = true
}

func (m *mockAnteKeeper) HasTransientStore() bool {
	return m.hasTransientStore
}

func (m *mockAnteKeeper) GetPlayerIndexFromAddress(_ context.Context, address string) uint64 {
	return m.playerIndexes[address]
}

func (m *mockAnteKeeper) GetPermissionsByBytes(_ context.Context, permissionId []byte) types.Permission {
	return m.permissions[string(permissionId)]
}

func (m *mockAnteKeeper) GetGridAttribute(_ context.Context, gridAttributeId string) uint64 {
	return m.gridAttrs[gridAttributeId]
}

func (m *mockAnteKeeper) IncrementPlayerMsgCount(_ context.Context, playerId string, delta uint64) uint64 {
	m.incrementOrder = append(m.incrementOrder, playerId)
	m.msgCounts[playerId] += delta
	return m.msgCounts[playerId]
}

func (m *mockAnteKeeper) GetPlayerMsgCount(_ context.Context, playerId string) uint64 {
	return m.msgCounts[playerId]
}

func (m *mockAnteKeeper) HasThrottleKey(_ context.Context, throttleKey string) bool {
	return m.throttleKeys[throttleKey]
}

func (m *mockAnteKeeper) SetThrottleKey(_ context.Context, throttleKey string) {
	m.throttleKeys[throttleKey] = true
}

func (m *mockAnteKeeper) ThrottleTargetAuthorized(_ context.Context, creator string, _ types.ObjectType, targetId string, _ types.Permission) bool {
	if m.throttleAuthDenyFn != nil && m.throttleAuthDenyFn(creator, targetId) {
		return false
	}
	return !m.throttleAuthDenied[creator+"|"+targetId]
}

// mockTx implements sdk.Tx for testing.
type mockTx struct {
	msgs    []sdk.Msg
	txBytes []byte
}

func (t mockTx) GetMsgs() []sdk.Msg                                    { return t.msgs }
func (t mockTx) GetMsgsV2() ([]proto.Message, error)                    { return nil, nil }
func (t mockTx) ValidateBasic() error                                   { return nil }

// mockMsg implements sdk.Msg with a configurable type URL.
type mockMsg struct {
	typeURL string
	creator string
}

func (m mockMsg) ProtoMessage()                     {}
func (m mockMsg) Reset()                            {}
func (m mockMsg) String() string                    { return m.typeURL }
func (m mockMsg) ValidateBasic() error              { return nil }

// identityHandler is a terminal ante handler that records it was called.
func identityHandler() (sdk.AnteHandler, *bool) {
	called := false
	h := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		called = true
		return ctx, nil
	}
	return h, &called
}
