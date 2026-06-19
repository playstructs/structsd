package v0_19_0

// UpgradeName is the on-chain upgrade plan name for the v0.19.0 binary,
// which corrects the v0.18.0 raid "shieldsVulnerable" mechanic so that a
// defending fleet's location is taken into account.
//
// Consensus / handler changes (binary):
//
//   - IsDefenderCommandStructVulnerable now also reports vulnerable when the
//     defending fleet is not on station. The Command Ship travels with the
//     fleet, so a defender who moves their fleet away (e.g. to raid
//     elsewhere) leaves their home planet undefended even with an online
//     Command Ship. Previously such a home planet read as "shields up" and
//     an attacker could never win the raid.
//   - A fleet move now refreshes the raid state of the moving fleet owner's
//     home planet: moving the Command Ship away mid-raid sets the
//     vulnerability window (shieldsVulnerable), and returning with an online
//     Command Ship restores shields (ongoing) and clears the window.
//   - The Command Ship online/offline raid hook recomputes the full
//     vulnerability predicate rather than trusting the online flag, so a
//     Command Ship coming back online while the fleet is still away no
//     longer wrongly restores shields.
//
// State migration at upgrade height:
//
//   - MigrateAwayDefenderRaidClock anchors blockStartRaid to the upgrade
//     height for every in-progress raid whose defender is now vulnerable
//     under the broadened predicate but whose clock is still zero (the
//     defender-away case that v0.18.0 cleared). Without this, those raids
//     would read as vulnerable yet remain uncompletable (raid_clock_unset)
//     until some unrelated event restarted the clock.
//
// No store-key changes; the migration writes to the existing planet
// attribute prefix store.
const UpgradeName = "v0.19.0"
