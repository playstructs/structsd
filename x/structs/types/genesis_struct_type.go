package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"

)

func CreateStructTypeGenesis() (genesisStructTypes []StructType) {
    var structType StructType

// Struct Type: Command Ship
structType = StructType{                                                 
 Id: 1,                                               
 Type: "Command Ship",                                      
 Class: "Command Ship",                                     
 ClassAbbreviation: "CMD Ship",             
 DefaultCosmeticModelNumber: "ST-21", 
 DefaultCosmeticName: "Spearpoint",               
 Category: ObjectType_fleet,       
                                                                         
 BuildLimit: 1,                                 
 BuildDifficulty: 200,                       
 BuildDraw: 50000,                                   
 PassiveDraw: 50000,                               
 MaxHealth: 6,                                   
                                                                         
 PossibleAmbit: 30,           
 Movable: true,                                  
 SlotBound: true,                          
                                                                                                   
 PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,                                 
 PrimaryWeaponControl: TechWeaponControl_guided,                     
 PrimaryWeaponCharge: 1,                                         
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
            
 PassiveWeaponry: TechPassiveWeaponry_strongCounterAttack,           
 UnitDefenses: TechUnitDefenses_noUnitDefenses,                    
 OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,  
 PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,     
 PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,           
 PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,   
 PowerGeneration: TechPowerGeneration_noPowerGeneration,           
            
 ActivateCharge: 1,                
 BuildCharge: 8,                      
 DefendChangeCharge: 1,        
 MoveCharge: 8,                        
 StealthActivateCharge: 0,  
            
 AttackReduction: 0,      
 AttackCounterable: true,  
 StealthSystems: false,        
            
 CounterAttack: 2,                  
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

 TriggerRaidDefeatByDestruction: true,
                                                         
 } 
genesisStructTypes = append(genesisStructTypes, structType) 
  
  
// Struct Type: Battleship                               
structType = StructType{                                                 
 Id: 2,                                               
 Type: "Battleship",                                      
 Class: "Battleship",                                     
 ClassAbbreviation: "Battleship",             
 DefaultCosmeticModelNumber: "CT-C", 
 DefaultCosmeticName: "Cataclysm",               
 Category: ObjectType_fleet,       
                                                                         
 BuildLimit: 0,                                 
 BuildDifficulty: 765,                       
 BuildDraw: 135000,                                   
 PassiveDraw: 135000,                               
 MaxHealth: 3,                                   
                                                                         
 PossibleAmbit: 16,           
 Movable: false,                                  
 SlotBound: true,                          
                                                                                                   
 PrimaryWeapon:  TechActiveWeaponry_unguidedWeaponry,                                 
 PrimaryWeaponControl: TechWeaponControl_unguided,                     
 PrimaryWeaponCharge: 20,                                         
 PrimaryWeaponAmbits: 22,                          
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
            
 ActivateCharge: 1,                
 BuildCharge: 8,                      
 DefendChangeCharge: 1,        
 MoveCharge: 0,                        
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
 GuidedDefensiveSuccessRateDenominator: 3,     
                                                         
 } 
genesisStructTypes = append(genesisStructTypes, structType) 
  
  
// Struct Type: Starfighter                               
structType = StructType{                                                 
 Id: 3,                                               
 Type: "Starfighter",                                      
 Class: "Starfighter",                                     
 ClassAbbreviation: "Starfighter",             
 DefaultCosmeticModelNumber: "GB-1", 
 DefaultCosmeticName: "Gambit",               
 Category: ObjectType_fleet,       
                                                                         
 BuildLimit: 0,                                 
 BuildDifficulty: 250,                       
 BuildDraw: 100000,                                   
 PassiveDraw: 100000,                               
 MaxHealth: 3,                                   
                                                                         
 PossibleAmbit: 16,           
 Movable: false,                                  
 SlotBound: true,                          
                                                                                                   
 PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,                                 
 PrimaryWeaponControl: TechWeaponControl_guided,                     
 PrimaryWeaponCharge: 1,                                         
 PrimaryWeaponAmbits: 16,                          
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
 SecondaryWeaponCharge: 8,                                         
 SecondaryWeaponAmbits: 16,                          
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
            
 ActivateCharge: 1,                
 BuildCharge: 8,                      
 DefendChangeCharge: 1,        
 MoveCharge: 0,                        
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
  
  
// Struct Type: Frigate                               
structType = StructType{                                                 
 Id: 4,                                               
 Type: "Frigate",                                      
 Class: "Frigate",                                     
 ClassAbbreviation: "Frigate",             
 DefaultCosmeticModelNumber: "SK-4", 
 DefaultCosmeticName: "Skylight",               
 Category: ObjectType_fleet,       
                                                                         
 BuildLimit: 0,                                 
 BuildDifficulty: 450,                       
 BuildDraw: 75000,                                   
 PassiveDraw: 75000,                               
 MaxHealth: 3,                                   
                                                                         
 PossibleAmbit: 16,           
 Movable: false,                                  
 SlotBound: true,                          
                                                                                                   
 PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,                                 
 PrimaryWeaponControl: TechWeaponControl_guided,                     
 PrimaryWeaponCharge: 8,                                         
 PrimaryWeaponAmbits: 24,                          
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
            
 ActivateCharge: 1,                
 BuildCharge: 8,                      
 DefendChangeCharge: 1,        
 MoveCharge: 0,                        
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
  
  
// Struct Type: Pursuit Fighter                               
structType = StructType{                                                 
 Id: 5,                                               
 Type: "Pursuit Fighter",                                      
 Class: "Pursuit Fighter",                                     
 ClassAbbreviation: "Pursuit Fighter",             
 DefaultCosmeticModelNumber: "SQ-11", 
 DefaultCosmeticName: "Squall",               
 Category: ObjectType_fleet,       
                                                                         
 BuildLimit: 0,                                 
 BuildDifficulty: 215,                       
 BuildDraw: 60000,                                   
 PassiveDraw: 60000,                               
 MaxHealth: 3,                                   
                                                                         
 PossibleAmbit: 8,           
 Movable: false,                                  
 SlotBound: true,                          
                                                                                                   
 PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,                                 
 PrimaryWeaponControl: TechWeaponControl_guided,                     
 PrimaryWeaponCharge: 1,                                         
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
 UnitDefenses: TechUnitDefenses_signalJamming,                    
 OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,  
 PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,     
 PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,           
 PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,   
 PowerGeneration: TechPowerGeneration_noPowerGeneration,           
            
 ActivateCharge: 1,                
 BuildCharge: 8,                      
 DefendChangeCharge: 1,        
 MoveCharge: 0,                        
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
 GuidedDefensiveSuccessRateDenominator: 3,     
                                                         
 } 
genesisStructTypes = append(genesisStructTypes, structType) 
  
  
// Struct Type: Stealth Bomber                               
structType = StructType{                                                 
 Id: 6,                                               
 Type: "Stealth Bomber",                                      
 Class: "Stealth Bomber",                                     
 ClassAbbreviation: "Stealth Bomber",             
 DefaultCosmeticModelNumber: "RT-4", 
 DefaultCosmeticName: "Rolling Thunder",               
 Category: ObjectType_fleet,       
                                                                         
 BuildLimit: 0,                                 
 BuildDifficulty: 455,                       
 BuildDraw: 125000,                                   
 PassiveDraw: 125000,                               
 MaxHealth: 3,                                   
                                                                         
 PossibleAmbit: 8,           
 Movable: false,                                  
 SlotBound: true,                          
                                                                                                   
 PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,                                 
 PrimaryWeaponControl: TechWeaponControl_guided,                     
 PrimaryWeaponCharge: 8,                                         
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
 UnitDefenses: TechUnitDefenses_stealthMode,                    
 OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,  
 PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,     
 PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,           
 PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,   
 PowerGeneration: TechPowerGeneration_noPowerGeneration,           
            
 ActivateCharge: 1,                
 BuildCharge: 8,                      
 DefendChangeCharge: 1,        
 MoveCharge: 0,                        
 StealthActivateCharge: 1,  
            
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
 Class: "High Altitude Interceptor",                                     
 ClassAbbreviation: "Interceptor",             
 DefaultCosmeticModelNumber: "SKMR", 
 DefaultCosmeticName: "Skimmer",               
 Category: ObjectType_fleet,       
                                                                         
 BuildLimit: 0,                                 
 BuildDifficulty: 460,                       
 BuildDraw: 125000,                                   
 PassiveDraw: 125000,                               
 MaxHealth: 3,                                   
                                                                         
 PossibleAmbit: 8,           
 Movable: false,                                  
 SlotBound: true,                          
                                                                                                   
 PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,                                 
 PrimaryWeaponControl: TechWeaponControl_guided,                     
 PrimaryWeaponCharge: 8,                                         
 PrimaryWeaponAmbits: 24,                          
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
            
 ActivateCharge: 1,                
 BuildCharge: 8,                      
 DefendChangeCharge: 1,        
 MoveCharge: 0,                        
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
 UnguidedDefensiveSuccessRateDenominator: 3, 
            
 GuidedDefensiveSuccessRateNumerator: 0,       
 GuidedDefensiveSuccessRateDenominator: 0,     
                                                         
 } 
genesisStructTypes = append(genesisStructTypes, structType) 
  
  
// Struct Type: Mobile Artillery                               
structType = StructType{                                                 
 Id: 8,                                               
 Type: "Mobile Artillery",                                      
 Class: "Mobile Artillery",                                     
 ClassAbbreviation: "Mobile Artillery",             
 DefaultCosmeticModelNumber: "AC-4", 
 DefaultCosmeticName: "Archer",               
 Category: ObjectType_fleet,       
                                                                         
 BuildLimit: 0,                                 
 BuildDifficulty: 305,                       
 BuildDraw: 75000,                                   
 PassiveDraw: 75000,                               
 MaxHealth: 3,                                   
                                                                         
 PossibleAmbit: 4,           
 Movable: false,                                  
 SlotBound: true,                          
                                                                                                   
 PrimaryWeapon:  TechActiveWeaponry_unguidedWeaponry,                                 
 PrimaryWeaponControl: TechWeaponControl_unguided,                     
 PrimaryWeaponCharge: 8,                                         
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
            
 PassiveWeaponry: TechPassiveWeaponry_noPassiveWeaponry,           
 UnitDefenses: TechUnitDefenses_indirectCombatModule,                    
 OreReserveDefenses: TechOreReserveDefenses_noOreReserveDefenses,  
 PlanetaryDefenses: TechPlanetaryDefenses_noPlanetaryDefense,     
 PlanetaryMining: TechPlanetaryMining_noPlanetaryMining,           
 PlanetaryRefinery: TechPlanetaryRefineries_noPlanetaryRefinery,   
 PowerGeneration: TechPowerGeneration_noPowerGeneration,           
            
 ActivateCharge: 1,                
 BuildCharge: 8,                      
 DefendChangeCharge: 1,        
 MoveCharge: 0,                        
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
 Class: "Tank",                                     
 ClassAbbreviation: "Tank",             
 DefaultCosmeticModelNumber: "BR-9", 
 DefaultCosmeticName: "Breakaway",               
 Category: ObjectType_fleet,       
                                                                         
 BuildLimit: 0,                                 
 BuildDifficulty: 220,                       
 BuildDraw: 75000,                                   
 PassiveDraw: 75000,                               
 MaxHealth: 3,                                   
                                                                         
 PossibleAmbit: 4,           
 Movable: false,                                  
 SlotBound: true,                          
                                                                                                   
 PrimaryWeapon:  TechActiveWeaponry_unguidedWeaponry,                                 
 PrimaryWeaponControl: TechWeaponControl_unguided,                     
 PrimaryWeaponCharge: 1,                                         
 PrimaryWeaponAmbits: 4,                          
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
            
 ActivateCharge: 1,                
 BuildCharge: 8,                      
 DefendChangeCharge: 1,        
 MoveCharge: 0,                        
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
 Class: "SAM Launcher",                                     
 ClassAbbreviation: "SAM Launcher",             
 DefaultCosmeticModelNumber: "LG-5", 
 DefaultCosmeticName: "Longshot",               
 Category: ObjectType_fleet,       
                                                                         
 BuildLimit: 0,                                 
 BuildDifficulty: 450,                       
 BuildDraw: 75000,                                   
 PassiveDraw: 75000,                               
 MaxHealth: 3,                                   
                                                                         
 PossibleAmbit: 4,           
 Movable: false,                                  
 SlotBound: true,                          
                                                                                                   
 PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,                                 
 PrimaryWeaponControl: TechWeaponControl_guided,                     
 PrimaryWeaponCharge: 8,                                         
 PrimaryWeaponAmbits: 24,                          
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
            
 ActivateCharge: 1,                
 BuildCharge: 8,                      
 DefendChangeCharge: 1,        
 MoveCharge: 0,                        
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
 Class: "Cruiser",                                     
 ClassAbbreviation: "Cruiser",             
 DefaultCosmeticModelNumber: "HD-44", 
 DefaultCosmeticName: "Hydra",               
 Category: ObjectType_fleet,       
                                                                         
 BuildLimit: 0,                                 
 BuildDifficulty: 515,                       
 BuildDraw: 110000,                                   
 PassiveDraw: 110000,                               
 MaxHealth: 3,                                   
                                                                         
 PossibleAmbit: 2,           
 Movable: false,                                  
 SlotBound: true,                          
                                                                                                   
 PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,                                 
 PrimaryWeaponControl: TechWeaponControl_guided,                     
 PrimaryWeaponCharge: 8,                                         
 PrimaryWeaponAmbits: 6,                          
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
 SecondaryWeaponCharge: 1,                                         
 SecondaryWeaponAmbits: 8,                          
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
            
 ActivateCharge: 1,                
 BuildCharge: 8,                      
 DefendChangeCharge: 1,        
 MoveCharge: 0,                        
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
 GuidedDefensiveSuccessRateDenominator: 3,     
                                                         
 } 
genesisStructTypes = append(genesisStructTypes, structType) 
  
  
// Struct Type: Destroyer                               
structType = StructType{                                                 
 Id: 12,                                               
 Type: "Destroyer",                                      
 Class: "Destroyer",                                     
 ClassAbbreviation: "Destroyer",             
 DefaultCosmeticModelNumber: "KR-3", 
 DefaultCosmeticName: "Kraken",               
 Category: ObjectType_fleet,       
                                                                         
 BuildLimit: 0,                                 
 BuildDifficulty: 600,                       
 BuildDraw: 100000,                                   
 PassiveDraw: 100000,                               
 MaxHealth: 3,                                   
                                                                         
 PossibleAmbit: 2,           
 Movable: false,                                  
 SlotBound: true,                          
                                                                                                   
 PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,                                 
 PrimaryWeaponControl: TechWeaponControl_guided,                     
 PrimaryWeaponCharge: 8,                                         
 PrimaryWeaponAmbits: 10,                          
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
            
 ActivateCharge: 1,                
 BuildCharge: 8,                      
 DefendChangeCharge: 1,        
 MoveCharge: 0,                        
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
  
  
    // Struct Type: Submersible
    structType = StructType{
     Id: 13,
     Type: "Submersible",
     Class: "Submersible",
     ClassAbbreviation: "Submersible",
     DefaultCosmeticModelNumber: "LV-2",
     DefaultCosmeticName: "Leviathan",
     Category: ObjectType_fleet,

     BuildLimit: 0,
     BuildDifficulty: 455,
     BuildDraw: 125000,
     PassiveDraw: 125000,
     MaxHealth: 3,

     PossibleAmbit: 2,
     Movable: false,
     SlotBound: true,

     PrimaryWeapon:  TechActiveWeaponry_guidedWeaponry,
     PrimaryWeaponControl: TechWeaponControl_guided,
     PrimaryWeaponCharge: 8,
     PrimaryWeaponAmbits: 18,
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

     ActivateCharge: 1,
     BuildCharge: 8,
     DefendChangeCharge: 1,
     MoveCharge: 0,
     StealthActivateCharge: 1,

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


    // Struct Type: Ore Extractor
    structType = StructType{
     Id: 14,
     Type: "Ore Extractor",
     Class: "Ore Extractor",
     ClassAbbreviation: "Extractor",
     DefaultCosmeticModelNumber: "LS-0",
     DefaultCosmeticName: "Lasersword",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 700,
     BuildDraw: 500000,
     PassiveDraw: 500000,
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

     ActivateCharge: 1,
     BuildCharge: 8,
     DefendChangeCharge: 1,
     MoveCharge: 0,
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
     Id: 15,
     Type: "Ore Refinery",
     Class: "Ore Refinery",
     ClassAbbreviation: "Refinery",
     DefaultCosmeticModelNumber: "GR-3",
     DefaultCosmeticName: "Greybox",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 700,
     BuildDraw: 500000,
     PassiveDraw: 500000,
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

     ActivateCharge: 1,
     BuildCharge: 8,
     DefendChangeCharge: 1,
     MoveCharge: 0,
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
     OreRefiningDifficulty: 28000,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)


    // Struct Type: Orbital Shield Generator
    structType = StructType{
     Id: 16,
     Type: "Orbital Shield Generator",
     Class: "Orbital Shield Generator",
     ClassAbbreviation: "Orb. Shield",
     DefaultCosmeticModelNumber: "T70",
     DefaultCosmeticName: "Shieldwall",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 720,
     BuildDraw: 200000,
     PassiveDraw: 200000,
     MaxHealth: 3,

     PossibleAmbit: 16,
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

     ActivateCharge: 1,
     BuildCharge: 8,
     DefendChangeCharge: 1,
     MoveCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 1500,

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
     Id: 17,
     Type: "Jamming Satellite",
     Class: "Jamming Satellite",
     ClassAbbreviation: "Jamming Sat.",
     DefaultCosmeticModelNumber: "OB-6",
     DefaultCosmeticName: "Observer",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 2880,
     BuildDraw: 600000,
     PassiveDraw: 600000,
     MaxHealth: 3,

     PossibleAmbit: 16,
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

     ActivateCharge: 1,
     BuildCharge: 8,
     DefendChangeCharge: 1,
     MoveCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 4500,

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
     Id: 18,
     Type: "Ore Bunker",
     Class: "Ore Bunker",
     ClassAbbreviation: "Ore Bunker",
     DefaultCosmeticModelNumber: "HS-99",
     DefaultCosmeticName: "Hardshell",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 3600,
     BuildDraw: 750000,
     PassiveDraw: 750000,
     MaxHealth: 3,

     PossibleAmbit: 4,
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

     ActivateCharge: 1,
     BuildCharge: 8,
     DefendChangeCharge: 1,
     MoveCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 9000,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)


    // Struct Type: Planetary Defense Cannon
    structType = StructType{
     Id: 19,
     Type: "Planetary Defense Cannon",
     Class: "Planetary Defense Cannon",
     ClassAbbreviation: "PDC",
     DefaultCosmeticModelNumber: "LK-7",
     DefaultCosmeticName: "Lookout",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 2880,
     BuildDraw: 600000,
     PassiveDraw: 600000,
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

     ActivateCharge: 1,
     BuildCharge: 8,
     DefendChangeCharge: 1,
     MoveCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 0,
     PlanetaryShieldContribution: 4500,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)


    // Struct Type: Field Generator
    structType = StructType{
     Id: 20,
     Type: "Field Generator",
     Class: "Field Generator",
     ClassAbbreviation: "Generator",
     DefaultCosmeticModelNumber: "MG-4",
     DefaultCosmeticName: "Magenta",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 700,
     BuildDraw: 500000,
     PassiveDraw: 500000,
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

     ActivateCharge: 1,
     BuildCharge: 8,
     DefendChangeCharge: 1,
     MoveCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 2,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)


    // Struct Type: Continental Power Plant
    structType = StructType{
     Id: 21,
     Type: "Continental Power Plant",
     Class: "Continental Power Plant",
     ClassAbbreviation: "Power Plant",
     DefaultCosmeticModelNumber: "HS-5",
     DefaultCosmeticName: "Homestead",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 1440,
     BuildDraw: 10000000,
     PassiveDraw: 10000000,
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

     ActivateCharge: 1,
     BuildCharge: 8,
     DefendChangeCharge: 1,
     MoveCharge: 0,
     StealthActivateCharge: 0,

     AttackReduction: 0,
     AttackCounterable: true,
     StealthSystems: false,

     CounterAttack: 0,
     CounterAttackSameAmbit: 0,

     PostDestructionDamage: 0,
     GeneratingRate: 5,
     PlanetaryShieldContribution: 0,

     OreMiningDifficulty: 0,
     OreRefiningDifficulty: 0,

     UnguidedDefensiveSuccessRateNumerator: 0,
     UnguidedDefensiveSuccessRateDenominator: 0,

     GuidedDefensiveSuccessRateNumerator: 0,
     GuidedDefensiveSuccessRateDenominator: 0,

     }
    genesisStructTypes = append(genesisStructTypes, structType)


    // Struct Type: World Engine
    structType = StructType{
     Id: 22,
     Type: "World Engine",
     Class: "World Engine",
     ClassAbbreviation: "World Engine",
     DefaultCosmeticModelNumber: "CS-8",
     DefaultCosmeticName: "Constellation",
     Category: ObjectType_planet,

     BuildLimit: 1,
     BuildDifficulty: 5000,
     BuildDraw: 100000000,
     PassiveDraw: 100000000,
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

     ActivateCharge: 1,
     BuildCharge: 8,
     DefendChangeCharge: 1,
     MoveCharge: 0,
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
  


    return
}




