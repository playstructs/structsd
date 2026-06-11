package v0_18_0

// UpgradeName is the on-chain upgrade plan name for the v0.18.0 binary,
// which reworks planet raid mechanics around the defending Command Ship
// (the "SHIELDS_VULNERABLE" raid phase):
//
// Consensus / handler changes (binary):
//
//   - MsgPlanetRaidComplete now rejects proofs while the defending player's
//     Command Ship is online. The raid hashing puzzle can only be won while
//     the defender's Command Ship is offline, destroyed, or non-existent.
//   - blockStartRaid is no longer set on raider arrival alone. While a raid
//     is in progress it tracks the Command Ship vulnerability window: set
//     when the defending Command Ship goes offline (or when raiders arrive
//     to find it already down), cleared when it comes back online and when
//     the raid ends.
//   - New raidStatus enum value `shieldsVulnerable` (6) is emitted when the
//     raid becomes winnable; the previously unused `ongoing` (2) is emitted
//     when the defending Command Ship comes back online mid-raid.
//
// Battleship vs. Tank balancing (binary + struct-type rewrite):
//
//   - New per-weapon armour-piercing tech: struct types gain
//     primaryWeaponArmourPiercing / secondaryWeaponArmourPiercing booleans
//     (default false). An armour-piercing weapon negates the target's
//     attackReduction during volley damage resolution; attack events report
//     the piercing via EventAttackShotDetail.armourPiercing.
//   - Battleship primary (unguided) is now armour piercing and restricted
//     to land + water ambits (22 -> 6). Battleship gains a guided secondary
//     weapon targeting space (charge 5, damage 1).
//
// Planetary struct health (binary + struct-type rewrite + state migration):
//
//   - Every planetary struct type's MaxHealth is raised from 3: the
//     baseline planetary structs (Ore Extractor, Ore Refinery, Orbital
//     Shield Generator, Jamming Satellite, Ore Bunker, Planetary Defense
//     Cannon) go to 6. Fleet struct types are unchanged.
//   - The three power generators are hardened further so disrupting power
//     is a deliberate, costly raid objective rather than incidental
//     collateral: Field Generator 3 -> 8, Continental Power Plant 3 -> 10,
//     World Engine 3 -> 10, each gaining armour unit defenses and
//     attackReduction 1 (the same mitigation the Tank already uses;
//     armour-piercing weapons bypass it).
//   - MigratePlanetaryStructHealth sets every built, non-destroyed
//     planetary struct's stored health to its new MaxHealth at upgrade
//     height. Health is only stamped at materialization, so without this
//     migration pre-upgrade planetary structs would read as permanently
//     damaged (old 3 against the new maxima) with no repair mechanism.
//
// Difficulty rebase (state migration at upgrade height):
//
//   - Planetary shield values are rebased from hour/day-scale targets to
//     fractions of the Command Ship rebuild window (BuildDifficulty: 200):
//     base 1500 -> 25, Orbital Shield Generator 1500 -> 25, Jamming
//     Satellite 4500 -> 12, Ore Bunker 9000 -> 50, Planetary Defense
//     Cannon 4500 -> 13.
//   - All struct types are rewritten from CreateStructTypeGenesis() and
//     every planet's planetaryShield attribute is recomputed as the new
//     base plus the new contributions of its online defense structs.
//   - Every planet's blockStartRaid attribute is normalized: set to the
//     upgrade height where a raid is in progress and the defending Command
//     Ship is vulnerable, cleared to zero otherwise. Stale pre-upgrade
//     values would otherwise collapse the puzzle difficulty to trivial
//     (age is measured from blockStartRaid) or anchor proofs to inputs
//     that no longer match the new semantics.
//
// Charge rebalance (struct-type rewrite):
//
//   - Action charges are retuned across all struct types: activateCharge
//     1 -> 2, and the Command Ship's moveCharge 8 -> 3. stealthActivateCharge
//     goes 1 -> 2 on the only stealth-capable types (Stealth Bomber,
//     Submersible); defendChangeCharge (1) and buildCharge (8) are unchanged.
//   - Weapon charges are flattened toward a 3/5 cadence so secondary and
//     low-charge weapons stay relevant: primaries are 3 (Command Ship,
//     Starfighter, Pursuit Fighter, Tank) or 5 (Battleship, Frigate, Stealth
//     Bomber, High Altitude Interceptor, Mobile Artillery, SAM Launcher,
//     Cruiser, Destroyer, Submersible); secondaries are 5 (Battleship,
//     Starfighter) or 3 (Cruiser).
//   - Applied via the CreateStructTypeGenesis() rewrite; no extra migration.
//
// No store-key changes; all migrations write to existing prefix stores.
const UpgradeName = "v0.18.0"
