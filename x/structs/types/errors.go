package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// =============================================================================
// Structured Error Sentinel Codes
// =============================================================================
// Code Range Allocation:
//   1000-1049: Core/System Errors
//   1050-1099: Object Not Found Errors
//   1100-1149: Permission Errors
//   1150-1199: Player State Errors
//   1200-1249: Player Power/Capacity Errors
//   1250-1299: Struct State/Capability Errors
//   1300-1349: Struct Action Errors
//   1350-1399: Struct Build Errors
//   1400-1449: Combat Errors
//   1450-1499: Fleet Errors
//   1500-1549: Guild Errors
//   1550-1599: Allocation Errors
//   1600-1649: Reactor Errors
//   1650-1699: Planet Errors
//   1700-1749: Provider/Agreement Errors
//   1750-1799: Address Errors
//   1800-1849: Work/Hash Errors
//   1900-1999: IBC/Packet Errors
// =============================================================================

// -----------------------------------------------------------------------------
// Core/System Errors (1000-1049)
// -----------------------------------------------------------------------------
var (
	ErrInvalidSigner     = sdkerrors.Register(ModuleName, 1001, "expected gov account as only signer for proposal message")
	ErrInvalidParameters = sdkerrors.Register(ModuleName, 1002, "invalid message parameters")
	ErrSample            = sdkerrors.Register(ModuleName, 1003, "sample error")
)

// -----------------------------------------------------------------------------
// Object Not Found Errors (1050-1099)
// -----------------------------------------------------------------------------
var (
	ErrObjectNotFound     = sdkerrors.Register(ModuleName, 1050, "object not found")
	ErrPlayerNotFound     = sdkerrors.Register(ModuleName, 1051, "player not found")
	ErrStructNotFound     = sdkerrors.Register(ModuleName, 1052, "struct not found")
	ErrStructTypeNotFound = sdkerrors.Register(ModuleName, 1053, "struct type not found")
	ErrGuildNotFound      = sdkerrors.Register(ModuleName, 1054, "guild not found")
	ErrPlanetNotFound     = sdkerrors.Register(ModuleName, 1055, "planet not found")
	ErrFleetNotFound      = sdkerrors.Register(ModuleName, 1056, "fleet not found")
	ErrSubstationNotFound = sdkerrors.Register(ModuleName, 1057, "substation not found")
	ErrAllocationNotFound = sdkerrors.Register(ModuleName, 1058, "allocation not found")
	ErrReactorNotFound    = sdkerrors.Register(ModuleName, 1059, "reactor not found")
	ErrInfusionNotFound   = sdkerrors.Register(ModuleName, 1060, "infusion not found")
	ErrProviderNotFound   = sdkerrors.Register(ModuleName, 1061, "provider not found")
	ErrAgreementNotFound  = sdkerrors.Register(ModuleName, 1062, "agreement not found")
)

