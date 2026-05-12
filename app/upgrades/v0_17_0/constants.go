package v0_17_0

// UpgradeName is the on-chain upgrade plan name for the v0.17.0 binary, which
// hardens the ante chain in response to incident 2026-05 (see
// docs/incident-2026-05-ante.md):
//
//   - ThrottleDecorator and StakingThrottleDecorator no longer skip CheckTx /
//     ReCheckTx, so transactions that would fail at DeliverTx now also fail at
//     mempool admission and are evicted via ReCheckTx.
//   - Per-tx in-memory deduplication catches duplicate charge / proof / object-
//     throttle keys in a single tx without relying on consensus state.
//   - MsgStructActivate now calls playerCache.Discharge(), aligning the
//     handler with its ChargeMessages registration.
//   - Typed errors in the structs-ante codespace replace string-formatted
//     errors so clients can switch on ABCI codes.
//
// No store-key changes; this is a pure handler/ante upgrade.
const UpgradeName = "v0.17.0"
