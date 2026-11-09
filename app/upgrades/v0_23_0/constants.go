package v0_23_0

// UpgradeName is the on-chain plan name for the v0.23.0 development release.
// Adds an optional per-connection capacity maximum to substations, stored in
// the grid and managed through a PermUpdate transaction. Connection shares are
// clamped after the existing calculation and written only when changed.
// Missing/zero maximum means unlimited, so existing state needs no backfill or
// store-key upgrade. Changed transaction validity, gas and events require the
// coordinated v0.23.0 binary switch.
// Dependency security fixes already present in v0.22.0 are retained.
const UpgradeName = "v0.23.0"