// -----------------------------------------------------------------------------
// Permission Errors (1100-1149)
// -----------------------------------------------------------------------------
var (
	ErrPermission                           = sdkerrors.Register(ModuleName, 1100, "permission denied")
	ErrPermissionPlay                       = sdkerrors.Register(ModuleName, 1101, "play permission denied")
	ErrPermissionAssets                     = sdkerrors.Register(ModuleName, 1102, "asset permission denied")
	ErrPermissionEnergy                     = sdkerrors.Register(ModuleName, 1103, "energy permission denied")
	ErrPermissionGuild                      = sdkerrors.Register(ModuleName, 1104, "guild permission denied")
	ErrPermissionSubstation                 = sdkerrors.Register(ModuleName, 1105, "substation permission denied")
	ErrPermissionAllocation                 = sdkerrors.Register(ModuleName, 1106, "allocation permission denied")
	ErrPermissionPlayer                     = sdkerrors.Register(ModuleName, 1107, "player permission denied")
	ErrPermissionAddress                    = sdkerrors.Register(ModuleName, 1108, "address permission denied")
	ErrPermissionAssociation                = sdkerrors.Register(ModuleName, 1109, "address association permission denied")
	ErrPermissionRevoke                     = sdkerrors.Register(ModuleName, 1110, "address revocation permission denied")
	ErrPermissionManageAssets               = sdkerrors.Register(ModuleName, 1111, "asset management permission denied")
	ErrPermissionManagePlayer               = sdkerrors.Register(ModuleName, 1112, "player management permission denied")
	ErrPermissionManageGuild                = sdkerrors.Register(ModuleName, 1113, "guild management permission denied")
	ErrPermissionManageEnergy               = sdkerrors.Register(ModuleName, 1114, "energy management permission denied")
	ErrPermissionGuildRegister              = sdkerrors.Register(ModuleName, 1115, "guild registration permission denied")
	ErrPermissionSubstationDelete           = sdkerrors.Register(ModuleName, 1116, "substation deletion permission denied")
	ErrPermissionSubstationAllocationConnect    = sdkerrors.Register(ModuleName, 1117, "substation allocation connection permission denied")
	ErrPermissionSubstationAllocationDisconnect = sdkerrors.Register(ModuleName, 1118, "substation allocation disconnection permission denied")
	ErrPermissionSubstationPlayerConnect        = sdkerrors.Register(ModuleName, 1119, "substation player connection permission denied")
	ErrPermissionSubstationPlayerDisconnect     = sdkerrors.Register(ModuleName, 1120, "substation player disconnection permission denied")
	ErrPermissionPlayerPlay                     = sdkerrors.Register(ModuleName, 1121, "player play permission denied")
	ErrPermissionPlayerSquad                    = sdkerrors.Register(ModuleName, 1122, "player squad permission denied")
)

// -----------------------------------------------------------------------------
// Player State Errors (1150-1199)
// -----------------------------------------------------------------------------
var (
	ErrPlayerRequired = sdkerrors.Register(ModuleName, 1150, "player account required")
	ErrPlayerHalted   = sdkerrors.Register(ModuleName, 1151, "player is halted")
	ErrPlayerOffline  = sdkerrors.Register(ModuleName, 1152, "player is offline")
	ErrPlayerUpdate   = sdkerrors.Register(ModuleName, 1153, "player update failed")
)

// -----------------------------------------------------------------------------
// Player Power/Capacity Errors (1200-1249)
// -----------------------------------------------------------------------------
var (
	ErrInsufficientCharge    = sdkerrors.Register(ModuleName, 1200, "insufficient charge for action")
	ErrPlayerPowerOffline    = sdkerrors.Register(ModuleName, 1201, "player offline due to power")
	ErrPlayerCapacityExceeded = sdkerrors.Register(ModuleName, 1202, "player capacity exceeded")
	ErrPlayerAffordability   = sdkerrors.Register(ModuleName, 1203, "player cannot afford action")
)

// -----------------------------------------------------------------------------
// Struct State/Capability Errors (1250-1299)
// -----------------------------------------------------------------------------
var (
	ErrStructState           = sdkerrors.Register(ModuleName, 1250, "invalid struct state")
	ErrStructOffline         = sdkerrors.Register(ModuleName, 1251, "struct is offline")
	ErrStructBuilding        = sdkerrors.Register(ModuleName, 1252, "struct is still building")
	ErrStructAlreadyOnline   = sdkerrors.Register(ModuleName, 1253, "struct is already online")
	ErrStructAlreadyOffline  = sdkerrors.Register(ModuleName, 1254, "struct is already offline")
	ErrStructCapability      = sdkerrors.Register(ModuleName, 1260, "struct missing capability")
	ErrStructNoMiningSystem  = sdkerrors.Register(ModuleName, 1261, "struct has no mining system")
	ErrStructNoRefiningSystem = sdkerrors.Register(ModuleName, 1262, "struct has no refining system")
	ErrStructNoStealthSystem = sdkerrors.Register(ModuleName, 1263, "struct has no stealth system")
	ErrStructNoGenerationSystem = sdkerrors.Register(ModuleName, 1264, "struct has no generation system")
	ErrStructInfuse          = sdkerrors.Register(ModuleName, 1270, "struct fuel infusion failed")
)

