package types

const (
	// ModuleName defines the module name
	ModuleName = "structs"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// StoreKey defines the transient module store key
	// Data stored only during block processing
	TStoreKey = "transient_structs"

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_structs"

	// Version defines the current version the IBC module supports
	Version = "structs-1"

	// PortID is the default port id that module binds to
	PortID = "structs"

	// Starting value for Keeper IDs
    KeeperStartValue = 1

    // Starting value for Reactor Owner Initialization
    InitialReactorOwnerEnergy = 100

    // Starting value for Reactor Owner Initialization
    InitialSubstationOwnerEnergy = 100


    DifficultyBuildAgeRange  = 36000 // 10 // 1800 // 36000 // 1 days
    DifficultyActionAgeRange = 252000 // 20 // 3600 // 252000 // 7 days

    DifficultySabotageRangeMine   = DifficultyActionAgeRange  // 36000 // 1 days
    DifficultySabotageRangeRefine = DifficultyActionAgeRange // 252000 // 7 days
    DifficultySabotageRangePower  = 252000 // 252000 // 7 days


    // Current Aim is a 3 hour max
    Charge_Volts = 100000000
    Charge_Resistance = 100.0
    Charge_Capacitance = 10.0
)

var (
	ParamsKey = []byte("p_structs")
)

var (
	// PortKey defines the key to store the port ID in store
	PortKey = KeyPrefix("structs-port-")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}


const (
    PermissionKey = "Permission/value/"
)

const (
    GridAttributeKey = "Grid/attribute/"
    GridCascadeQueue = "Grid/cascadeQueue/"
)

const (
	ReactorKey          = "Reactor/value/"
	ReactorCountKey     = "Reactor/count/"
	ReactorValidatorKey = "Reactor/validator/"

	ReactorFuelToEnergyConversion = 100
)

const (
	SubstationKey       = "Substation/value/"
	SubstationCountKey  = "Substation/count/"
	SubstationStatusKey = "Substation/status/"
)

const (
	AllocationKey      = "Allocation/value/"
	AllocationAutoResizeKey = "Allocation/autoResize/"
)

const (
	InfusionKey      = "Infusion/value/"
	InfusionCountKey = "Infusion/count/"
)

const (
	FleetKey      = "Fleet/value/"
)

const (
	GuildKey      = "Guild/value/"
	GuildCountKey = "Guild/count/"
	GuildMembershipApplicationKey = "Guild/membershipApplication/"
)


const (
	PlayerKey      = "Player/value/"
	PlayerCountKey = "Player/count/"

	PlayerPassiveDraw = 25
)

const (
	AddressPlayerKey = "Address/player/"
)

const (
	PlanetKey                = "Planet/value/"
	PlanetCountKey           = "Planet/count/"
	PlanetAttributeKey       = "Planet/attribute/"

	// TODO Make these dynamic in the future
	PlanetStartingOre = 5
	PlanetStartingSlots = 4
)

const (
	StructKey      = "Struct/value/"
	StructCountKey  = "Struct/count/"
	StructDefenderKey  = "Struct/defender/"
    StructAttributeKey  = "Struct/attribute/"

	StructFuelToEnergyConversion = 200
)

const (
	StructTypeKey       = "StructType/value/"
	StructTypeCountKey  = "StructType/count/"
)



/*
 * Additional code needed for ObjectType enumeration that the proto
 * file doesn't seem to generate in keys.pb.go
 *
 * So this seems like as good a place as any for it.
 */
var ObjectType_enum = map[string]ObjectType{
	"guild":      ObjectType_guild,
	"player":     ObjectType_player,
	"planet":     ObjectType_planet,
	"reactor":    ObjectType_reactor,
	"substation": ObjectType_substation,
	"struct":     ObjectType_struct,
	"allocation": ObjectType_allocation,
	"infusion":   ObjectType_infusion,
	"address":    ObjectType_address,
    "fleet":      ObjectType_fleet,
}

