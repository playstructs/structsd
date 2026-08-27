package ante

import (
	"github.com/hashicorp/go-metrics"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
)

// observeReject is the single place where ante rejections become visible to
// operators. Every decorator that returns an error MUST funnel it through
// here so:
//
//  1. A structured log line lands in the validator's log with the codespace,
//     code, phase, message type, and signer (when known). This is what lets
//     a human operator follow up a "tx failed" support ticket.
//
//     Only a deliverTx rejection is logged at ERROR. A rejection during
//     checkTx or reCheckTx is admission control doing its job, and its volume
//     is set by whoever is sending: a rejected transaction advances no
//     sequence and consumes no throttle count, so the same correctly signed
//     over-cap transaction can be replayed for as long as somebody cares to,
//     one ERROR line each, at no fee - and CometBFT's check_tx RPC bypasses
//     the mempool cache that would otherwise deduplicate it. A deliverTx
//     rejection cannot be produced faster than blocks are produced, which is
//     what makes it safe to log unconditionally and worth reading: that
//     transaction got past admission and failed anyway.
//
//     The aggregate signal does not move. The counter below carries a phase
//     label, so "reject rate by code" and any alert built on it see checkTx
//     rejections exactly as before; what stops is one disk line per attempt.
//
//  2. A telemetry counter increments with the same labels, so a Grafana /
//     Prometheus dashboard can show "ante reject rate by code" and alert
//     when a particular code spikes (e.g. ErrDuplicateChargeInTx > 0 means
//     a buggy client just hit production). This fires in every phase and is
//     the durable signal; the log line is the per-transaction detail.
//
// The function returns the underlying error unchanged so callers can `return
// ctx, observeReject(ctx, err, ...)` in one line.
//
// Recurrence of incident 2026-05 should be visible within seconds via the
// `structs_ante_reject{code="2020"}` counter going non-zero. That detection
// path is the counter, not the log line, so the phase-based level above does
// not weaken it.
func observeReject(ctx sdk.Context, decorator string, err error, labels ...metrics.Label) error {
	if err == nil {
		return nil
	}

	codespace, code, _ := errors.ABCIInfo(err, false)

	// SKIP_RATIONALE: phase classification, not a phase skip. We need to
	// label log lines and telemetry by which phase the reject fired in.
	phase := "deliverTx"
	switch {
	case ctx.IsReCheckTx():
		phase = "reCheckTx"
	case ctx.IsCheckTx():
		phase = "checkTx"
	}

	logger := ctx.Logger().With("module", "ante")
	logReject := logger.Error
	if phase != "deliverTx" {
		// Attacker-paced and free to repeat; see the note above. The counter
		// below still fires, so nothing an operator alerts on is lost.
		logReject = logger.Debug
	}
	logReject("ante reject",
		"decorator", decorator,
		"phase", phase,
		"codespace", codespace,
		"code", code,
		"err", err.Error(),
	)

	allLabels := append([]metrics.Label{
		telemetry.NewLabel("decorator", decorator),
		telemetry.NewLabel("phase", phase),
		telemetry.NewLabel("codespace", codespace),
		telemetry.NewLabel("code", uintToStr(code)),
	}, labels...)
	telemetry.IncrCounterWithLabels(
		[]string{"structs", "ante", "reject"},
		1,
		allLabels,
	)

	return err
}

// observeAccept records a successful ante traversal for a particular decorator.
// This is a low-frequency counter (one per tx that reaches the next decorator)
// and gives a denominator for reject-rate dashboards.
func observeAccept(ctx sdk.Context, decorator string) {
	// SKIP_RATIONALE: phase classification, not a phase skip; see observeReject.
	phase := "deliverTx"
	switch {
	case ctx.IsReCheckTx():
		phase = "reCheckTx"
	case ctx.IsCheckTx():
		phase = "checkTx"
	}
	telemetry.IncrCounterWithLabels(
		[]string{"structs", "ante", "accept"},
		1,
		[]metrics.Label{
			telemetry.NewLabel("decorator", decorator),
			telemetry.NewLabel("phase", phase),
		},
	)
}

// uintToStr renders a small uint32 (ABCI error code) without pulling strconv.
func uintToStr(n uint32) string {
	if n == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
