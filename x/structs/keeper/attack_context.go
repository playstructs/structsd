package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/nethruster/go-fraction"

	"structs/x/structs/types"
)

type ShotOutcome struct {
	Hit    bool
	Damage uint64
}

type EvasionResult struct {
	Evaded          bool
	Cause           types.TechUnitDefenses
	PlanetaryEvaded bool
	PlanetaryCause  types.TechPlanetaryDefenses
}

type BlockResult struct {
	Blocked      bool
	Blocker      *StructCache
	HealthBefore uint64
	HealthMax    uint64
}

type PostDestructionResult struct {
	Damage            uint64
	AttackerDestroyed bool
	PassiveWeaponry   types.TechPassiveWeaponry
}

type CounterResult struct {
	Damage            uint64
	AttackerDestroyed bool
	IsTargetCounter   bool
	CounterStruct     *StructCache
	WeaponSystem      types.TechWeaponSystem
	WeaponControl     types.TechWeaponControl
	ActiveWeaponry    types.TechActiveWeaponry
	PassiveWeaponry   types.TechPassiveWeaponry
	PostDestruction   *PostDestructionResult
}

type VolleyResult struct {
	Shots                    []ShotOutcome
	RolledDamage             uint64
	DamageReduction          uint64
	UnitDefenses             types.TechUnitDefenses
	NetDamage                uint64
	IsBlocker                bool
	TargetOrBlockerDestroyed bool
	HealthAfter              uint64
	PostDestruction          *PostDestructionResult
}

type RecoilResult struct {
	Damage    uint64
	Destroyed bool
}

type PlanetaryDefenseResult struct {
	Damage    uint64
	Destroyed bool
}

type AttackContext struct {
	Attacker     *StructCache
	WeaponSystem types.TechWeaponSystem

	AttackDetail *types.EventAttackDetail

	Target           *StructCache
	Evasion          *EvasionResult
	Block            *BlockResult
	DefenderCounters []CounterResult
	Volley           *VolleyResult
	TargetCounter    *CounterResult
	Recoil           *RecoilResult
	PlanetaryDefense *PlanetaryDefenseResult

	TargetedPlanets     []*PlanetCache
	targetedPlanetsSeen map[string]bool
}

func NewAttackContext(cc *CurrentContext, attacker *StructCache, weaponSystem types.TechWeaponSystem) *AttackContext {
	ac := &AttackContext{
		Attacker:            attacker,
		WeaponSystem:        weaponSystem,
		AttackDetail:        types.CreateEventAttackDetail(),
		TargetedPlanets:     make([]*PlanetCache, 0),
		targetedPlanetsSeen: make(map[string]bool),
	}

	ac.AttackDetail.SetBaseDetails(
		attacker.GetOwnerId(),
		attacker.GetStructId(),
		attacker.GetTypeId(),
		attacker.GetStructType().Type,
		attacker.GetLocationType(),
		attacker.GetLocationId(),
		attacker.GetOperatingAmbit(),
		attacker.GetSlot(),
		weaponSystem,
		attacker.GetStructType().GetWeaponControl(weaponSystem),
		attacker.GetStructType().GetWeapon(weaponSystem),
	)
	ac.AttackDetail.AttackerHealthBefore = attacker.GetHealth()
	ac.AttackDetail.AttackerHealthMax = attacker.GetStructType().MaxHealth

	cc.Attack = ac
	return ac
}

func (ac *AttackContext) BeginShot(target *StructCache) {
	ac.Target = target
	ac.Evasion = nil
	ac.Block = nil
	ac.DefenderCounters = nil
	ac.Volley = nil
	ac.TargetCounter = nil
}

func (ac *AttackContext) ValidateTarget() error {
	return ac.Attacker.CanAttack(ac.Target, ac.WeaponSystem)
}