// -----------------------------------------------------------------------------
// Struct Action Errors (1300-1349)
// -----------------------------------------------------------------------------
var (
	ErrStructAction     = sdkerrors.Register(ModuleName, 1300, "struct action failed")
	ErrStructOwnership  = sdkerrors.Register(ModuleName, 1301, "struct ownership mismatch")
	ErrStructLocation   = sdkerrors.Register(ModuleName, 1302, "invalid struct location")
	ErrStructAmbit      = sdkerrors.Register(ModuleName, 1303, "invalid struct ambit")
	ErrStructActivate   = sdkerrors.Register(ModuleName, 1304, "struct activation failed")
	ErrStructMine       = sdkerrors.Register(ModuleName, 1305, "struct mining failed")
	ErrStructRefine     = sdkerrors.Register(ModuleName, 1306, "struct refining failed")
	ErrStructMineActivate    = sdkerrors.Register(ModuleName, 1307, "struct mining activation failed")
	ErrStructMineDeactivate  = sdkerrors.Register(ModuleName, 1308, "struct mining deactivation failed")
	ErrStructRefineActivate  = sdkerrors.Register(ModuleName, 1309, "struct refining activation failed")
	ErrStructRefineDeactivate = sdkerrors.Register(ModuleName, 1310, "struct refining deactivation failed")
	ErrStructAllocationCreate = sdkerrors.Register(ModuleName, 1311, "struct allocation creation failed")
)

// -----------------------------------------------------------------------------
// Struct Build Errors (1350-1399)
// -----------------------------------------------------------------------------
var (
	ErrStructBuild          = sdkerrors.Register(ModuleName, 1350, "struct build failed")
	ErrStructBuildInitiate  = sdkerrors.Register(ModuleName, 1351, "struct build initiation failed")
	ErrStructBuildComplete  = sdkerrors.Register(ModuleName, 1352, "struct build completion failed")
	ErrStructSlotOccupied   = sdkerrors.Register(ModuleName, 1353, "struct slot already occupied")
	ErrStructSlotUnavailable = sdkerrors.Register(ModuleName, 1354, "struct slot unavailable")
	ErrStructTypeUnsupported = sdkerrors.Register(ModuleName, 1355, "struct type not yet supported")
	ErrStructCommandExists  = sdkerrors.Register(ModuleName, 1356, "command struct already exists")
)

// -----------------------------------------------------------------------------
// Combat Errors (1400-1449)
// -----------------------------------------------------------------------------
var (
	ErrCombatTargeting       = sdkerrors.Register(ModuleName, 1400, "combat targeting failed")
	ErrTargetDestroyed       = sdkerrors.Register(ModuleName, 1401, "target is destroyed")
	ErrTargetUnreachable     = sdkerrors.Register(ModuleName, 1402, "target is unreachable")
	ErrTargetBlocked         = sdkerrors.Register(ModuleName, 1403, "target is blocked")
	ErrTargetHidden          = sdkerrors.Register(ModuleName, 1404, "target is hidden")
	ErrTargetOutOfRange      = sdkerrors.Register(ModuleName, 1405, "target is out of range")
	ErrIncompleteTargeting   = sdkerrors.Register(ModuleName, 1406, "incomplete targeting")
	ErrDefenseRange          = sdkerrors.Register(ModuleName, 1407, "defender out of range")
)

// -----------------------------------------------------------------------------
// Fleet Errors (1450-1499)
// -----------------------------------------------------------------------------
var (
	ErrFleetCommand          = sdkerrors.Register(ModuleName, 1450, "fleet command error")
	ErrFleetNoCommandStruct  = sdkerrors.Register(ModuleName, 1451, "fleet has no command struct")
	ErrFleetCommandOffline   = sdkerrors.Register(ModuleName, 1452, "fleet command struct is offline")
	ErrFleetState            = sdkerrors.Register(ModuleName, 1453, "invalid fleet state")
	ErrFleetNotOnStation     = sdkerrors.Register(ModuleName, 1454, "fleet not on station")
	ErrFleetRaidQueue        = sdkerrors.Register(ModuleName, 1455, "fleet raid queue error")
)

