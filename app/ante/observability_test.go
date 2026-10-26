package ante

import (
	"bytes"
	"testing"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

/* Regression suite for unbounded rejection logging.
 *
 * Every ante rejection used to emit an ERROR line carrying the error text,
 * checkTx included. A rejected transaction advances no sequence and consumes no
 * throttle count, so the same correctly signed over-cap transaction can be
 * replayed indefinitely - one ERROR line each, at no fee, and CometBFT's
 * check_tx RPC bypasses the mempool cache that would otherwise deduplicate it.
 * The volume was set by whoever was sending.
 *
 * A deliverTx rejection is different in kind: it cannot be produced faster than
 * blocks are, and it means a transaction got past admission and failed anyway,
 * which is worth an operator's attention. That is where ERROR stays.
 */

// rejectAt runs a rejection through the ante observability helper with a logger
// that only records at or above the given level, and returns what was written.
func rejectAt(t *testing.T, level zerolog.Level, mod func(sdk.Context) sdk.Context) string {
	t.Helper()

	var buf bytes.Buffer
	logger := log.NewLogger(&buf, log.LevelOption(level))

	ctx := mod(sdk.NewContext(nil, cmtproto.Header{Height: 100}, false, logger))

	observeReject(ctx, "TestDecorator", ErrPlayerMsgCapExceeded)

	return buf.String()
}

func TestObserveReject_CheckTxIsNotLoggedAtError(t *testing.T) {
	out := rejectAt(t, zerolog.ErrorLevel, func(c sdk.Context) sdk.Context { return c.WithIsCheckTx(true) })
	require.Empty(t, out,
		"a checkTx rejection is attacker-paced and free to repeat; it must not write a line an operator's ERROR log keeps")

	out = rejectAt(t, zerolog.ErrorLevel, func(c sdk.Context) sdk.Context { return c.WithIsReCheckTx(true) })
	require.Empty(t, out, "the same holds for reCheckTx")
}

// The detail is still available to an operator who asks for it.
func TestObserveReject_CheckTxIsStillLoggedAtDebug(t *testing.T) {
	out := rejectAt(t, zerolog.DebugLevel, func(c sdk.Context) sdk.Context { return c.WithIsCheckTx(true) })
	require.Contains(t, out, "ante reject")
	require.Contains(t, out, "checkTx")
}

/* A deliverTx rejection stays at ERROR. It is bounded by block production and
 * means the transaction passed admission and failed anyway, which is the case
 * worth reading.
 */
func TestObserveReject_DeliverTxStaysAtError(t *testing.T) {
	out := rejectAt(t, zerolog.ErrorLevel, func(c sdk.Context) sdk.Context { return c })
	require.Contains(t, out, "ante reject")
	require.Contains(t, out, "deliverTx")
}