// resolveEvasion checks unit evasion and planetary defense evasion.
// Returns true if the attack was evaded.
func (ac *AttackContext) resolveEvasion() bool {
	target := ac.Target
	attacker := ac.Attacker

	result := &EvasionResult{}

	var successRate fraction.Fraction
	switch attacker.GetStructType().GetWeaponControl(ac.WeaponSystem) {
	case types.TechWeaponControl_guided:
		successRate = target.GetStructType().GetGuidedDefensiveSuccessRate()
	case types.TechWeaponControl_unguided:
		successRate = target.GetStructType().GetUnguidedDefensiveSuccessRate()
	}

	if successRate.Numerator() != int64(0) {
		result.Evaded = target.IsSuccessful(successRate)
	}

	if result.Evaded {
		result.Cause = target.GetStructType().UnitDefenses
	}

	if !result.Evaded {
		if attacker.GetLocationType() == types.ObjectType_fleet {
			if target.GetPlanet().GetOwnerId() == target.GetOwnerId() {
				planetaryRate, planetaryErr := target.GetPlanet().GetLowOrbitBallisticsInterceptorNetworkSuccessRate()
				if planetaryErr == nil {
					if (attacker.GetOperatingAmbit() == types.Ambit_air) || (attacker.GetOperatingAmbit() == types.Ambit_space) {
						if (target.GetOperatingAmbit() == types.Ambit_water) || (target.GetOperatingAmbit() == types.Ambit_land) {
							result.PlanetaryEvaded = target.IsSuccessful(planetaryRate)
							if result.PlanetaryEvaded {
								result.Evaded = true
								result.PlanetaryCause = types.TechPlanetaryDefenses_lowOrbitBallisticInterceptorNetwork
							}
						}
					}
				}
			}
		}
	}

	ac.Evasion = result
	return result.Evaded
}

// resolveCounterDamage applies counter-attack damage from counterStruct to the attacker.
func (ac *AttackContext) resolveCounterDamage(counterStruct *StructCache, isTargetCounter bool) CounterResult {
	attacker := ac.Attacker

	result := CounterResult{
		IsTargetCounter: isTargetCounter,
		CounterStruct:   counterStruct,
	}

	if attacker.IsDestroyed() {
		return result
	}

	damage := counterStruct.GetStructType().GetCounterAttackDamage(attacker.GetOperatingAmbit() == counterStruct.GetOperatingAmbit())
	attacker.CC.k.logger.Debug("Struct Counter-Attack", "damage", damage, "counterAttacker", counterStruct.GetStructId(), "target", attacker.GetStructId())

	result.Damage = damage

	if damage != 0 {
		counterStruct.SetCounterSpent()
		attacker.CC.SetStructAttributeDecrement(attacker.HealthAttributeId, damage)

		if attacker.GetHealth() == 0 {
			attacker.CC.k.logger.Debug("Struct Destroyed During Counter-Attack", "counterAttacker", counterStruct.GetStructId(), "target", attacker.GetStructId())
			if attacker.GetStructType().PostDestructionDamage > 0 {
				pdDamage, pdDestroyed, pdPassive := counterStruct.applyPostDestructionDamageCore(attacker)
				result.PostDestruction = &PostDestructionResult{
					Damage:            pdDamage,
					AttackerDestroyed: pdDestroyed,
					PassiveWeaponry:   pdPassive,
				}
			}
			attacker.DestroyAndCommit()
		}
	}

	result.AttackerDestroyed = attacker.IsDestroyed()

	counterWS := counterStruct.GetStructType().GetCounterWeaponSystem(
		counterStruct.GetOperatingAmbit(), ac.Attacker.GetOperatingAmbit())
	result.WeaponSystem = counterWS
	result.WeaponControl = counterStruct.GetStructType().GetWeaponControl(counterWS)
	result.ActiveWeaponry = counterStruct.GetStructType().GetWeapon(counterWS)
	result.PassiveWeaponry = counterStruct.GetStructType().PassiveWeaponry

	return result
}

// resolveBlock attempts a block by the defender. Returns true if the block succeeded.
func (ac *AttackContext) resolveBlock(defender *StructCache) bool {
	if !defender.Ready || !ac.Attacker.Ready {
		return false
	}
	if defender.GetOperatingAmbit() != ac.Target.GetOperatingAmbit() {
		return false
	}

	ac.Block = &BlockResult{
		Blocked:      true,
		Blocker:      defender,
		HealthBefore: defender.GetHealth(),
		HealthMax:    defender.GetStructType().MaxHealth,
	}

	ac.resolveVolleyDamageOn(defender, true)
	return true
}