// -----------------------------------------------------------------------------
// Guild Errors (1500-1549)
// -----------------------------------------------------------------------------
var (
	ErrGuildUpdate                = sdkerrors.Register(ModuleName, 1500, "guild update failed")
	ErrGuildMembership            = sdkerrors.Register(ModuleName, 1501, "guild membership error")
	ErrGuildMembershipApplication = sdkerrors.Register(ModuleName, 1502, "guild membership application error")
	ErrGuildJoinType              = sdkerrors.Register(ModuleName, 1503, "invalid guild join type")
	ErrGuildAlreadyMember         = sdkerrors.Register(ModuleName, 1504, "player already guild member")
	ErrGuildNotMember             = sdkerrors.Register(ModuleName, 1505, "player not guild member")
	ErrGuildOwnerKick             = sdkerrors.Register(ModuleName, 1506, "cannot kick guild owner")
	ErrGuildMinimumNotMet         = sdkerrors.Register(ModuleName, 1507, "guild join minimum not met")
	ErrInvalidGuildJoinBypassLevel = sdkerrors.Register(ModuleName, 1508, "invalid guild join bypass level")
)

// -----------------------------------------------------------------------------
// Allocation Errors (1550-1599)
// -----------------------------------------------------------------------------
var (
	ErrAllocationCreate                 = sdkerrors.Register(ModuleName, 1550, "allocation creation failed")
	ErrAllocationUpdate                 = sdkerrors.Register(ModuleName, 1551, "allocation update failed")
	ErrAllocationAppend                 = sdkerrors.Register(ModuleName, 1552, "allocation append failed")
	ErrAllocationSet                    = sdkerrors.Register(ModuleName, 1553, "allocation set failed")
	ErrAllocationCapacity               = sdkerrors.Register(ModuleName, 1554, "allocation capacity exceeded")
	ErrAllocationAutomatedConflict      = sdkerrors.Register(ModuleName, 1555, "automated allocation conflict")
	ErrAllocationImmutableField         = sdkerrors.Register(ModuleName, 1556, "allocation field is immutable")
	ErrAllocationConnectionChangeImpossible = sdkerrors.Register(ModuleName, 1557, "allocation connection change impossible")
	ErrAllocationSourceType             = sdkerrors.Register(ModuleName, 1558, "invalid allocation source type")
	ErrAllocationSourceTypeMismatch     = sdkerrors.Register(ModuleName, 1559, "allocation source type mismatch")
	ErrAllocationSourceNotOnline        = sdkerrors.Register(ModuleName, 1560, "allocation source not online")
	ErrSubstationHasNoPowerSource       = sdkerrors.Register(ModuleName, 1561, "substation has no power source")
)

// -----------------------------------------------------------------------------
// Reactor Errors (1600-1649)
// -----------------------------------------------------------------------------
var (
	ErrReactor              = sdkerrors.Register(ModuleName, 1600, "reactor error")
	ErrReactorRequired      = sdkerrors.Register(ModuleName, 1601, "reactor required")
	ErrReactorActivation    = sdkerrors.Register(ModuleName, 1602, "reactor activation failed")
	ErrReactorInfusion      = sdkerrors.Register(ModuleName, 1603, "reactor infusion failed")
	ErrReactorDefusion      = sdkerrors.Register(ModuleName, 1604, "reactor defusion failed")
	ErrReactorBeginMigration = sdkerrors.Register(ModuleName, 1605, "reactor migration failed")
	ErrReactorCancelDefusion = sdkerrors.Register(ModuleName, 1606, "reactor cancel defusion failed")
	ErrReactorInvalidAddress = sdkerrors.Register(ModuleName, 1607, "invalid reactor address")
	ErrReactorInvalidAmount = sdkerrors.Register(ModuleName, 1608, "invalid reactor amount")
	ErrReactorInvalidDenom  = sdkerrors.Register(ModuleName, 1609, "invalid reactor denomination")
	ErrReactorInvalidHeight = sdkerrors.Register(ModuleName, 1610, "invalid reactor height")
	ErrReactorBalanceExceeded = sdkerrors.Register(ModuleName, 1611, "reactor balance exceeded")
	ErrReactorAlreadyProcessed = sdkerrors.Register(ModuleName, 1612, "reactor unbonding already processed")
)

