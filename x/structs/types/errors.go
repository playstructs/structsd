package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/structs module sentinel errors
var (
	ErrInvalidSigner        = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSample               = sdkerrors.Register(ModuleName, 1101, "sample error")
	ErrInvalidPacketTimeout = sdkerrors.Register(ModuleName, 1500, "invalid packet timeout")
	ErrInvalidVersion       = sdkerrors.Register(ModuleName, 1501, "invalid version")
)


var (
    ErrGridMalfunction                      = sdkerrors.Register(ModuleName, 1800, "Grid malfunction")
    ErrInsufficientCharge                   = sdkerrors.Register(ModuleName, 1801, "Insufficient Charge for Action")
    ErrPlayerHalted                         = sdkerrors.Register(ModuleName, 1802, "Player is currently Halted, must be Resumed before more actions")

    ErrObjectNotFound                       = sdkerrors.Register(ModuleName, 1900, "object not found")


	ErrAllocationSourceType                 = sdkerrors.Register(ModuleName, 1511, "invalid source type")
	ErrAllocationSourceTypeMismatch         = sdkerrors.Register(ModuleName, 1512, "source type mismatch")
	ErrAllocationSourceNotOnline            = sdkerrors.Register(ModuleName, 1514, "source not online")
	ErrAllocationConnectionChangeImpossible = sdkerrors.Register(ModuleName, 1515, "allocation connection change attempted is impossible")
	ErrAllocationSet                        = sdkerrors.Register(ModuleName, 1516, "allocation could not be updated")
	ErrAllocationAppend                     = sdkerrors.Register(ModuleName, 1517, "allocation could not be created")

    ErrPlayerRequired                       = sdkerrors.Register(ModuleName, 1530, "player account required for this action")
    ErrPlayerUpdate                         = sdkerrors.Register(ModuleName, 1532, "player account update failed")

	ErrSubstationHasNoPowerSource           = sdkerrors.Register(ModuleName, 1551, "substation has no power source")

	ErrReactorActivation                    = sdkerrors.Register(ModuleName, 1571, "reactor activation failure")
	ErrReactorRequired                      = sdkerrors.Register(ModuleName, 1573, "reactor account required for this action")

    ErrGuildUpdate                          = sdkerrors.Register(ModuleName,  1581, "guild could not be updated")
    ErrInvalidGuildJoinBypassLevel          = sdkerrors.Register(ModuleName,  1582, "invalid guild join bypass level")
    ErrGuildMembershipApplication           = sdkerrors.Register(ModuleName,  1583, "invalid application process")


    ErrPermission                           = sdkerrors.Register(ModuleName, 1607, "Permission error")
	ErrPermissionAssociation                = sdkerrors.Register(ModuleName, 1600, "Permission error during address association")
	ErrPermissionRevoke                     = sdkerrors.Register(ModuleName, 1601, "Permission error during address revocation")
	ErrPermissionPlay                       = sdkerrors.Register(ModuleName, 1602, "Permission error during play action")
	ErrPermissionManageAssets               = sdkerrors.Register(ModuleName, 1603, "Permission error during asset management action")
	ErrPermissionManagePlayer               = sdkerrors.Register(ModuleName, 1604, "Permission error during player management action")
    ErrPermissionManageGuild                = sdkerrors.Register(ModuleName, 1605, "Permission error during guild management action")
    ErrPermissionManageEnergy               = sdkerrors.Register(ModuleName, 1606, "Permission error during asset management action")

    ErrPermissionGuildRegister              = sdkerrors.Register(ModuleName, 1611, "Guild permission error during player register")

    ErrPermissionAllocation                 = sdkerrors.Register(ModuleName, 1630, "Allocation not owned by calling player")

    ErrPermissionSubstationDelete               = sdkerrors.Register(ModuleName, 1632, "Substation permission error during allocation creation")
    ErrPermissionSubstationAllocationConnect    = sdkerrors.Register(ModuleName, 1633, "Substation permission error during allocation connection")
    ErrPermissionSubstationAllocationDisconnect = sdkerrors.Register(ModuleName, 1634, "Substation permission error during allocation disconnection")
    ErrPermissionSubstationPlayerConnect        = sdkerrors.Register(ModuleName, 1635, "Substation permission error during player connection")
    ErrPermissionSubstationPlayerDisconnect     = sdkerrors.Register(ModuleName, 1636, "Substation permission error during player disconnection")

    ErrPermissionPlayerPlay                     = sdkerrors.Register(ModuleName, 1641, "Player cannot play other players yet (no sudo yo)")
    ErrPermissionPlayerSquad                    = sdkerrors.Register(ModuleName, 1642, "Player cannot update other players squad status")

    ErrPlanetExploration                        = sdkerrors.Register(ModuleName,  1711, "planet exploration failed")


    ErrStructBuildInitiate                      = sdkerrors.Register(ModuleName,  1721, "Struct build initialization failed")
    ErrStructBuildComplete                      = sdkerrors.Register(ModuleName,  1722, "Struct build completion failed")
    ErrStructMineActivate                       = sdkerrors.Register(ModuleName,  1723, "Struct mining system activation failed")
    ErrStructMineDeactivate                     = sdkerrors.Register(ModuleName,  1724, "Struct mining system deactivation failed")
    ErrStructMine                               = sdkerrors.Register(ModuleName,  1725, "Struct mining action failed")
    ErrStructRefineActivate                     = sdkerrors.Register(ModuleName,  1726, "Struct refining system activation failed")
    ErrStructRefineDeactivate                   = sdkerrors.Register(ModuleName,  1727, "Struct refining system deactivation failed")
    ErrStructRefine                             = sdkerrors.Register(ModuleName,  1728, "Struct refining action failed")
    ErrStructInfuse                             = sdkerrors.Register(ModuleName,  1729, "Struct infusion action failed")
    ErrStructAllocationCreate                   = sdkerrors.Register(ModuleName,  1730, "Allocation of power from struct failed")
    ErrStructActivate                           = sdkerrors.Register(ModuleName,  1731, "Struct activation failed")

    ErrStructAction                             = sdkerrors.Register(ModuleName,  1700, "Struct action failed")

    ErrInvalidParameters                        = sdkerrors.Register(ModuleName,  1800, "Invalid Message Details")

    ErrSabotage                                 = sdkerrors.Register(ModuleName,  3800, "Sabotage failed")

)