// resolveVolleyDamage applies volley damage to the target (or blocker if blocked).
func (ac *AttackContext) resolveVolleyDamage() {
	if ac.Block != nil && ac.Block.Blocked {
		return
	}
	ac.resolveVolleyDamageOn(ac.Target, false)
}

// resolveVolleyDamageOn rolls shots and applies damage to the given struct.
func (ac *AttackContext) resolveVolleyDamageOn(target *StructCache, isBlocker bool) {
	attacker := ac.Attacker

	if target.IsDestroyed() {
		ac.Volley = &VolleyResult{IsBlocker: isBlocker, HealthAfter: 0}
		return
	}

	n := int(attacker.GetStructType().GetWeaponShots(ac.WeaponSystem))
	if n == 0 {
		ac.Volley = &VolleyResult{IsBlocker: isBlocker, HealthAfter: target.GetHealth()}
		return
	}

	shots := make([]ShotOutcome, n)
	var rolled uint64
	for i := 0; i < n; i++ {
		if attacker.IsSuccessful(attacker.GetStructType().GetWeaponShotSuccessRate(ac.WeaponSystem)) {
			d := attacker.GetStructType().GetWeaponDamage(ac.WeaponSystem)
			shots[i] = ShotOutcome{Hit: true, Damage: d}
			rolled += d
		}
	}

	net := rolled
	damageReduction := target.GetStructType().AttackReduction
	unitDefenses := target.GetStructType().UnitDefenses
	if rolled != 0 && damageReduction > 0 {
		if damageReduction >= rolled {
			net = 1
		} else {
			net = rolled - damageReduction
		}
	}

	vr := &VolleyResult{
		Shots:           shots,
		RolledDamage:    rolled,
		DamageReduction: damageReduction,
		UnitDefenses:    unitDefenses,
		NetDamage:       net,
		IsBlocker:       isBlocker,
	}

	if net != 0 {
		target.CC.SetStructAttributeDecrement(target.HealthAttributeId, net)

		if target.GetHealth() == 0 {
			vr.TargetOrBlockerDestroyed = true
			if target.GetStructType().PostDestructionDamage > 0 {
				pdDamage, pdDestroyed, pdPassive := attacker.applyPostDestructionDamageCore(target)
				vr.PostDestruction = &PostDestructionResult{
					Damage:            pdDamage,
					AttackerDestroyed: pdDestroyed,
					PassiveWeaponry:   pdPassive,
				}
			}
			target.DestroyAndCommit()
		}
	}

	vr.HealthAfter = target.GetHealth()
	ac.Volley = vr
}

// resolveRecoil applies recoil damage to the attacker.
func (ac *AttackContext) resolveRecoil() {
	attacker := ac.Attacker
	if attacker.IsDestroyed() {
		ac.Recoil = &RecoilResult{}
		return
	}

	damage := attacker.GetStructType().GetWeaponRecoilDamage(ac.WeaponSystem)

	if damage != 0 {
		attacker.CC.SetStructAttributeDecrement(attacker.HealthAttributeId, damage)
		if attacker.GetHealth() == 0 {
			attacker.DestroyAndCommit()
		}
	}

	ac.Recoil = &RecoilResult{
		Damage:    damage,
		Destroyed: attacker.IsDestroyed(),
	}
}

// resolvePlanetaryDefense applies planetary defense cannon damage to the attacker.
func (ac *AttackContext) resolvePlanetaryDefense() {
	weaponCounterable := ac.Attacker.GetStructType().AttackCounterable && ac.Attacker.GetStructType().GetWeaponCounterable(ac.WeaponSystem)
	if ac.Attacker.IsDestroyed() || !weaponCounterable {
		return
	}

	var totalDamage uint64
	for _, planet := range ac.TargetedPlanets {
		if ac.Attacker.IsDestroyed() {
			break
		}
		cannonDamage := planet.GetDefensiveCannonQuantity()
		if cannonDamage > 0 {
			if !ac.Attacker.IsDestroyed() {
				ac.Attacker.CC.SetStructAttributeDecrement(ac.Attacker.HealthAttributeId, cannonDamage)
				if ac.Attacker.GetHealth() == 0 {
					ac.Attacker.DestroyAndCommit()
				}
			}
			totalDamage += cannonDamage
		}
	}

	if totalDamage > 0 {
		ac.PlanetaryDefense = &PlanetaryDefenseResult{
			Damage:    totalDamage,
			Destroyed: ac.Attacker.IsDestroyed(),
		}
	}
}

