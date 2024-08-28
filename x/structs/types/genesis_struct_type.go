package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"

)

func CreateStructTypeGenesis() (genesisStructTypes []StructType) {
    var structType StructType

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


    return
}




