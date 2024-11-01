package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"fmt"
)


func CreateEventAttackDetail() (*EventAttackDetail) {
    return &EventAttackDetail{ }
}



func (eventAttackDetail *EventAttackDetail) SetBaseDetails(attackerPlayerId string, attackerStructId string, attackerStructType uint64, attackerStructLocationType ObjectType, attackerStructLocationId string, attackerStructOperatingAmbit Ambit,  attackerStructSlot uint64, weaponSystem TechWeaponSystem, weaponControl TechWeaponControl, activeWeaponry TechActiveWeaponry) {

  eventAttackDetail.AttackerPlayerId                = attackerPlayerId
  eventAttackDetail.AttackerStructId                = attackerStructId
  eventAttackDetail.AttackerStructType              = attackerStructType
  eventAttackDetail.AttackerStructLocationType      = attackerStructLocationType
  eventAttackDetail.AttackerStructLocationId        = attackerStructLocationId
  eventAttackDetail.AttackerStructOperatingAmbit    = attackerStructOperatingAmbit
  eventAttackDetail.AttackerStructSlot              = attackerStructSlot

  eventAttackDetail.WeaponSystem    = weaponSystem
  eventAttackDetail.WeaponControl   = weaponControl
  eventAttackDetail.ActiveWeaponry  = activeWeaponry
}

func (eventAttackDetail *EventAttackDetail) SetTargetPlayerId(targetPlayerId string) {
  eventAttackDetail.TargetPlayerId = targetPlayerId
}

func (eventAttackDetail *EventAttackDetail) SetRecoilDamage(recoilDamage uint64, recoilDamageDestroyedAttacker bool) {
    eventAttackDetail.RecoilDamageToAttacker = true
    eventAttackDetail.RecoilDamage = recoilDamage
    eventAttackDetail.RecoilDamageDestroyedAttacker = recoilDamageDestroyedAttacker
}

func (eventAttackDetail *EventAttackDetail) SetPlanetaryDefenseCannonDamage(planetaryDefenseCannonDamage uint64, planetaryDefenseCannonDamageDestroyedAttacker bool) {
    eventAttackDetail.PlanetaryDefenseCannonDamageToAttacker = true
    eventAttackDetail.PlanetaryDefenseCannonDamage = planetaryDefenseCannonDamage
    eventAttackDetail.PlanetaryDefenseCannonDamageDestroyedAttacker = planetaryDefenseCannonDamageDestroyedAttacker
}


func (eventAttackDetail *EventAttackDetail) AppendShot(eventAttackShotDetail *EventAttackShotDetail) {
    eventAttackDetail.EventAttackShotDetail = append(eventAttackDetail.EventAttackShotDetail, eventAttackShotDetail)
}


/* Sub Event - Attack Shots */
func CreateEventAttackShotDetail(targetStructId string) (*EventAttackShotDetail) {
    return &EventAttackShotDetail{
        TargetStructId: targetStructId,
        EventAttackDefenderCounterDetail: []*EventAttackDefenderCounterDetail{},
    }
}

func (eventAttackShotDetail *EventAttackShotDetail)  SetTargetDetails(targetStructId string, targetStructType uint64, targetStructLocationType ObjectType, targetStructLocationId string, targetStructOperatingAmbit Ambit, targetStructSlot uint64) {
  eventAttackShotDetail.TargetStructId                = targetStructId
  eventAttackShotDetail.TargetStructType              = targetStructType
  eventAttackShotDetail.TargetStructLocationType      = targetStructLocationType
  eventAttackShotDetail.TargetStructLocationId        = targetStructLocationId
  eventAttackShotDetail.TargetStructOperatingAmbit    = targetStructOperatingAmbit
  eventAttackShotDetail.TargetStructSlot              = targetStructSlot
}

func (eventAttackShotDetail *EventAttackShotDetail) SetEvade(evaded bool, evadedCause TechUnitDefenses) {
    eventAttackShotDetail.Evaded = evaded
    eventAttackShotDetail.EvadedCause = evadedCause
}


func (eventAttackShotDetail *EventAttackShotDetail) SetEvadeByPlanetaryDefenses(evaded bool, evadedCause TechPlanetaryDefenses) {
    eventAttackShotDetail.EvadedByPlanetaryDefenses = evaded
    eventAttackShotDetail.EvadedByPlanetaryDefensesCause = evadedCause
}