// -----------------------------------------------------------------------------
// Planet Errors (1650-1699)
// -----------------------------------------------------------------------------
var (
	ErrPlanetState        = sdkerrors.Register(ModuleName, 1650, "invalid planet state")
	ErrPlanetComplete     = sdkerrors.Register(ModuleName, 1651, "planet already complete")
	ErrPlanetEmpty        = sdkerrors.Register(ModuleName, 1652, "planet is empty")
	ErrPlanetHasOre       = sdkerrors.Register(ModuleName, 1653, "planet still has ore")
	ErrPlanetExploration  = sdkerrors.Register(ModuleName, 1654, "planet exploration failed")
)

// -----------------------------------------------------------------------------
// Provider/Agreement Errors (1700-1749)
// -----------------------------------------------------------------------------
var (
	ErrProviderAccess        = sdkerrors.Register(ModuleName, 1700, "provider access denied")
	ErrProviderMarketClosed  = sdkerrors.Register(ModuleName, 1701, "provider market closed")
	ErrProviderGuildNotApproved = sdkerrors.Register(ModuleName, 1702, "guild not approved for provider")
	ErrParameterValidation   = sdkerrors.Register(ModuleName, 1710, "parameter validation failed")
	ErrParameterBelowMinimum = sdkerrors.Register(ModuleName, 1711, "parameter below minimum")
	ErrParameterAboveMaximum = sdkerrors.Register(ModuleName, 1712, "parameter above maximum")
	ErrParameterExceedsCapacity = sdkerrors.Register(ModuleName, 1713, "parameter exceeds available capacity")
)

// -----------------------------------------------------------------------------
// Address Errors (1750-1799)
// -----------------------------------------------------------------------------
var (
	ErrAddressValidation    = sdkerrors.Register(ModuleName, 1750, "address validation failed")
	ErrAddressInvalidFormat = sdkerrors.Register(ModuleName, 1751, "invalid address format")
	ErrAddressNotRegistered = sdkerrors.Register(ModuleName, 1752, "address not registered")
	ErrAddressWrongPlayer   = sdkerrors.Register(ModuleName, 1753, "address belongs to wrong player")
	ErrAddressProofMismatch = sdkerrors.Register(ModuleName, 1754, "address proof mismatch")
	ErrAddressSignatureInvalid = sdkerrors.Register(ModuleName, 1755, "address signature invalid")
	ErrAddressAlreadyRegistered = sdkerrors.Register(ModuleName, 1756, "address already registered")
)

// -----------------------------------------------------------------------------
// Work/Hash Errors (1800-1849)
// -----------------------------------------------------------------------------
var (
	ErrWorkFailure      = sdkerrors.Register(ModuleName, 1800, "work verification failed")
	ErrWorkMineFailure  = sdkerrors.Register(ModuleName, 1801, "mining work failed")
	ErrWorkRefineFailure = sdkerrors.Register(ModuleName, 1802, "refining work failed")
	ErrWorkBuildFailure = sdkerrors.Register(ModuleName, 1803, "build work failed")
	ErrWorkRaidFailure  = sdkerrors.Register(ModuleName, 1804, "raid work failed")
)

// -----------------------------------------------------------------------------
// Sabotage Errors (1850-1899)
// -----------------------------------------------------------------------------
var (
	ErrSabotage = sdkerrors.Register(ModuleName, 1850, "sabotage failed")
)

// -----------------------------------------------------------------------------
// IBC/Packet Errors (1900-1999)
// -----------------------------------------------------------------------------
var (
	ErrInvalidPacketTimeout = sdkerrors.Register(ModuleName, 1900, "invalid packet timeout")
	ErrInvalidVersion       = sdkerrors.Register(ModuleName, 1901, "invalid version")
)

