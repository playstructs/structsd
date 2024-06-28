package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
 *
 * This entire document is <3 garbage <3 and will
 * be re-written
 *
 */




func (structure *Struct) SetStatus(status string) error {
	structure.Status = status
	return nil
}


func (structure *Struct) SetMiningSystemStatus(status string) error {
	structure.MiningSystemStatus = status
	return nil
}


func (structure *Struct) SetRefiningSystemStatus(status string) error {
	structure.RefiningSystemStatus = status
	return nil
}


func (structure *Struct) SetSlot(slot uint64) error {
	structure.Slot = slot
	return nil
}

func (structure *Struct) SetPlanetId(planetId string) error {
	structure.PlanetId = planetId
	return nil
}


func (structure *Struct) SetBuildStartBlock(block uint64) error {

    structure.BuildStartBlock = block

    return nil
}

func (structure *Struct) SetMiningSystemActivationBlock(block uint64) error {
    structure.ActiveMiningSystemBlock = block

    return nil
}

func (structure *Struct) SetRefiningSystemActivationBlock(block uint64) error {
    structure.ActiveRefiningSystemBlock = block

    return nil
}

// Take an amount of fuel and return the energy it will generate
//
// This will need some work later on to be more dynamic in
// relation to other system state, but for now it is static.
func CalculateStructPower(fuel uint64) (energy uint64, ratio uint64) {
    return fuel * StructFuelToEnergyConversion, StructFuelToEnergyConversion
}

func CreateBaseStruct(structType string) Struct {

    var newCategory         string
    var newType             string
    var newAmbit            string

    var newMiningSystem     uint64
    var newRefiningSystem   uint64
    var newPowerSystem      uint64

    var newPassiveDraw              uint64
    var newActiveMiningSystemDraw   uint64
    var newActiveRefiningSystemDraw uint64

    switch structType {
    	case "Mining Rig":
	    	newCategory = "Planetary Struct"
	    	newType     = structType
            newAmbit    = "LAND"

            newMiningSystem   = 1;
            newRefiningSystem = 0;
            newPowerSystem    = 0;

            newPassiveDraw              = 10;
            newActiveMiningSystemDraw   = 20;
            newActiveRefiningSystemDraw = 0;

	    case "Refinery":
	        newCategory = "Planetary Struct"
		    newType     = structType
            newAmbit    = "LAND"

            newMiningSystem   = 0;
            newRefiningSystem = 1;
            newPowerSystem    = 0;

            newPassiveDraw              = 20;
            newActiveMiningSystemDraw   = 0;
            newActiveRefiningSystemDraw = 30;

	    case "Small Generator":
	        newCategory = "Planetary Struct"
		    newType     = structType
            newAmbit    = "LAND"

            newMiningSystem   = 0;
            newRefiningSystem = 0;
            newPowerSystem    = 1;

            newPassiveDraw              = 5;
            newActiveMiningSystemDraw   = 0;
            newActiveRefiningSystemDraw = 0;

	    default:
    }

	return Struct{
		Creator:  "",
		Owner: "",
		Status: "BUILDING",

		MaxHealth: 3,
		Health: 3,

		Type: newType,
        Category: newCategory,
        Ambit: newAmbit,
        Slot: 0,

        MiningSystem: newMiningSystem,
        RefiningSystem: newRefiningSystem,
        PowerSystem: newPowerSystem,

        PassiveDraw: newPassiveDraw,
        ActiveMiningSystemDraw: newActiveMiningSystemDraw,
        ActiveRefiningSystemDraw: newActiveRefiningSystemDraw,

        MiningSystemStatus: "INACTIVE",
        ActiveMiningSystemBlock: 0,

        RefiningSystemStatus: "INACTIVE",
        ActiveRefiningSystemBlock: 0,

	}
}



var Feature_enum = map[string]Feature {
	"featureless":                                      Featureless,
    "coordinated_global_shield_network":                FeatureCoordinatedGlobalShieldNetwork,
    "defensive_cannon":                                 FeatureDefensiveCannon,
    "indirect_combat_module":                           FeatureIndirectCombatModule,
    "last_resort":                                      FeatureLastResort,
	"ore_mining":                                       FeatureOreMining,
    "ore_refining":                                     FeatureOreRefining,
    "repair_network":                                   FeatureRepairNetwork,
    "stealth_mode":                                     FeatureStealthMode,
    "movable":                                          FeatureMovable,
    "low_orbit_ballistic_interceptor_network":          FeatureLowOrbitBallisticInterceptorNetwork,
    "advanced_low_orbit_ballistic_interceptor_network": FeatureAdvancedLowOrbitBallisticInterceptorNetwork,
    "orbital_jamming_station":                          FeatureOrbitalJammingStation,
    "advanced_orbital_jamming_station":                 FeatureAdvancedOrbitalJammingStation,
	"all":                                              FeatureAll,
}



type State uint64

const (
    // 1
	StateBuilt State = 1 << iota
	// 2
	StateOnline
	// 4
	StateStored
	// 8
	StateStealth
	// 16
    StateDestroyed
    // 32
    StateLocked // Unsure if needed
)

const (
    Stateless State = 0 << iota
	StateAll = StateBuilt | StateOnline | StateStored | StateStealth | StateLocked
)


var State_enum = map[string]State {
	"stateless":    Stateless,
    "built":        StateBuilt,
    "online":       StateOnline,
    "stored":       StateStored,
    "stealth":      StateStealth,
    "destroyed":    StateDestroyed,
    "locked":       StateLocked,
	"all":          StateAll,
}