func (eventAttackShotDetail *EventAttackShotDetail) SetBlocker(blockedByStructId string, blockedByStructType uint64, blockedByStructLocationType ObjectType, blockedByStructLocationId string, blockedByStructOperatingAmbit Ambit, blockedByStructSlot uint64) {
    eventAttackShotDetail.BlockedByStructId                 = blockedByStructId
    eventAttackShotDetail.BlockedByStructType               = blockedByStructType
    eventAttackShotDetail.BlockedByStructLocationType       = blockedByStructLocationType
    eventAttackShotDetail.BlockedByStructLocationId         = blockedByStructLocationId
    eventAttackShotDetail.BlockedByStructOperatingAmbit     = blockedByStructOperatingAmbit
    eventAttackShotDetail.BlockedByStructSlot               = blockedByStructSlot
    eventAttackShotDetail.Blocked                           = true
}

func (eventAttackShotDetail *EventAttackShotDetail) SetBlockerDestroyed() {
    eventAttackShotDetail.BlockerDestroyed = true
}

func (eventAttackShotDetail *EventAttackShotDetail) SetTargetDestroyed() {
    eventAttackShotDetail.TargetDestroyed = true
}

func (eventAttackShotDetail *EventAttackShotDetail) SetDamageReduction(damageReduction uint64, damageReductionCause TechUnitDefenses) {
    eventAttackShotDetail.DamageReduction = damageReduction
    eventAttackShotDetail.DamageReductionCause = damageReductionCause
}

func (eventAttackShotDetail *EventAttackShotDetail) SetDamageDealt(damageDealt uint64) {
    eventAttackShotDetail.DamageDealt = damageDealt
}

func (eventAttackShotDetail *EventAttackShotDetail) SetDamage(damage uint64) {
    eventAttackShotDetail.Damage = damage
}

func (eventAttackShotDetail *EventAttackShotDetail) SetPostDestructionDamage(postDestructionDamage uint64, postDestructionDamageDestroyedAttacker bool, postDestructionDamageCause TechPassiveWeaponry) {
    eventAttackShotDetail.PostDestructionDamageToAttacker         = true
    eventAttackShotDetail.PostDestructionDamage                   = postDestructionDamage
    eventAttackShotDetail.PostDestructionDamageDestroyedAttacker  = postDestructionDamageDestroyedAttacker
    eventAttackShotDetail.PostDestructionDamageCause              = postDestructionDamageCause
}

func (eventAttackShotDetail *EventAttackShotDetail) AppendDefenderCounter(counterByStructId string, counterDamage uint64, counterDestroyedAttacker bool, counterByStructType uint64, counterByStructLocationType ObjectType, counterByStructLocationId string, counterByStructOperatingAmbit Ambit, counterByStructSlot uint64) {
    fmt.Printf("Recording Defensive Counter Attack %s %d %t \n", counterByStructId, counterDamage, counterDestroyedAttacker)
    eventAttackDefenderCounterDetail := EventAttackDefenderCounterDetail{
        CounterByStructId: counterByStructId,
        CounterByStructType: counterByStructType,
        CounterByStructLocationType: counterByStructLocationType,
        CounterByStructLocationId: counterByStructLocationId,
        CounterByStructOperatingAmbit: counterByStructOperatingAmbit,
        CounterByStructSlot: counterByStructSlot,
        CounterDamage: counterDamage,
        CounterDestroyedAttacker: counterDestroyedAttacker,
    }
    eventAttackShotDetail.EventAttackDefenderCounterDetail = append(eventAttackShotDetail.EventAttackDefenderCounterDetail, &eventAttackDefenderCounterDetail)

}

func (eventAttackShotDetail *EventAttackShotDetail) AppendTargetCounter(counterDamage uint64, counterDestroyedAttacker bool, passiveWeaponry TechPassiveWeaponry) {
    fmt.Printf("Recording Primary Counter Attack %d %t \n",  counterDamage, counterDestroyedAttacker)
    eventAttackShotDetail.TargetCountered = true
    eventAttackShotDetail.TargetCounteredDamage = counterDamage
    eventAttackShotDetail.TargetCounterDestroyedAttacker = counterDestroyedAttacker
    eventAttackShotDetail.TargetCounterCause = passiveWeaponry
}

