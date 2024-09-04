package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"

)

func CreateStructTypeGenesis() (genesisStructTypes []StructType) {
    var structType StructType

/*
    structType = StructType{
        Id: 1,
        Type: "something",
        Category: ObjectType_fleet,

        BuildDifficulty: 0,
        BuildDraw: 0,
        MaxHealth: 3,
        PassiveDraw: 0,

        PossibleAmbit: 0,
        Movable: true,
        SlotBound: true,

        PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,
        PrimaryWeaponControl: TechWeaponControl_guided,
        PrimaryWeaponCharge: 0,
        PrimaryWeaponAmbits: 0,
        PrimaryWeaponTargets: 0,
        PrimaryWeaponShots: 0,
        PrimaryWeaponDamage: 0,
        PrimaryWeaponBlockable: true,
        PrimaryWeaponCounterable: true,
        PrimaryWeaponRecoilDamage: 0,
        PrimaryWeaponShotSuccessRateNumerator: 1,
        PrimaryWeaponShotSuccessRateDenominator: 1,

        SecondaryWeapon:  TechActiveWeaponry_guidedWeaponry,
        SecondaryWeaponControl: TechWeaponControl_guided,
        SecondaryWeaponCharge: 0,
        SecondaryWeaponAmbits: 0,
        SecondaryWeaponTargets: 0,
        SecondaryWeaponShots: 0,
        SecondaryWeaponDamage: 0,
        SecondaryWeaponBlockable: true,
        SecondaryWeaponCounterable: true,
        SecondaryWeaponRecoilDamage: 0,
        SecondaryWeaponShotSuccessRateNumerator: 1,
        SecondaryWeaponShotSuccessRateDenominator: 1,


        PassiveWeaponry: TechPassiveWeaponry_counterAttack,
        UnitDefenses: TechUnitDefenses_noUnitDefenses,
        OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
        PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
        PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
        PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
        PowerGeneration: TechPowerGeneration_noPowerGeneration,

        ActivateCharge: 0,
        BuildCharge: 0,
        DefendChangeCharge: 0,
        MoveCharge: 0,
        OreMiningCharge: 0,
        OreRefiningCharge: 0,
        StealthActivateCharge: 0,


        AttackReduction: 0,
        AttackCounterable: true,
        StealthSystems: true,

        CounterAttack: 1,
        CounterAttackSameAmbit: 1,

        PostDestructionDamage: 0,
        GeneratingRate: 0,
        PlanetaryShieldContribution: 0,

        OreMiningDifficulty: 0,
        OreRefiningDifficulty: 0,

        UnguidedDefensiveSuccessRateNumerator: 0,
        UnguidedDefensiveSuccessRateDenominator: 1,

        GuidedDefensiveSuccessRateNumerator: 0,
        GuidedDefensiveSuccessRateDenominator: 1,

    }
    genesisStructTypes = append(genesisStructTypes, structType)
*/

    // Struct Type: Command
    structType = StructType{
     Id: 1,
     Type: "Command",
     Category: ObjectType_fleet,

     BuildLimit: 1,
     BuildDifficulty: 200,
     BuildDraw: 100,
     PassiveDraw: 50,
     MaxHealth: 6,

     PossibleAmbit: 52,
     Movable: true,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,
     PrimaryWeaponControl: TechWeaponControl_guided,
     PrimaryWeaponCharge: 12,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 1,
     PrimaryWeaponShots: 1,
     PrimaryWeaponDamage: 2,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 1,
     PrimaryWeaponShotSuccessRateDenominator: 1,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_strongCounterAttack,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 120,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 2,
     CounterAttackSameAmbit: 1,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Galactic Battleship
    structType = StructType{
     Id: 2,
     Type: "Galactic Battleship",
     Category: ObjectType_fleet,

     BuildLimit: 0,
     BuildDifficulty: 765,
     BuildDraw: 375,
     PassiveDraw: 225,
     MaxHealth: 3,

     PossibleAmbit: 30,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_unguidedWeaponry,
     PrimaryWeaponControl: TechWeaponControl_unguided,
     PrimaryWeaponCharge: 360,
     PrimaryWeaponAmbits: 38,
     PrimaryWeaponTargets: 1,
     PrimaryWeaponShots: 1,
     PrimaryWeaponDamage: 2,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 1,
     PrimaryWeaponShotSuccessRateDenominator: 1,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_counterAttack,
     UnitDefenses: TechUnitDefenses_signalJamming,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 1,
     CounterAttackSameAmbit: 1,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 2,
     GuidedDefensiveSuccessRateDenominator: 2,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Star Fighter
    structType = StructType{
     Id: 3,
     Type: "Star Fighter",
     Category: ObjectType_fleet,

     BuildLimit: 0,
     BuildDifficulty: 250,
     BuildDraw: 200,
     PassiveDraw: 100,
     MaxHealth: 3,

     PossibleAmbit: 30,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,
     PrimaryWeaponControl: TechWeaponControl_guided,
     PrimaryWeaponCharge: 12,
     PrimaryWeaponAmbits: 30,
     PrimaryWeaponTargets: 1,
     PrimaryWeaponShots: 1,
     PrimaryWeaponDamage: 2,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 1,
     PrimaryWeaponShotSuccessRateDenominator: 1,

     SecondaryWeapon:  TechActiveWeaponry_attackRun,
     SecondaryWeaponControl: TechWeaponControl_unguided,
     SecondaryWeaponCharge: 120,
     SecondaryWeaponAmbits: 30,
     SecondaryWeaponTargets: 1,
     SecondaryWeaponShots: 3,
     SecondaryWeaponDamage: 1,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 1,
     SecondaryWeaponShotSuccessRateDenominator: 3,

     PassiveWeaponry: TechPassiveWeaponry_counterAttack,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 1,
     CounterAttackSameAmbit: 1,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Space Frigate
    structType = StructType{
     Id: 4,
     Type: "Space Frigate",
     Category: ObjectType_fleet,

     BuildLimit: 0,
     BuildDifficulty: 450,
     BuildDraw: 150,
     PassiveDraw: 75,
     MaxHealth: 3,

     PossibleAmbit: 30,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,
     PrimaryWeaponControl: TechWeaponControl_guided,
     PrimaryWeaponCharge: 120,
     PrimaryWeaponAmbits: 44,
     PrimaryWeaponTargets: 1,
     PrimaryWeaponShots: 1,
     PrimaryWeaponDamage: 2,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 1,
     PrimaryWeaponShotSuccessRateDenominator: 1,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_counterAttack,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 1,
     CounterAttackSameAmbit: 1,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Fighter Jet
    structType = StructType{
     Id: 5,
     Type: "Fighter Jet",
     Category: ObjectType_fleet,

     BuildLimit: 0,
     BuildDifficulty: 215,
     BuildDraw: 225,
     PassiveDraw: 150,
     MaxHealth: 3,

     PossibleAmbit: 14,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,
     PrimaryWeaponControl: TechWeaponControl_guided,
     PrimaryWeaponCharge: 12,
     PrimaryWeaponAmbits: 14,
     PrimaryWeaponTargets: 1,
     PrimaryWeaponShots: 1,
     PrimaryWeaponDamage: 2,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 1,
     PrimaryWeaponShotSuccessRateDenominator: 1,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_counterAttack,
     UnitDefenses: TechUnitDefenses_signalJamming,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 1,
     CounterAttackSameAmbit: 1,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 2,
     GuidedDefensiveSuccessRateDenominator: 2,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Stealth Bomber
    structType = StructType{
     Id: 6,
     Type: "Stealth Bomber",
     Category: ObjectType_fleet,

     BuildLimit: 0,
     BuildDifficulty: 455,
     BuildDraw: 200,
     PassiveDraw: 125,
     MaxHealth: 3,

     PossibleAmbit: 14,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,
     PrimaryWeaponControl: TechWeaponControl_guided,
     PrimaryWeaponCharge: 120,
     PrimaryWeaponAmbits: 8,
     PrimaryWeaponTargets: 1,
     PrimaryWeaponShots: 1,
     PrimaryWeaponDamage: 2,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 1,
     PrimaryWeaponShotSuccessRateDenominator: 1,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_counterAttack,
     UnitDefenses: TechUnitDefenses_stealthMode,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 12,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: true,

     CounterAttack: 1,
     CounterAttackSameAmbit: 1,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: High Altitude Interceptor
    structType = StructType{
     Id: 7,
     Type: "High Altitude Interceptor",
     Category: ObjectType_fleet,

     BuildLimit: 0,
     BuildDifficulty: 460,
     BuildDraw: 225,
     PassiveDraw: 125,
     MaxHealth: 3,

     PossibleAmbit: 14,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,
     PrimaryWeaponControl: TechWeaponControl_guided,
     PrimaryWeaponCharge: 120,
     PrimaryWeaponAmbits: 44,
     PrimaryWeaponTargets: 1,
     PrimaryWeaponShots: 1,
     PrimaryWeaponDamage: 2,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 1,
     PrimaryWeaponShotSuccessRateDenominator: 1,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_counterAttack,
     UnitDefenses: TechUnitDefenses_defensiveManeuver,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 1,
     CounterAttackSameAmbit: 1,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 2,
     UnguidedDefensiveSuccessRateDenominator: 2,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Artillery
    structType = StructType{
     Id: 8,
     Type: "Artillery",
     Category: ObjectType_fleet,

     BuildLimit: 0,
     BuildDifficulty: 305,
     BuildDraw: 125,
     PassiveDraw: 75,
     MaxHealth: 3,

     PossibleAmbit: 6,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_unguidedWeaponry,
     PrimaryWeaponControl: TechWeaponControl_unguided,
     PrimaryWeaponCharge: 120,
     PrimaryWeaponAmbits: 8,
     PrimaryWeaponTargets: 1,
     PrimaryWeaponShots: 1,
     PrimaryWeaponDamage: 2,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 1,
     PrimaryWeaponShotSuccessRateDenominator: 1,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_indirectCombatModule,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: false,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Tank
    structType = StructType{
     Id: 9,
     Type: "Tank",
     Category: ObjectType_fleet,

     BuildLimit: 0,
     BuildDifficulty: 220,
     BuildDraw: 200,
     PassiveDraw: 75,
     MaxHealth: 3,

     PossibleAmbit: 6,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_unguidedWeaponry,
     PrimaryWeaponControl: TechWeaponControl_unguided,
     PrimaryWeaponCharge: 12,
     PrimaryWeaponAmbits: 6,
     PrimaryWeaponTargets: 1,
     PrimaryWeaponShots: 1,
     PrimaryWeaponDamage: 2,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 1,
     PrimaryWeaponShotSuccessRateDenominator: 1,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_counterAttack,
     UnitDefenses: TechUnitDefenses_armour,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 1,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 1,
     CounterAttackSameAmbit: 1,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: SAM Launcher
    structType = StructType{
     Id: 10,
     Type: "SAM Launcher",
     Category: ObjectType_fleet,

     BuildLimit: 0,
     BuildDifficulty: 450,
     BuildDraw: 150,
     PassiveDraw: 75,
     MaxHealth: 3,

     PossibleAmbit: 6,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,
     PrimaryWeaponControl: TechWeaponControl_guided,
     PrimaryWeaponCharge: 120,
     PrimaryWeaponAmbits: 44,
     PrimaryWeaponTargets: 1,
     PrimaryWeaponShots: 1,
     PrimaryWeaponDamage: 2,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 1,
     PrimaryWeaponShotSuccessRateDenominator: 1,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_counterAttack,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 1,
     CounterAttackSameAmbit: 1,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Cruiser
    structType = StructType{
     Id: 11,
     Type: "Cruiser",
     Category: ObjectType_fleet,

     BuildLimit: 0,
     BuildDifficulty: 515,
     BuildDraw: 325,
     PassiveDraw: 200,
     MaxHealth: 3,

     PossibleAmbit: 2,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,
     PrimaryWeaponControl: TechWeaponControl_guided,
     PrimaryWeaponCharge: 120,
     PrimaryWeaponAmbits: 8,
     PrimaryWeaponTargets: 1,
     PrimaryWeaponShots: 1,
     PrimaryWeaponDamage: 2,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 1,
     PrimaryWeaponShotSuccessRateDenominator: 1,

     SecondaryWeapon:  TechActiveWeaponry_unguidedWeaponry,
     SecondaryWeaponControl: TechWeaponControl_unguided,
     SecondaryWeaponCharge: 12,
     SecondaryWeaponAmbits: 14,
     SecondaryWeaponTargets: 1,
     SecondaryWeaponShots: 1,
     SecondaryWeaponDamage: 2,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 1,
     SecondaryWeaponShotSuccessRateDenominator: 1,

     PassiveWeaponry: TechPassiveWeaponry_counterAttack,
     UnitDefenses: TechUnitDefenses_signalJamming,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 1,
     CounterAttackSameAmbit: 1,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 2,
     GuidedDefensiveSuccessRateDenominator: 2,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Destroyer
    structType = StructType{
     Id: 12,
     Type: "Destroyer",
     Category: ObjectType_fleet,

     BuildLimit: 0,
     BuildDifficulty: 600,
     BuildDraw: 200,
     PassiveDraw: 100,
     MaxHealth: 3,

     PossibleAmbit: 2,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,
     PrimaryWeaponControl: TechWeaponControl_guided,
     PrimaryWeaponCharge: 120,
     PrimaryWeaponAmbits: 16,
     PrimaryWeaponTargets: 1,
     PrimaryWeaponShots: 1,
     PrimaryWeaponDamage: 2,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 1,
     PrimaryWeaponShotSuccessRateDenominator: 1,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_advancedCounterAttack,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 1,
     CounterAttackSameAmbit: 2,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Sub
    structType = StructType{
     Id: 13,
     Type: "Sub",
     Category: ObjectType_fleet,

     BuildLimit: 0,
     BuildDifficulty: 455,
     BuildDraw: 200,
     PassiveDraw: 125,
     MaxHealth: 3,

     PossibleAmbit: 2,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,
     PrimaryWeaponControl: TechWeaponControl_guided,
     PrimaryWeaponCharge: 120,
     PrimaryWeaponAmbits: 32,
     PrimaryWeaponTargets: 1,
     PrimaryWeaponShots: 1,
     PrimaryWeaponDamage: 2,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 1,
     PrimaryWeaponShotSuccessRateDenominator: 1,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_counterAttack,
     UnitDefenses: TechUnitDefenses_stealthMode,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 12,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: true,

     CounterAttack: 1,
     CounterAttackSameAmbit: 1,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Mine Shaft
    structType = StructType{
     Id: 14,
     Type: "Mine Shaft",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 700,
     BuildDraw: 100,
     PassiveDraw: 500,
     MaxHealth: 3,

     PossibleAmbit: 6,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     PrimaryWeaponControl: TechWeaponControl_noWeaponControl,
     PrimaryWeaponCharge: 0,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 0,
     PrimaryWeaponShots: 0,
     PrimaryWeaponDamage: 0,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 0,
     PrimaryWeaponShotSuccessRateDenominator: 0,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_oreMiningRig,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 360,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 14000,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Deep Ocean Minig Rig
    structType = StructType{
     Id: 15,
     Type: "Deep Ocean Minig Rig",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 700,
     BuildDraw: 100,
     PassiveDraw: 500,
     MaxHealth: 3,

     PossibleAmbit: 2,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     PrimaryWeaponControl: TechWeaponControl_noWeaponControl,
     PrimaryWeaponCharge: 0,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 0,
     PrimaryWeaponShots: 0,
     PrimaryWeaponDamage: 0,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 0,
     PrimaryWeaponShotSuccessRateDenominator: 0,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_oreMiningRig,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 360,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 14000,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Ore Refinery
    structType = StructType{
     Id: 16,
     Type: "Ore Refinery",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 700,
     BuildDraw: 100,
     PassiveDraw: 500,
     MaxHealth: 3,

     PossibleAmbit: 6,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     PrimaryWeaponControl: TechWeaponControl_noWeaponControl,
     PrimaryWeaponCharge: 0,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 0,
     PrimaryWeaponShots: 0,
     PrimaryWeaponDamage: 0,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 0,
     PrimaryWeaponShotSuccessRateDenominator: 0,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_oreRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 360,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 1400,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Water Cooled Ore Refinery
    structType = StructType{
     Id: 17,
     Type: "Water Cooled Ore Refinery",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 700,
     BuildDraw: 100,
     PassiveDraw: 500,
     MaxHealth: 3,

     PossibleAmbit: 2,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     PrimaryWeaponControl: TechWeaponControl_noWeaponControl,
     PrimaryWeaponCharge: 0,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 0,
     PrimaryWeaponShots: 0,
     PrimaryWeaponDamage: 0,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 0,
     PrimaryWeaponShotSuccessRateDenominator: 0,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_oreRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 360,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 1400,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Orbital Shield Generator
    structType = StructType{
     Id: 18,
     Type: "Orbital Shield Generator",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 720,
     BuildDraw: 100,
     PassiveDraw: 200,
     MaxHealth: 3,

     PossibleAmbit: 30,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     PrimaryWeaponControl: TechWeaponControl_noWeaponControl,
     PrimaryWeaponCharge: 0,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 0,
     PrimaryWeaponShots: 0,
     PrimaryWeaponDamage: 0,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 0,
     PrimaryWeaponShotSuccessRateDenominator: 0,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_monitoringStation,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 1440,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Jamming Satellite
    structType = StructType{
     Id: 19,
     Type: "Jamming Satellite",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 2880,
     BuildDraw: 450,
     PassiveDraw: 600,
     MaxHealth: 3,

     PossibleAmbit: 30,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     PrimaryWeaponControl: TechWeaponControl_noWeaponControl,
     PrimaryWeaponCharge: 0,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 0,
     PrimaryWeaponShots: 0,
     PrimaryWeaponDamage: 0,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 0,
     PrimaryWeaponShotSuccessRateDenominator: 0,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_coordinatedReserveResponseTracker,
     PlanetaryDefenses: TechPlanetaryDefenses_lowOrbitBallisticInterceptorNetwork,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 2880,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Ore Bunker
    structType = StructType{
     Id: 20,
     Type: "Ore Bunker",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 3600,
     BuildDraw: 200,
     PassiveDraw: 750,
     MaxHealth: 3,

     PossibleAmbit: 6,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     PrimaryWeaponControl: TechWeaponControl_noWeaponControl,
     PrimaryWeaponCharge: 0,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 0,
     PrimaryWeaponShots: 0,
     PrimaryWeaponDamage: 0,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 0,
     PrimaryWeaponShotSuccessRateDenominator: 0,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_oreBunker,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 7200,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Defense Cannon
    structType = StructType{
     Id: 21,
     Type: "Defense Cannon",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 2880,
     BuildDraw: 450,
     PassiveDraw: 600,
     MaxHealth: 3,

     PossibleAmbit: 6,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     PrimaryWeaponControl: TechWeaponControl_noWeaponControl,
     PrimaryWeaponCharge: 0,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 0,
     PrimaryWeaponShots: 0,
     PrimaryWeaponDamage: 0,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 0,
     PrimaryWeaponShotSuccessRateDenominator: 0,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_coordinatedReserveResponseTracker,
     PlanetaryDefenses: TechPlanetaryDefenses_defensiveCannon,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 2880,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Offshore Defensive Turret
    structType = StructType{
     Id: 22,
     Type: "Offshore Defensive Turret",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 2880,
     BuildDraw: 450,
     PassiveDraw: 600,
     MaxHealth: 3,

     PossibleAmbit: 2,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     PrimaryWeaponControl: TechWeaponControl_noWeaponControl,
     PrimaryWeaponCharge: 0,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 0,
     PrimaryWeaponShots: 0,
     PrimaryWeaponDamage: 0,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 0,
     PrimaryWeaponShotSuccessRateDenominator: 0,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_coordinatedReserveResponseTracker,
     PlanetaryDefenses: TechPlanetaryDefenses_defensiveCannon,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_noPowerGeneration,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 2880,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Small Generator
    structType = StructType{
     Id: 23,
     Type: "Small Generator",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 700,
     BuildDraw: 500,
     PassiveDraw: 500,
     MaxHealth: 3,

     PossibleAmbit: 6,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     PrimaryWeaponControl: TechWeaponControl_noWeaponControl,
     PrimaryWeaponCharge: 0,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 0,
     PrimaryWeaponShots: 0,
     PrimaryWeaponDamage: 0,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 0,
     PrimaryWeaponShotSuccessRateDenominator: 0,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_smallGenerator,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 10,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Grid Booster
    structType = StructType{
     Id: 24,
     Type: "Grid Booster",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 1440,
     BuildDraw: 50000,
     PassiveDraw: 5000,
     MaxHealth: 3,

     PossibleAmbit: 6,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     PrimaryWeaponControl: TechWeaponControl_noWeaponControl,
     PrimaryWeaponCharge: 0,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 0,
     PrimaryWeaponShots: 0,
     PrimaryWeaponDamage: 0,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 0,
     PrimaryWeaponShotSuccessRateDenominator: 0,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_mediumGenerator,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 100,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Full-Scale Power Plant
    structType = StructType{
     Id: 25,
     Type: "Full-Scale Power Plant",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 5000,
     BuildDraw: 500000,
     PassiveDraw: 50000,
     MaxHealth: 3,

     PossibleAmbit: 6,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     PrimaryWeaponControl: TechWeaponControl_noWeaponControl,
     PrimaryWeaponCharge: 0,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 0,
     PrimaryWeaponShots: 0,
     PrimaryWeaponDamage: 0,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 0,
     PrimaryWeaponShotSuccessRateDenominator: 0,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_largeGenerator,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 1000,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)



    // Struct Type: Deep Sea Generator
    structType = StructType{
     Id: 26,
     Type: "Deep Sea Generator",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 700,
     BuildDraw: 500,
     PassiveDraw: 500,
     MaxHealth: 3,

     PossibleAmbit: 2,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     PrimaryWeaponControl: TechWeaponControl_noWeaponControl,
     PrimaryWeaponCharge: 0,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 0,
     PrimaryWeaponShots: 0,
     PrimaryWeaponDamage: 0,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 0,
     PrimaryWeaponShotSuccessRateDenominator: 0,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_smallGenerator,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 10,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)


    // Struct Type: Off-Shore Power Plant
    structType = StructType{
     Id: 27,
     Type: "Off-Shore Power Plant",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 1440,
     BuildDraw: 50000,
     PassiveDraw: 5000,
     MaxHealth: 3,

     PossibleAmbit: 2,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     PrimaryWeaponControl: TechWeaponControl_noWeaponControl,
     PrimaryWeaponCharge: 0,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 0,
     PrimaryWeaponShots: 0,
     PrimaryWeaponDamage: 0,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 0,
     PrimaryWeaponShotSuccessRateDenominator: 0,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_mediumGenerator,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 100,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)


    // Struct Type: Heavy Water Power Generator
    structType = StructType{
     Id: 28,
     Type: "Heavy Water Power Generator",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 5000,
     BuildDraw: 500000,
     PassiveDraw: 50000,
     MaxHealth: 3,

     PossibleAmbit: 2,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     PrimaryWeaponControl: TechWeaponControl_noWeaponControl,
     PrimaryWeaponCharge: 0,
     PrimaryWeaponAmbits: 0,
     PrimaryWeaponTargets: 0,
     PrimaryWeaponShots: 0,
     PrimaryWeaponDamage: 0,
     PrimaryWeaponBlockable: true,
     PrimaryWeaponCounterable: true,
     PrimaryWeaponRecoilDamage: 0,
     PrimaryWeaponShotSuccessRateNumerator: 0,
     PrimaryWeaponShotSuccessRateDenominator: 0,

     SecondaryWeapon:  TechActiveWeaponry_noActiveWeaponry,
     SecondaryWeaponControl: TechWeaponControl_noWeaponControl,
     SecondaryWeaponCharge: 0,
     SecondaryWeaponAmbits: 0,
     SecondaryWeaponTargets: 0,
     SecondaryWeaponShots: 0,
     SecondaryWeaponDamage: 0,
     SecondaryWeaponBlockable:  true,
     SecondaryWeaponCounterable: true,
     SecondaryWeaponRecoilDamage:0,
     SecondaryWeaponShotSuccessRateNumerator: 0,
     SecondaryWeaponShotSuccessRateDenominator: 0,

     PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,
     UnitDefenses: TechUnitDefenses_noUnitDefenses,
     OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,
     PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,
     PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,
     PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,
     PowerGeneration: TechPowerGeneration_largeGenerator,

     ActivateCharge: 360,
     BuildCharge: 12,
     DefendChangeCharge: 12,
     MoveCharge: 0,
     OreMiningCharge: 0,
     OreRefiningCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 1000,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)
  

    return
}




