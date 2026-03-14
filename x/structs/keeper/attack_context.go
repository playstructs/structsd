package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

type AttackContext struct {
	Attacker     *StructCache
	WeaponSystem types.TechWeaponSystem

	AttackDetail *types.EventAttackDetail

	Target     *StructCache
	ShotDetail *types.EventAttackShotDetail
	Blocked    bool
	Blocker    *StructCache

	TargetWasPlanetary bool
	TargetPlanet       *PlanetCache
}

func NewAttackContext(cc *CurrentContext, attacker *StructCache, weaponSystem types.TechWeaponSystem) *AttackContext {
	ac := &AttackContext{
		Attacker:     attacker,
		WeaponSystem: weaponSystem,
		AttackDetail: types.CreateEventAttackDetail(),
	}

	ac.AttackDetail.SetBaseDetails(
		attacker.GetOwnerId(),
		attacker.GetStructId(),
		attacker.GetTypeId(),
		attacker.GetLocationType(),
		attacker.GetLocationId(),
		attacker.GetOperatingAmbit(),
		attacker.GetSlot(),
		weaponSystem,
		attacker.GetStructType().GetWeaponControl(weaponSystem),
		attacker.GetStructType().GetWeapon(weaponSystem),
	)

	cc.Attack = ac
	return ac
}

func (ac *AttackContext) BeginShot(target *StructCache) {
	ac.Target = target
	ac.Blocked = false
	ac.Blocker = nil

	ac.ShotDetail = types.CreateEventAttackShotDetail(target.GetStructId())
	ac.ShotDetail.SetTargetDetails(
		target.GetStructId(),
		target.GetTypeId(),
		target.GetLocationType(),
		target.GetLocationId(),
		target.GetOperatingAmbit(),
		target.GetSlot(),
	)

	ac.AttackDetail.SetTargetPlayerId(target.GetOwnerId())

	currentTargetHealth := target.GetHealth()
	ac.ShotDetail.SetTargetHealthBefore(currentTargetHealth)
	ac.ShotDetail.SetTargetHealthAfter(currentTargetHealth)
}

func (ac *AttackContext) ValidateTarget() error {
	return ac.Attacker.CanAttack(ac.Target, ac.WeaponSystem)
}

func (ac *AttackContext) ResolveEvasion() bool {
	if ac.Target.CanEvade(ac.Attacker, ac.WeaponSystem) {
		ac.Attacker.CC.k.logger.Info("Struct Evaded", "target", ac.Target.GetStructId())
		return true
	}
	return false
}

func (ac *AttackContext) ResolveDefenders(skipBlock bool) {
	weaponBlockable := ac.Attacker.GetStructType().GetWeaponBlockable(ac.WeaponSystem)
	weaponCounterable := ac.Attacker.GetStructType().AttackCounterable && ac.Attacker.GetStructType().GetWeaponCounterable(ac.WeaponSystem)

	if !weaponBlockable && !weaponCounterable {
		return
	}

	ac.Attacker.CC.k.logger.Info("Struct Attacker Status",
		"structId", ac.Attacker.GetStructId(),
		"blockable", weaponBlockable,
		"counterable", weaponCounterable,
	)

	defenders := ac.Target.GetDefenders()
	for _, defender := range defenders {
		ac.Attacker.CC.k.logger.Info("Defender at Location", "defender", defender.GetStructId(), "locationId", defender.GetLocationId())

		defender = ac.Attacker.CC.GetStruct(defender.GetStructId())

		defenderReadinessError := defender.ReadinessCheck()
		if defenderReadinessError == nil {
			ac.Attacker.CC.k.logger.Info("Defender seems ready to defend")

			if weaponCounterable {
				ac.Attacker.CC.k.logger.Info("Defender trying to counter!.. ")
				counterErrors := defender.CanCounterAttack(ac.Attacker)
				if counterErrors == nil {
					ac.Attacker.CC.k.logger.Info("Defender counter-attacking!")
					ac.Attacker.TakeCounterAttackDamage(defender)
				}
			}

			if !ac.Blocked && !skipBlock && weaponBlockable {
				ac.Attacker.CC.k.logger.Info("Defender to attempt a block!")
				ac.Blocked = defender.AttemptBlock(ac.Attacker, ac.WeaponSystem, ac.Target)
			}
		}
	}
}

func (ac *AttackContext) ResolveAttackDamage() {
	if !ac.Blocked && ac.Attacker.IsOnline() {
		ac.Attacker.CC.k.logger.Info("Moving forward with the attack", "target", ac.Target.GetStructId())
		ac.Target.TakeAttackDamage(ac.Attacker, ac.WeaponSystem)
	} else {
		ac.Attacker.CC.k.logger.Info("Attack against target was blocked", "target", ac.Target.GetStructId())
	}
}

func (ac *AttackContext) ResolveTargetCounter() {
	if ac.Attacker.GetStructType().AttackCounterable && ac.Attacker.GetStructType().GetWeaponCounterable(ac.WeaponSystem) {
		ac.Attacker.CC.k.logger.Info("Target trying to Counter now!")
		counterErrors := ac.Target.CanCounterAttack(ac.Attacker)
		if counterErrors == nil {
			ac.Attacker.CC.k.logger.Info("Target Countering!")
			ac.Attacker.TakeCounterAttackDamage(ac.Target)
		}
	}
}

func (ac *AttackContext) EndShot() {
	ac.AttackDetail.AppendShot(ac.ShotDetail)

	if ac.Target.GetStructType().Category == types.ObjectType_planet {
		ac.TargetWasPlanetary = true
		ac.TargetPlanet = ac.Target.GetPlanet()
	}
}

func (ac *AttackContext) ResolveRecoil() {
	if !ac.Attacker.IsDestroyed() {
		ac.Attacker.TakeRecoilDamage(ac.WeaponSystem)
	}
}

func (ac *AttackContext) ResolvePlanetaryDefense() {
	weaponCounterable := ac.Attacker.GetStructType().AttackCounterable && ac.Attacker.GetStructType().GetWeaponCounterable(ac.WeaponSystem)
	if !ac.Attacker.IsDestroyed() && ac.TargetWasPlanetary && weaponCounterable {
		ac.TargetPlanet.AttemptDefenseCannon(ac.Attacker)
	}
}

func (ac *AttackContext) Finalize(ctx sdk.Context) {
	ac.AttackDetail.SetAttackerHealthAfter(ac.Attacker.GetHealth())
	_ = ctx.EventManager().EmitTypedEvent(&types.EventAttack{EventAttackDetail: ac.AttackDetail})
	ac.Attacker.CC.Attack = nil
}