// Doing the same for GridAttributeType
var GridAttributeType_enum = map[string]GridAttributeType{
    "ore":                      GridAttributeType_ore,
	"fuel":                     GridAttributeType_fuel,
	"capacity":                 GridAttributeType_capacity,
	"load":                     GridAttributeType_load,
	"structsLoad":              GridAttributeType_structsLoad,
	"power":                    GridAttributeType_power,
	"connectionCapacity":       GridAttributeType_connectionCapacity,
	"connectionCount":          GridAttributeType_connectionCount,
	"allocationPointerStart":   GridAttributeType_allocationPointerStart,
	"allocationPointerEnd":     GridAttributeType_allocationPointerEnd,
	"proxyNonce":               GridAttributeType_proxyNonce,
	"lastAction":               GridAttributeType_lastAction,
	"nonce":                    GridAttributeType_nonce,
}


// Doing the same for AllocationType
var AllocationType_enum = map[string]AllocationType{
	"static":       AllocationType_static,
	"dynamic":      AllocationType_dynamic,
	"automated":    AllocationType_automated,

}

// Going to stop repeating the same "doin the same" comment,
// but everything below is "doin the same"

var GuildJoinBypassLevel_enum = map[string]GuildJoinBypassLevel {
	"closed":        GuildJoinBypassLevel_closed,
	"permissioned":  GuildJoinBypassLevel_permissioned,
	"member":        GuildJoinBypassLevel_member,
}


var RegistrationStatus_enum = map[string]RegistrationStatus {
	"proposed":  RegistrationStatus_proposed,
	"approved":  RegistrationStatus_approved,
	"denied":    RegistrationStatus_denied,
    "revoked":   RegistrationStatus_revoked,
}



var Ambit_enum = map[string]Ambit {
    "none":  Ambit_none,
    "water": Ambit_water,
    "land":  Ambit_land,
    "air":   Ambit_air,
    "space": Ambit_space,
    "local": Ambit_local,
}

var Ambit_flag = map[Ambit]uint64 {
    Ambit_none:  uint64(1) << (Ambit_none),
    Ambit_water: uint64(1) << (Ambit_water),
    Ambit_land:  uint64(1) << (Ambit_land),
    Ambit_air:   uint64(1) << (Ambit_air),
    Ambit_space: uint64(1) << (Ambit_space),
    Ambit_local: uint64(1) << (Ambit_local),
}


var PlanetStatus_enum = map[string]PlanetStatus {
    "active":       PlanetStatus_active,
    "complete":     PlanetStatus_complete,
}


var StructAttributeType_enum = map[string]StructAttributeType {
    "health":               StructAttributeType_health,
    "status":               StructAttributeType_status,

    "blockStartBuild":      StructAttributeType_blockStartBuild,
    "blockStartOreMine":    StructAttributeType_blockStartOreMine,
    "blockStartOreRefine":  StructAttributeType_blockStartOreRefine,

    "protectedStructIndex": StructAttributeType_protectedStructIndex,
}

var PlanetAttributeType_enum = map[string]PlanetAttributeType {
    "planetaryShield":                                          PlanetAttributeType_planetaryShield,
    "repairNetworkQuantity":                                    PlanetAttributeType_repairNetworkQuantity,
    "defensiveCannonQuantity":                                  PlanetAttributeType_defensiveCannonQuantity,
    "coordinatedGlobalShieldNetworkQuantity":                   PlanetAttributeType_coordinatedGlobalShieldNetworkQuantity,

    "lowOrbitBallisticsInterceptorNetworkQuantity":             PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkQuantity,
    "advancedLowOrbitBallisticsInterceptorNetworkQuantity":     PlanetAttributeType_advancedLowOrbitBallisticsInterceptorNetworkQuantity,

    "orbitalJammingStationQuantity":                            PlanetAttributeType_orbitalJammingStationQuantity,
    "advancedOrbitalJammingStationQuantity":                    PlanetAttributeType_advancedOrbitalJammingStationQuantity,

}

var TechWeaponSystem_enum = map[string]TechWeaponSystem {
    "primaryWeapon":    TechWeaponSystem_primaryWeapon,
    "secondaryWeapon":  TechWeaponSystem_secondaryWeapon,
}

var TechWeaponControl_enum = map[string]TechWeaponControl {
    "noWeaponControl":  TechWeaponControl_noWeaponControl,
    "guided":           TechWeaponControl_guided,
    "unguided":         TechWeaponControl_unguided,
}

