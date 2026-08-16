package v0_21_0

// UpgradeName is the on-chain upgrade plan name for the v0.21.0 binary.
//
// v0.21.0 combines guild-bank conversion and charter features with consensus
// and state repairs discovered during the release review. The binary changes:
//
//   - add guild-token conversion, conversion fees, integer redemption math, and
//     holder restrictions for clawback-enabled guild tokens;
//   - introduce proof-priced guild charters and one-time reactor entitlements;
//   - reconcile reactor energy with staking, delegation ownership, and validator
//     health;
//   - harden agreement settlement, checkpoint ordering, expiry, capacity
//     validation, and provider-pool accounting;
//   - reject destroyed structs, invalid guild policy enums, phantom player ids,
//     and membership transitions without a pending application;
//   - make cache commits and Unicode-sensitive validation deterministic; and
//   - authorize object-global ante throttle reservations before writing them.
//
// The upgrade handler backfills all state derived from those rules. Its order is
// load-bearing: provider pool indices precede guild-token transfers, infusion
// ownership precedes staking reconciliation, agreement accounting repairs
// precede overdue expiry, and the auto-resize index rebuild runs last.
//
// See docs/upgrades/v0.21.0.md for the detailed operator and migration notes.
const UpgradeName = "v0.21.0"
