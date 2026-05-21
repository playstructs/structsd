package v0_17_0

// UpgradeName is the on-chain upgrade plan name for the v0.17.0 binary, which
// hardens the ante chain in response to incident 2026-05 (see
// docs/incident-2026-05-ante.md), recovers from the stuck-defusing
// reactor-infusion incident (see docs/incident-2026-05-defusing.md), and
// prunes the orphan grid-attribute row left behind by the pre-daac34c
// AutoResizeAllocation bug (see docs/incident-2026-05-grid-orphan.md):
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
// Malformed grid-attribute pruning (state migration at upgrade height):
//
//   - One-time walk of the GridAttribute store deletes every row whose key
//     fails IsValidGridAttributeID — i.e. anything that is not the
//     "<prefix>-<non-empty objectId>" shape. On structstestnet-111 this
//     prunes the single "2-" row at value 840000 left behind by the
//     pre-daac34c AutoResizeAllocation bug (which wrote capacity for an
//     empty DestinationId).
//   - Going forward, Keeper.SetGridAttribute itself rejects any write
//     against a malformed id with a logged error and a no-op, so the same
//     leak class cannot resurface even if a future caller forgets the
//     destination guard.
//
// No store-key changes required for these recoveries; all three write to
// existing prefix stores.
const UpgradeName = "v0.17.0"
