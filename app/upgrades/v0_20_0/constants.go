package v0_20_0

// UpgradeName is the on-chain upgrade plan name for the v0.20.0 binary.
//
// Consensus / handler changes (binary):
//
//   - Combat defender resolution no longer lets a destroyed attacker land
//     blocked damage. ResolveDefenders now resolves every defender counter
//     before any block: defender counters are simultaneous from the attacker's
//     perspective, so if their combined damage destroys the attacker the attack
//     is fully neutralized and no block volley may land. Previously the counter
//     and block were interleaved per defender in struct-ID iteration order, so
//     a blocker that sorted ahead of the lethal counter unit would absorb the
//     attacker's volley (taking blocker damage) even though a later defender's
//     counter destroyed the attacker in the same action - an "attack from the
//     dead".
//
//   - StructDeactivate no longer rejects players who are offline due to power
//     overload. Players can deactivate online structs to reduce structsLoad and
//     return to power compliance.
//
//   - New StructDeactivateBatch message deactivates up to 65 structs in a
//     single transaction. All structs are validated (existence, PermPlay,
//     built, online) before any are taken offline, so the batch is atomic:
//     an ineligible struct rejects the entire message with no partial changes.
//
//   - MsgAllocationUpdate no longer double-counts the allocation's own power
//     in the source capacity check. Dynamic allocations can grow into capacity
//     they already hold (e.g. after MsgReactorInfuse increases source capacity).
//
//   - The Jamming Satellite planetary defense (lowOrbitBallisticInterceptorNetwork)
//     now shields a planetary (non-fleet) struct on its own planet from guided
//     ordnance regardless of the source or target ambit. Its evasion runs off a
//     jamming/guidance disruption model, so unguided ordnance is unaffected.
//     Previously evasion required the attacker to be on a fleet and only applied
//     to an air/space source striking a land/water target - and it incorrectly
//     jammed unguided weapons in that geometry. The ambit constraints are
//     removed; the requirements are now planetary target + guided weapon.
//
//   - New StructTrash message destroys any non-destroyed struct (built or
//     still building) that the caller has PermPlay permission over. It mirrors
//     StructBuildCancel but gates on the owner having at least the struct type's
//     BuildCharge and consumes that charge (Discharge) on success. Attempting to
//     trash an already-destroyed struct returns a state error with no penalty.
//
// This upgrade is binary-only. There is no state migration: no struct type,
// planet attribute, or other persisted state is read or written by this
// upgrade, and there are no store-key changes. The new StructTrash message adds
// a transaction type but no new persisted state.
const UpgradeName = "v0.20.0"
