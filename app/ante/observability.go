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
//  2. A telemetry counter increments with the same labels, so a Grafana /
//     Prometheus dashboard can show "ante reject rate by code" and alert
//     when a particular code spikes (e.g. ErrDuplicateChargeInTx > 0 means
//     a buggy client just hit production).
//
// The function returns the underlying error unchanged so callers can `return
// ctx, observeReject(ctx, err, ...)` in one line.
//
// Recurrence of incident 2026-05 should be visible within seconds via the
// `structs_ante_reject{code="2020"}` counter going non-zero.
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
	logger.Error("ante reject",
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