func (ac *AttackContext) ResolveEvasion() bool {
	evaded := ac.resolveEvasion()
	if evaded {
		ac.Attacker.CC.k.logger.Debug("Struct Evaded", "target", ac.Target.GetStructId())
	}
	return evaded
}

func (ac *AttackContext) ResolveDefenders(skipBlock bool) {
	weaponBlockable := ac.Attacker.GetStructType().GetWeaponBlockable(ac.WeaponSystem)
	weaponCounterable := ac.Attacker.GetStructType().AttackCounterable && ac.Attacker.GetStructType().GetWeaponCounterable(ac.WeaponSystem)

	if !weaponBlockable && !weaponCounterable {
		return
	}

	ac.Attacker.CC.k.logger.Debug("Struct Attacker Status",
		"structId", ac.Attacker.GetStructId(),
		"blockable", weaponBlockable,
		"counterable", weaponCounterable,
	)

	blocked := false
	defenders := ac.Target.GetDefenders()
	for _, defender := range defenders {
		ac.Attacker.CC.k.logger.Debug("Defender at Location", "defender", defender.GetStructId(), "locationId", defender.GetLocationId())

		defender = ac.Attacker.CC.GetStruct(defender.GetStructId())

		defenderReadinessError := defender.ReadinessCheck()
		if defenderReadinessError == nil {
			ac.Attacker.CC.k.logger.Debug("Defender seems ready to defend")

			if weaponCounterable {
				ac.Attacker.CC.k.logger.Debug("Defender trying to counter")
				counterErrors := defender.CanCounterAttack(ac.Attacker)
				if counterErrors == nil {
					ac.Attacker.CC.k.logger.Debug("Defender counter-attacking")
					cr := ac.resolveCounterDamage(defender, false)
					ac.DefenderCounters = append(ac.DefenderCounters, cr)
				}
			}

			if !blocked && !skipBlock && weaponBlockable {
				ac.Attacker.CC.k.logger.Debug("Defender to attempt a block")
				blocked = ac.resolveBlock(defender)
			}
		}
	}
}

func (ac *AttackContext) ResolveAttackDamage() {
	blocked := ac.Block != nil && ac.Block.Blocked
	if !blocked && ac.Attacker.IsOnline() {
		ac.Attacker.CC.k.logger.Debug("Moving forward with the attack", "target", ac.Target.GetStructId())
		ac.resolveVolleyDamage()
	} else {
		ac.Attacker.CC.k.logger.Debug("Attack against target was blocked", "target", ac.Target.GetStructId())
	}
}

func (ac *AttackContext) ResolveTargetCounter() {
	if ac.Attacker.GetStructType().AttackCounterable && ac.Attacker.GetStructType().GetWeaponCounterable(ac.WeaponSystem) {
		ac.Attacker.CC.k.logger.Debug("Target trying to Counter now")
		counterErrors := ac.Target.CanCounterAttack(ac.Attacker)
		if counterErrors == nil {
			ac.Attacker.CC.k.logger.Debug("Target Countering")
			cr := ac.resolveCounterDamage(ac.Target, true)
			ac.TargetCounter = &cr
		}
	}
}

func (ac *AttackContext) ResolveRecoil() {
	if !ac.Attacker.IsDestroyed() {
		ac.resolveRecoil()
	}
}

func (ac *AttackContext) ResolvePlanetaryDefense() {
	ac.resolvePlanetaryDefense()
}

// newProjectileRow creates a copy of the base shot detail for intermediate projectile rows.
// Outcome fields are zeroed; the defender counter slice is nil-ed to prevent shared-array aliasing.
func newProjectileRow(base *types.EventAttackShotDetail) *types.EventAttackShotDetail {
	row := *base
	row.EventAttackDefenderCounterDetail = nil
	row.DamageDealt = 0
	row.DamageReduction = 0
	row.DamageReductionCause = 0
	row.Damage = 0
	row.BlockerDestroyed = false
	row.TargetDestroyed = false
	row.TargetCountered = false
	row.TargetCounteredDamage = 0
	row.TargetCounterDestroyedAttacker = false
	row.TargetCounterPassiveWeaponry = 0
	row.TargetCounterWeaponSystem = 0
	row.TargetCounterWeaponControl = 0
	row.TargetCounterActiveWeaponry = 0
	row.PostDestructionDamageToAttacker = false
	row.PostDestructionDamage = 0
	row.PostDestructionDamageDestroyedAttacker = false
	row.PostDestructionDamagePassiveWeaponry = 0
	row.TargetHealthAfter = 0
	row.BlockerHealthAfter = 0
	return &row
}

