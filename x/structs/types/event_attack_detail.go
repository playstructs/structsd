package types

func CreateEventAttackDetail() *EventAttackDetail {
	return &EventAttackDetail{}
}

func (eventAttackDetail *EventAttackDetail) SetBaseDetails(attackerPlayerId string, attackerStructId string, attackerStructTypeId uint64, attackerStructType string, attackerStructLocationType ObjectType, attackerStructLocationId string, attackerStructOperatingAmbit Ambit, attackerStructSlot uint64, weaponSystem TechWeaponSystem, weaponControl TechWeaponControl, activeWeaponry TechActiveWeaponry) {
	eventAttackDetail.AttackerPlayerId = attackerPlayerId
	eventAttackDetail.AttackerStructId = attackerStructId
	eventAttackDetail.AttackerStructTypeId = attackerStructTypeId
	eventAttackDetail.AttackerStructType = attackerStructType
	eventAttackDetail.AttackerStructLocationType = attackerStructLocationType
	eventAttackDetail.AttackerStructLocationId = attackerStructLocationId
	eventAttackDetail.AttackerStructOperatingAmbit = attackerStructOperatingAmbit
	eventAttackDetail.AttackerStructSlot = attackerStructSlot

	eventAttackDetail.WeaponSystem = weaponSystem
	eventAttackDetail.WeaponControl = weaponControl
	eventAttackDetail.ActiveWeaponry = activeWeaponry
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
func CreateEventAttackShotDetail(targetStructId string) *EventAttackShotDetail {
	return &EventAttackShotDetail{
		TargetStructId:                   targetStructId,
		EventAttackDefenderCounterDetail: []*EventAttackDefenderCounterDetail{},
	}
}

func (eventAttackShotDetail *EventAttackShotDetail) SetTargetDetails(targetStructId string, targetStructTypeId uint64, targetStructType string, targetStructLocationType ObjectType, targetStructLocationId string, targetStructOperatingAmbit Ambit, targetStructSlot uint64) {
	eventAttackShotDetail.TargetStructId = targetStructId
	eventAttackShotDetail.TargetStructTypeId = targetStructTypeId
	eventAttackShotDetail.TargetStructType = targetStructType
	eventAttackShotDetail.TargetStructLocationType = targetStructLocationType
	eventAttackShotDetail.TargetStructLocationId = targetStructLocationId
	eventAttackShotDetail.TargetStructOperatingAmbit = targetStructOperatingAmbit
	eventAttackShotDetail.TargetStructSlot = targetStructSlot
}

func (eventAttackShotDetail *EventAttackShotDetail) SetEvade(evaded bool, evadedCause TechUnitDefenses) {
	eventAttackShotDetail.Evaded = evaded
	eventAttackShotDetail.EvadedCause = evadedCause
}

func (eventAttackShotDetail *EventAttackShotDetail) SetEvadeByPlanetaryDefenses(evaded bool, evadedCause TechPlanetaryDefenses) {
	eventAttackShotDetail.EvadedByPlanetaryDefenses = evaded
	eventAttackShotDetail.EvadedByPlanetaryDefensesCause = evadedCause
}

func (eventAttackShotDetail *EventAttackShotDetail) SetBlocker(blockedByStructId string, blockedByStructTypeId uint64, blockedByStructType string, blockedByStructLocationType ObjectType, blockedByStructLocationId string, blockedByStructOperatingAmbit Ambit, blockedByStructSlot uint64) {
	eventAttackShotDetail.BlockedByStructId = blockedByStructId
	eventAttackShotDetail.BlockedByStructTypeId = blockedByStructTypeId
	eventAttackShotDetail.BlockedByStructType = blockedByStructType
	eventAttackShotDetail.BlockedByStructLocationType = blockedByStructLocationType
	eventAttackShotDetail.BlockedByStructLocationId = blockedByStructLocationId
	eventAttackShotDetail.BlockedByStructOperatingAmbit = blockedByStructOperatingAmbit
	eventAttackShotDetail.BlockedByStructSlot = blockedByStructSlot
	eventAttackShotDetail.Blocked = true
}

func (eventAttackShotDetail *EventAttackShotDetail) SetPostDestructionDamage(postDestructionDamage uint64, postDestructionDamageDestroyedAttacker bool, postDestructionDamagePassiveWeaponry TechPassiveWeaponry) {
	eventAttackShotDetail.PostDestructionDamageToAttacker = true
	eventAttackShotDetail.PostDestructionDamage = postDestructionDamage
	eventAttackShotDetail.PostDestructionDamageDestroyedAttacker = postDestructionDamageDestroyedAttacker
	eventAttackShotDetail.PostDestructionDamagePassiveWeaponry = postDestructionDamagePassiveWeaponry
}

func (eventAttackShotDetail *EventAttackShotDetail) AppendDefenderCounter(counterByStructId string, counterDamage uint64, counterDestroyedAttacker bool, counterByStructTypeId uint64, counterByStructType string, counterByStructLocationType ObjectType, counterByStructLocationId string, counterByStructOperatingAmbit Ambit, counterByStructSlot uint64, counterByStructWeaponSystem TechWeaponSystem, counterByStructWeaponControl TechWeaponControl, counterByStructActiveWeaponry TechActiveWeaponry) {
	eventAttackDefenderCounterDetail := EventAttackDefenderCounterDetail{
		CounterByStructId:             counterByStructId,
		CounterByStructTypeId:         counterByStructTypeId,
		CounterByStructType:           counterByStructType,
		CounterByStructLocationType:   counterByStructLocationType,
		CounterByStructLocationId:     counterByStructLocationId,
		CounterByStructOperatingAmbit: counterByStructOperatingAmbit,
		CounterByStructSlot:           counterByStructSlot,
		CounterDamage:                 counterDamage,
		CounterDestroyedAttacker:      counterDestroyedAttacker,
		CounterByStructWeaponSystem:   counterByStructWeaponSystem,
		CounterByStructWeaponControl:  counterByStructWeaponControl,
		CounterByStructActiveWeaponry: counterByStructActiveWeaponry,
	}
	eventAttackShotDetail.EventAttackDefenderCounterDetail = append(eventAttackShotDetail.EventAttackDefenderCounterDetail, &eventAttackDefenderCounterDetail)
}

func (eventAttackShotDetail *EventAttackShotDetail) AppendTargetCounter(counterDamage uint64, counterDestroyedAttacker bool, passiveWeaponry TechPassiveWeaponry, activeWeaponSystem TechWeaponSystem, weaponControl TechWeaponControl, activeWeaponry TechActiveWeaponry) {
	eventAttackShotDetail.TargetCountered = true
	eventAttackShotDetail.TargetCounteredDamage = counterDamage
	eventAttackShotDetail.TargetCounterDestroyedAttacker = counterDestroyedAttacker
	eventAttackShotDetail.TargetCounterPassiveWeaponry = passiveWeaponry
	eventAttackShotDetail.TargetCounterWeaponSystem = activeWeaponSystem
	eventAttackShotDetail.TargetCounterWeaponControl = weaponControl
	eventAttackShotDetail.TargetCounterActiveWeaponry = activeWeaponry
}
