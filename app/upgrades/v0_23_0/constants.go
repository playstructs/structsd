package v0_23_0

// UpgradeName is the on-chain plan name for the v0.23.0 development release.
// Adds an optional per-connection capacity maximum to substations, stored in
// the grid and managed through a PermUpdate transaction. Connection shares are
// clamped after the existing calculation and written only when changed.
// Missing/zero maximum means unlimited, so that feature needs no backfill.
// Planets gain a sector, drawn when the planet is created; the upgrade handler
// backfills one for every existing planet. The ante now also rejects Structs
// messages wrapped in group proposals, including proposals stored before the
// upgrade. No store keys are added. Changed transaction validity, gas and
// events require the coordinated v0.23.0 binary switch.
// Dependency security fixes already present in v0.22.0 are retained.
const UpgradeName = "v0.23.0"