// EndShot is the sole per-target event constructor. It reads accumulated results
// and builds all EventAttackShotDetail rows for this target volley.
func (ac *AttackContext) EndShot() {
	base := types.CreateEventAttackShotDetail(ac.Target.GetStructId())
	base.SetTargetDetails(
		ac.Target.GetStructId(),
		ac.Target.GetTypeId(),
		ac.Target.GetStructType().Type,
		ac.Target.GetLocationType(),
		ac.Target.GetLocationId(),
		ac.Target.GetOperatingAmbit(),
		ac.Target.GetSlot(),
	)
	base.TargetPlayerId = ac.Target.GetOwnerId()

	targetHealth := ac.Target.GetHealth()
	if ac.Volley == nil || ac.Volley.IsBlocker {
		base.TargetHealthBefore = targetHealth
		base.TargetHealthAfter = targetHealth
	} else {
		base.TargetHealthBefore = ac.Volley.HealthAfter + ac.Volley.NetDamage
		base.TargetHealthAfter = ac.Volley.HealthAfter
	}
	base.TargetHealthMax = ac.Target.GetStructType().MaxHealth

	// Evasion
	if ac.Evasion != nil {
		if ac.Evasion.Evaded {
			if ac.Evasion.PlanetaryEvaded {
				base.SetEvadeByPlanetaryDefenses(true, ac.Evasion.PlanetaryCause)
			} else {
				base.SetEvade(true, ac.Evasion.Cause)
			}
			ac.AttackDetail.AppendShot(base)
			ac.trackTargetedPlanet()
			return
		}
	}

	// Block
	if ac.Block != nil && ac.Block.Blocked {
		blocker := ac.Block.Blocker
		base.SetBlocker(
			blocker.GetStructId(),
			blocker.GetTypeId(),
			blocker.GetStructType().Type,
			blocker.GetLocationType(),
			blocker.GetLocationId(),
			blocker.GetOperatingAmbit(),
			blocker.GetSlot(),
		)
		base.BlockerHealthBefore = ac.Block.HealthBefore
		base.BlockerHealthMax = ac.Block.HealthMax
	}

	// Defender counters
	for _, dc := range ac.DefenderCounters {
		cs := dc.CounterStruct
		base.AppendDefenderCounter(
			cs.GetStructId(), dc.Damage, dc.AttackerDestroyed,
			cs.GetTypeId(), cs.GetStructType().Type,
			cs.GetLocationType(), cs.GetLocationId(),
			cs.GetOperatingAmbit(), cs.GetSlot(),
			dc.WeaponSystem, dc.WeaponControl, dc.ActiveWeaponry,
		)
	}

	// Target counter
	if ac.TargetCounter != nil && ac.TargetCounter.Damage > 0 {
		base.AppendTargetCounter(
			ac.TargetCounter.Damage, ac.TargetCounter.AttackerDestroyed,
			ac.TargetCounter.PassiveWeaponry,
			ac.TargetCounter.WeaponSystem, ac.TargetCounter.WeaponControl, ac.TargetCounter.ActiveWeaponry,
		)
	}

	// No volley (attacker destroyed before damage, or blocked damage handled below)
	if ac.Volley == nil {
		ac.AttackDetail.AppendShot(base)
		ac.trackTargetedPlanet()
		return
	}

	vr := ac.Volley
	n := len(vr.Shots)

	if n <= 1 {
		// Single shot (or zero-shot weapon): apply everything to the base row
		if n == 1 {
			base.DamageDealt = vr.Shots[0].Damage
		}
		if vr.RolledDamage != 0 && vr.DamageReduction > 0 {
			base.DamageReduction = vr.DamageReduction
			base.DamageReductionCause = vr.UnitDefenses
		}
		base.Damage = vr.NetDamage

		if vr.IsBlocker {
			base.BlockerHealthAfter = vr.HealthAfter
		} else {
			base.TargetHealthAfter = vr.HealthAfter
		}

		if vr.TargetOrBlockerDestroyed {
			if vr.IsBlocker {
				base.BlockerDestroyed = true
			} else {
				base.TargetDestroyed = true
			}
		}

		if vr.PostDestruction != nil {
			base.SetPostDestructionDamage(vr.PostDestruction.Damage, vr.PostDestruction.AttackerDestroyed, vr.PostDestruction.PassiveWeaponry)
		}

		ac.AttackDetail.AppendShot(base)
	} else {
		// Multi-shot: N-1 intermediate rows + 1 last row
		for i := 0; i < n-1; i++ {
			row := newProjectileRow(base)
			row.DamageDealt = vr.Shots[i].Damage
			if vr.IsBlocker {
				row.BlockerHealthAfter = base.BlockerHealthBefore
			} else {
				row.TargetHealthAfter = base.TargetHealthBefore
			}
			ac.AttackDetail.AppendShot(row)
		}

		// Last row carries volley aggregate
		lastRow := newProjectileRow(base)
		lastRow.DamageDealt = vr.Shots[n-1].Damage
		if vr.RolledDamage != 0 && vr.DamageReduction > 0 {
			lastRow.DamageReduction = vr.DamageReduction
			lastRow.DamageReductionCause = vr.UnitDefenses
		}
		lastRow.Damage = vr.NetDamage

		if vr.IsBlocker {
			lastRow.BlockerHealthAfter = vr.HealthAfter
			lastRow.TargetHealthAfter = base.TargetHealthBefore
		} else {
			lastRow.TargetHealthAfter = vr.HealthAfter
		}

		if vr.TargetOrBlockerDestroyed {
			if vr.IsBlocker {
				lastRow.BlockerDestroyed = true
			} else {
				lastRow.TargetDestroyed = true
			}
		}

		if vr.PostDestruction != nil {
			lastRow.SetPostDestructionDamage(vr.PostDestruction.Damage, vr.PostDestruction.AttackerDestroyed, vr.PostDestruction.PassiveWeaponry)
		}

		// Restore defender counters and target counter on last row
		lastRow.EventAttackDefenderCounterDetail = base.EventAttackDefenderCounterDetail
		lastRow.TargetCountered = base.TargetCountered
		lastRow.TargetCounteredDamage = base.TargetCounteredDamage
		lastRow.TargetCounterDestroyedAttacker = base.TargetCounterDestroyedAttacker
		lastRow.TargetCounterPassiveWeaponry = base.TargetCounterPassiveWeaponry
		lastRow.TargetCounterWeaponSystem = base.TargetCounterWeaponSystem
		lastRow.TargetCounterWeaponControl = base.TargetCounterWeaponControl
		lastRow.TargetCounterActiveWeaponry = base.TargetCounterActiveWeaponry

		ac.AttackDetail.AppendShot(lastRow)
	}

	ac.trackTargetedPlanet()
}

func (ac *AttackContext) trackTargetedPlanet() {
	if ac.Target.GetStructType().Category == types.ObjectType_planet {
		planet := ac.Target.GetPlanet()
		planetId := planet.GetPlanetId()
		if !ac.targetedPlanetsSeen[planetId] {
			ac.targetedPlanetsSeen[planetId] = true
			ac.TargetedPlanets = append(ac.TargetedPlanets, planet)
		}
	}
}

func (ac *AttackContext) Finalize(ctx sdk.Context) {
	ac.AttackDetail.AttackerHealthAfter = ac.Attacker.GetHealth()

	if ac.Recoil != nil && ac.Recoil.Damage > 0 {
		ac.AttackDetail.SetRecoilDamage(ac.Recoil.Damage, ac.Recoil.Destroyed)
	}

	if ac.PlanetaryDefense != nil && ac.PlanetaryDefense.Damage > 0 {
		ac.AttackDetail.SetPlanetaryDefenseCannonDamage(ac.PlanetaryDefense.Damage, ac.PlanetaryDefense.Destroyed)
	}

	_ = ctx.EventManager().EmitTypedEvent(&types.EventAttack{EventAttackDetail: ac.AttackDetail})
	ac.Attacker.CC.Attack = nil
}
