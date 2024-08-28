package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
)


func (eventAttackDetail *EventAttackDetail) SetSomething() {
}

func CreateEventAttackDetail() (*EventAttackDetail) {
    return &EventAttackDetail{}
}

func (eventAttackDetail *EventAttackDetail) SetBaseDetails(attackerStructId string, weaponSystem TechWeaponSystem, weaponControl TechWeaponControl, activeWeaponry TechActiveWeaponry) {
  eventAttackDetail.AttackerStructId = attackerStructId

  eventAttackDetail.WeaponSystem    = weaponSystem
  eventAttackDetail.WeaponControl   = weaponControl
  eventAttackDetail.ActiveWeaponry  = activeWeaponry
}

func (eventAttackDetail *EventAttackDetail) SetRecoilDamage(recoilDamage uint64, recoilDamageDestroyedAttacker bool) {
    eventAttackDetail.RecoilDamageToAttacker = true
    eventAttackDetail.RecoilDamage = recoilDamage
    eventAttackDetail.RecoilDamageDestroyedAttacker = recoilDamageDestroyedAttacker
}

func (eventAttackDetail *EventAttackDetail) AppendShot(eventAttackShotDetail *EventAttackShotDetail) {
    eventAttackDetail.EventAttackShotDetail = append(eventAttackDetail.EventAttackShotDetail, eventAttackShotDetail)
}


func CreateEventAttackShotDetail(targetStructId string) (*EventAttackShotDetail) {
    return &EventAttackShotDetail{
        TargetStructId: targetStructId,
    }
}

func (eventAttackShotDetail *EventAttackShotDetail) SetBlocker(blockedByStructId string) {
    eventAttackShotDetail.BlockedByStructId = blockedByStructId
    eventAttackShotDetail.Blocked = true
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




func (eventAttackShotDetail *EventAttackShotDetail) AppendDefenderCounter(counterByStructId string, counterDamage uint64, counterDestroyedAttacker bool) {
    eventAttackDefenderCounterDetail := EventAttackDefenderCounterDetail{
        CounterByStructId: counterByStructId,
        CounterDamage: counterDamage,
        CounterDestroyedAttacker: counterDestroyedAttacker,
    }
    eventAttackShotDetail.EventAttackDefenderCounterDetail = append(eventAttackShotDetail.EventAttackDefenderCounterDetail, &eventAttackDefenderCounterDetail)

}

func (eventAttackShotDetail *EventAttackShotDetail) AppendTargetCounter(counterDamage uint64, counterDestroyedAttacker bool, passiveWeaponry TechPassiveWeaponry) {
    eventAttackShotDetail.TargetCountered = true
    eventAttackShotDetail.TargetCounteredDamage = counterDamage
    eventAttackShotDetail.TargetCounterDestroyedAttacker = counterDestroyedAttacker
    eventAttackShotDetail.TargetCounterCause = passiveWeaponry
}


/*
message EventAttackShotDetail {
  string  targetStructId = 1;

  bool evaded                  = 2;
  techUnitDefenses evadedCause = 3;

  bool blocked              = 4;
  string blockedByStructId  = 5;

  repeated EventAttackDefenderCounterDetail eventAttackDefenderCounterDetail = 6;

  uint64 damageDealt                    = 7;
  uint64 damageReduction                = 8;
  techUnitDefenses damageReductionCause = 9;
  uint64 damage                         = 10;


  bool targetCountered                    = 11;
  uint64 targetCounteredDamage            = 12;
  bool targetCounterDestroyedAttacker     = 13;
  techPassiveWeaponry targetCounterCause  = 14;

  bool targetDestroyed = 15;

  bool postDestructionDamageToAttacker            = 16;
  uint64 postDestructionDamage                    = 17;
  bool postDestructionDamageDestroyedAttacker     = 18;
  techPassiveWeaponry postDestructionDamageCause  = 19;

}

message EventAttackDefenderCounterDetail {
  string  counterByStructId         = 1;
  uint64  counterDamage             = 2;
  bool    counterDestroyedAttacker  = 3;
}

*/