var TechActiveWeaponry_enum = map[string]TechActiveWeaponry {
    "noActiveWeaponry": TechActiveWeaponry_noActiveWeaponry,
    "guidedWeaponry":   TechActiveWeaponry_guidedWeaponry,
    "unguidedWeaponry": TechActiveWeaponry_unguidedWeaponry,
    "attackRun":        TechActiveWeaponry_attackRun,
    "selfDestruct":     TechActiveWeaponry_selfDestruct,
}

var TechPassiveWeaponry_enum = map[string]TechPassiveWeaponry {
    "noPassiveWeaponry":        TechPassiveWeaponry_noPassiveWeaponry,
    "counterAttack":            TechPassiveWeaponry_counterAttack,
    "strongCounterAttack":      TechPassiveWeaponry_strongCounterAttack,
    "advancedCounterAttack":    TechPassiveWeaponry_advancedCounterAttack,
    "lastResort":               TechPassiveWeaponry_lastResort,
}

var TechUnitDefenses_enum = map[string]TechUnitDefenses {
    "noUnitDefenses":       TechUnitDefenses_noUnitDefenses,
    "defensiveManeuver":    TechUnitDefenses_defensiveManeuver,
    "signalJamming":        TechUnitDefenses_signalJamming,
    "armour":               TechUnitDefenses_armour,
    "indirectCombatModule": TechUnitDefenses_indirectCombatModule,
    "stealthMode":          TechUnitDefenses_stealthMode,
    "perimeterFencing":     TechUnitDefenses_perimeterFencing,
    "reinforcedWalls":      TechUnitDefenses_reinforcedWalls,
}


var TechOreReserveDefenses_enum = map[string]TechOreReserveDefenses {
    "noOreReserveDefenses":             TechOreReserveDefenses_noOreReserveDefenses,
    "coordinatedReserveResponseTracker": TechOreReserveDefenses_coordinatedReserveResponseTracker ,
    "rapidResponsePackage":             TechOreReserveDefenses_rapidResponsePackage,
    "activeScanning":                   TechOreReserveDefenses_activeScanning,
    "monitoringStation":                TechOreReserveDefenses_monitoringStation,
    "oreBunker":                        TechOreReserveDefenses_oreBunker,
}


var TechPlanetaryDefenses_enum = map[string]TechPlanetaryDefenses {
    "noPlanetaryDefense":                           TechPlanetaryDefenses_noPlanetaryDefense,
    "defensiveCannon":                              TechPlanetaryDefenses_defensiveCannon,
    "lowOrbitBallisticInterceptorNetwork":          TechPlanetaryDefenses_lowOrbitBallisticInterceptorNetwork,
    //"advancedLowOrbitBallisticInterceptorNetwork":  TechPlanetaryDefenses_advancedLowOrbitBallisticInterceptorNetwork,
    //"repairNetwork":                                TechPlanetaryDefenses_repairNetwork,
    //"coordinatedGlobalShieldNetwork":               TechPlanetaryDefenses_coordinatedGlobalShieldNetwork,
    //"orbitalJammingStation":                        TechPlanetaryDefenses_orbitalJammingStation,
    //"advancedOrbitalJammingStation":                TechPlanetaryDefenses_advancedOrbitalJammingStation,
}



var TechStorageFacilities_enum = map[string]TechStorageFacilities {
    "noStorageFacilities":  TechStorageFacilities_noStorageFacilities ,
    "dock":                 TechStorageFacilities_dock ,
    "hanger":               TechStorageFacilities_hanger ,
    "fleetBase":            TechStorageFacilities_fleetBase ,
}



var TechPlanetaryMining_enum = map[string]TechPlanetaryMining {
    "noPlanetaryMining":    TechPlanetaryMining_noPlanetaryMining ,
    "oreMiningRig":         TechPlanetaryMining_oreMiningRig ,
}


var TechPlanetaryRefineries_enum = map[string]TechPlanetaryRefineries {
    "noPlanetaryRefinery":  TechPlanetaryRefineries_noPlanetaryRefinery,
    "oreRefinery":          TechPlanetaryRefineries_oreRefinery,
}

var TechPowerGeneration_enum = map[string]TechPowerGeneration {
    "noPowerGeneration":    TechPowerGeneration_noPowerGeneration,
    "smallGenerator":       TechPowerGeneration_smallGenerator,
    "mediumGenerator":      TechPowerGeneration_mediumGenerator,
    "largeGenerator":       TechPowerGeneration_largeGenerator,
}

