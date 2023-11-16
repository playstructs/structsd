package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/structs module sentinel errors
var (
	ErrSample               = sdkerrors.Register(ModuleName, 1100, "sample error")
	ErrInvalidPacketTimeout = sdkerrors.Register(ModuleName, 1500, "invalid packet timeout")
	ErrInvalidVersion       = sdkerrors.Register(ModuleName, 1501, "invalid version")
)

var (
	ErrAllocationNotFound                   = sdkerrors.Register(ModuleName, 1510, "allocation not found")
	ErrAllocationSourceType                 = sdkerrors.Register(ModuleName, 1511, "invalid source type")
	ErrAllocationSourceTypeMismatch         = sdkerrors.Register(ModuleName, 1512, "source type mismatch")
	ErrAllocationSourceNotFound             = sdkerrors.Register(ModuleName, 1513, "source not found")
	ErrAllocationSourceNotOnline            = sdkerrors.Register(ModuleName, 1514, "source not online")
	ErrAllocationConnectionChangeImpossible = sdkerrors.Register(ModuleName, 1515, "allocation connection change attempted is impossible")

    ErrPlayerRequired                       = sdkerrors.Register(ModuleName, 1530, "player account required for this action")
    ErrPlayerNotFound                       = sdkerrors.Register(ModuleName, 1531, "player account specified does not exist")
    ErrPlayerUpdate                         = sdkerrors.Register(ModuleName, 1532, "player account update failed")

	ErrSubstationNotFound         = sdkerrors.Register(ModuleName, 1550, "substation not found")
	ErrSubstationHasNoPowerSource = sdkerrors.Register(ModuleName, 1551, "substation has no power source")

	ErrSubstationAvailableCapacityInsufficient = sdkerrors.Register(ModuleName, 1552, "substation capacity lower then attempted change allows for")
	ErrSubstationOffline                       = sdkerrors.Register(ModuleName, 1553, "substation currently unable to support connected players")

    ErrReactorActivation = sdkerrors.Register(ModuleName, 1571, "reactor activation failure")
	ErrReactorAvailableCapacityInsufficient = sdkerrors.Register(ModuleName, 1572, "reactor capacity lower then attempted change allows for")
	ErrReactorRequired                      = sdkerrors.Register(ModuleName, 1573, "reactor account required for this action")

    ErrGuildNotFound                        = sdkerrors.Register(ModuleName,  1580, "guild specified does not exist")
    ErrGuildUpdate                          = sdkerrors.Register(ModuleName,  1581, "guild could not be updated")

	ErrPermissionAssociation                = sdkerrors.Register(ModuleName, 1600, "Permission error during address association")
	ErrPermissionRevoke                     = sdkerrors.Register(ModuleName, 1601, "Permission error during address revocation")
	ErrPermissionPlay                       = sdkerrors.Register(ModuleName, 1602, "Permission error during play action")
	ErrPermissionManageAssets               = sdkerrors.Register(ModuleName, 1603, "Permission error during asset management action")
	ErrPermissionManagePlayer               = sdkerrors.Register(ModuleName, 1604, "Permission error during player management action")
    ErrPermissionManageGuild                = sdkerrors.Register(ModuleName, 1605, "Permission error during guild management action")
    ErrPermissionManageEnergy               = sdkerrors.Register(ModuleName, 1606, "Permission error during asset management action")
    ErrPermissionManageSquad                = sdkerrors.Register(ModuleName, 1607, "Permission error during squad management action")

    ErrPermissionGuildRegister              = sdkerrors.Register(ModuleName, 1611, "Guild permission error during player register")

    ErrPermissionReactorAllocationCreate    = sdkerrors.Register(ModuleName, 1621, "Reactor permission error during allocation creation")

    ErrPermissionAllocation                 = sdkerrors.Register(ModuleName, 1630, "Allocation not owned by calling player")

    ErrPermissionSubstationAllocationCreate     = sdkerrors.Register(ModuleName, 1631, "Substation permission error during allocation creation")
    ErrPermissionSubstationDelete               = sdkerrors.Register(ModuleName, 1632, "Substation permission error during allocation creation")
    ErrPermissionSubstationAllocationConnect    = sdkerrors.Register(ModuleName, 1633, "Substation permission error during allocation connection")
    ErrPermissionSubstationAllocationDisconnect = sdkerrors.Register(ModuleName, 1634, "Substation permission error during allocation disconnection")
    ErrPermissionSubstationPlayerConnect        = sdkerrors.Register(ModuleName, 1635, "Substation permission error during player connection")
    ErrPermissionSubstationPlayerDisconnect     = sdkerrors.Register(ModuleName, 1636, "Substation permission error during player disconnection")

    ErrPermissionPlayerPlay                     = sdkerrors.Register(ModuleName, 1641, "Player cannot play other players yet (no sudo yo)")
    ErrPermissionPlayerSquad                    = sdkerrors.Register(ModuleName, 1642, "Player cannot update other players squad status")

    ErrPlanetNotFound                           = sdkerrors.Register(ModuleName,  1710, "planet specified does not exist")
    ErrPlanetExploration                        = sdkerrors.Register(ModuleName,  1711, "planet exploration failed")

    ErrStructNotFound                           = sdkerrors.Register(ModuleName,  1720, "struct specified does not exist")
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

    ErrSquadNotFound                            = sdkerrors.Register(ModuleName,  1810, "Squad not found")

    ErrSquadLeaderProposalNotFound              = sdkerrors.Register(ModuleName,  1820, "Squad leader proposal not found")
    ErrSquadLeaderProposalGuildMismatch         = sdkerrors.Register(ModuleName,  1821, "Attempted to add squad leader from outside guild")
    ErrSquadLeaderProposalPlayerMismatch        = sdkerrors.Register(ModuleName,  1822, "Attempting to accept a squad leader position for another player")
    ErrSquadLeaderProposalPlayerIneligible      = sdkerrors.Register(ModuleName,  1823, "Attempted to add a squad leader who cannot be squad leader")

    ErrPermissionSquadCreation                  = sdkerrors.Register(ModuleName,  1830, "Squad creation failed")


    ErrSabotage                                 = sdkerrors.Register(ModuleName,  3800, "Sabotage failed")

)
