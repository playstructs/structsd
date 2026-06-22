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
//   - Struct attacks now require the target to have the Built status. A
//     struct that is materialized but not yet built can no longer be
//     attacked (online/offline is irrelevant); CanAttack rejects such
//     targets with the "unbuilt" targeting reason.
//   - The fleet Command Ship is no longer required (present or online) to
//     attack, set/clear defensive assignments, or toggle stealth. The
//     IsCommandable() gate was removed from those handlers; the acting
//     struct itself must still be online (and the owner powered). The
//     Command Ship is still required to build new structs and to move the
//     fleet to another planet. Note: this means destroying or disabling an
//     enemy Command Ship no longer suppresses that fleet's weapons,
//     defenses, or stealth - only its ability to build and to move. A
//     Command Ship destroyed away from home still auto-returns the fleet
//     (raid defeat), so the only "stuck but still able to fight" case is a
//     Command Ship that goes offline (not destroyed) while away.
//
// Struct-type rebalance (binary + struct-type rewrite):
//
//   - The Battleship's secondary weapon damage is raised from 1 to 2.
//   - All struct types are rewritten from CreateStructTypeGenesis() at
//     upgrade height (MigrateStructTypes). Struct types are persisted in
//     state, so the genesis change only reaches an already-live chain
//     through this rewrite.
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
// No store-key changes; the migrations write to the existing struct type
// and planet attribute prefix stores.
const UpgradeName = "v0.19.0"
