package v0_17_0

// UpgradeName is the on-chain upgrade plan name for the v0.17.0 binary, which
// hardens the ante chain in response to incident 2026-05 (see
// docs/incident-2026-05-ante.md) and recovers from the stuck-defusing
// reactor-infusion incident (see docs/incident-2026-05-defusing.md):
//
// Ante / handler hardening:
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
// Defusing-infusion recovery (state migration at upgrade height):
//
//   - One-time scan of every reactor-type Infusion: any record with Defusing
//     > 0 is reconciled against the live Cosmos staking state, clearing
//     stale "defusing" amounts left over from UBDs that already matured
//     silently in x/staking's EndBlocker (Cosmos SDK v0.53 fires no
//     completion hook).
//   - Bootstraps the new InfusionMaturitySweepQueue from each defusing
//     infusion's still-live UBD entries so the new EndBlocker sweep
//     keeps clearing them as they mature post-upgrade.
//
// Going forward, AfterUnbondingInitiated enqueues maturity-sweep rows and
// the structs EndBlocker drains them, so this same class of stale state
// cannot accumulate again.
//
// No store-key changes required for these recoveries; both write to
// existing prefix stores.
const UpgradeName = "v0.17.0"
