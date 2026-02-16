package keeper

import (


    //"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

)

type PlayerCache struct {
    PlayerId string
    CC  *CurrentContext

    Ready bool

    Changed bool
    Deleted bool

    PlayerLoaded    bool
    Player          types.Player

    ActiveAddress string

    StorageLoaded bool
    Storage       sdk.Coins

    NonceAttributeId                string
    LastActionAttributeId           string
    LoadAttributeId                 string
    CapacityAttributeId             string
    StructsLoadAttributeId          string
    CapacitySecondaryAttributeId    string
    StoredOreAttributeId            string

}

func (cache *PlayerCache) Commit() () {
    if cache.Changed {
        cache.CC.k.logger.Info("Updating Player From Cache","playerId", cache.PlayerId)
        cache.CC.k.SetPlayer(cache.CC.ctx, cache.Player)
    }
    cache.Changed = false
}

func (cache *PlayerCache) IsChanged() bool {
    return cache.Changed
}

func (cache *PlayerCache) ID() string {
    return cache.PlayerId
}


func (cache *PlayerCache) LoadPlayer() (found bool) {
    cache.Player, cache.PlayerLoaded = cache.CC.k.GetPlayer(cache.CC.ctx, cache.PlayerId)

    return cache.PlayerLoaded
}



func (cache *PlayerCache) LoadStorage() (error){
    if (!cache.PlayerLoaded) {
        return nil // TODO update to be an error
    }
    playerAcc, _ := sdk.AccAddressFromBech32(cache.Player.PrimaryAddress)
    cache.Storage = cache.CC.k.bankKeeper.SpendableCoins(cache.CC.ctx, playerAcc)

    return nil
}

func (cache *PlayerCache) CheckPlayer() (error) {
    if (!cache.PlayerLoaded) {
        if !cache.LoadPlayer() {
           return types.NewObjectNotFoundError("player", cache.PlayerId)
        }
    }
    return nil
}


func (cache *PlayerCache) GetPlayer() (types.Player) {
    if (!cache.PlayerLoaded) {
        cache.LoadPlayer()
    }
    return cache.Player
}


func (cache *PlayerCache) GetPlayerId()         (string) { return cache.PlayerId }
func (cache *PlayerCache) GetPrimaryAddress()   (string) { if (!cache.PlayerLoaded) { cache.LoadPlayer() }; return cache.Player.PrimaryAddress }
func (cache *PlayerCache) GetPrimaryAccount()   (sdk.AccAddress) { acc, _ := sdk.AccAddressFromBech32(cache.GetPrimaryAddress()); return acc }
func (cache *PlayerCache) GetActiveAddress()    (string) { return cache.ActiveAddress }
func (cache *PlayerCache) GetIndex()            (uint64) { if (!cache.PlayerLoaded) { cache.LoadPlayer() }; return cache.Player.Index }

func (cache *PlayerCache) GetFleetId()      (string) { if (!cache.PlayerLoaded) { cache.LoadPlayer() }; return cache.Player.FleetId }
func (cache *PlayerCache) GetGuildId()      (string) { if (!cache.PlayerLoaded) { cache.LoadPlayer() }; return cache.Player.GuildId }
func (cache *PlayerCache) GetPlanetId()     (string) { if (!cache.PlayerLoaded) { cache.LoadPlayer() }; return cache.Player.PlanetId }
func (cache *PlayerCache) GetSubstationId() (string) { if (!cache.PlayerLoaded) { cache.LoadPlayer() }; return cache.Player.SubstationId }

func (cache *PlayerCache) GetFleet()        (*FleetCache)       {
    fleetId := cache.GetFleetId()
    if fleetId == "" {
        // Player doesn't have a fleet yet; use the player index to
        // create/load one through GetFleet which properly sets CC.
        fleet, _ := cache.CC.GetFleet(cache.GetIndex())
        return fleet
    }
    fleet, _ := cache.CC.GetFleetById(fleetId)
    return fleet
}

func (cache *PlayerCache) GetGuild()        (*GuildCache)       { return cache.CC.GetGuild( cache.GetGuildId() ) }

func (cache *PlayerCache) GetPlanet()       (*PlanetCache)      { return cache.CC.GetPlanet( cache.GetPlanetId() ) }

func (cache *PlayerCache) GetSubstation()   (*SubstationCache)  {
    return cache.CC.GetSubstation( cache.GetSubstationId() )
}


func (cache *PlayerCache) GetNonce() (int64) {
    return int64(cache.CC.GetGridAttribute(cache.NonceAttributeId))
}
func (cache *PlayerCache) GetNextNonce() (int64) {
    return int64(cache.CC.SetGridAttributeIncrement(cache.NonceAttributeId, 1))
}

func (cache *PlayerCache) GetLastAction() (uint64) {
    return cache.CC.GetGridAttribute(cache.LastActionAttributeId)
}

func (cache *PlayerCache) GetLoad() (uint64) {
    return cache.CC.GetGridAttribute(cache.LoadAttributeId)
}

func (cache *PlayerCache) GetCapacity() (uint64) {
    return cache.CC.GetGridAttribute(cache.CapacityAttributeId)
}

func (cache *PlayerCache) GetCapacitySecondary() (uint64) {
    cache.CapacitySecondaryAttributeId = GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, cache.GetSubstationId())
    return cache.CC.GetGridAttribute(cache.CapacitySecondaryAttributeId)
}

func (cache *PlayerCache) GetStructsLoad() (uint64) {
    return cache.CC.GetGridAttribute(cache.StructsLoadAttributeId)
}

func (cache *PlayerCache) GetStoredOre() (uint64) {
    return cache.CC.GetGridAttribute(cache.StoredOreAttributeId)
}


func (cache *PlayerCache) GetCharge() (uint64) {
    ctxSDK := sdk.UnwrapSDKContext(cache.CC.ctx);
    return uint64(ctxSDK.BlockHeight()) - cache.GetLastAction()
}

func (cache *PlayerCache) GetAllocatableCapacity() (uint64) {
    return cache.GetCapacity() - cache.GetLoad()
}

func (cache *PlayerCache) GetAvailableCapacity() (uint64) {
    if (cache.GetLoad() + cache.GetStructsLoad()) > (cache.GetCapacity() + cache.GetCapacitySecondary()) {
        return 0
    } else {
        return (cache.GetCapacity() + cache.GetCapacitySecondary()) - (cache.GetLoad() + cache.GetStructsLoad())
    }
}

func (cache *PlayerCache) GetBuiltQuantity(structTypeId uint64) (uint64) {
    typeCountAttributeId := GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, cache.GetPlayerId(), structTypeId)
    return cache.CC.GetStructAttribute(typeCountAttributeId)
}

func (cache *PlayerCache) BuildQuantityIncrement(structTypeId uint64) {
    typeCountAttributeId := GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, cache.GetPlayerId(), structTypeId)
    cache.CC.SetStructAttributeIncrement(typeCountAttributeId, 1)
}

func (cache *PlayerCache) BuildQuantityDecrement(structTypeId uint64) {
    typeCountAttributeId := GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, cache.GetPlayerId(), structTypeId)
    cache.CC.SetStructAttributeDecrement(typeCountAttributeId, 1)
}

func (cache *PlayerCache) StoredOreEmpty() {
    cache.CC.ClearGridAttribute(cache.StoredOreAttributeId)
}

func (cache *PlayerCache) StoredOreDecrement(amount uint64) {
    cache.CC.SetGridAttributeDecrement(cache.StoredOreAttributeId, amount)
}

func (cache *PlayerCache) StoredOreIncrement(amount uint64) {
    cache.CC.SetGridAttributeIncrement(cache.StoredOreAttributeId, amount)
}

func (cache *PlayerCache) StructsLoadDecrement(amount uint64) {
    cache.CC.SetGridAttributeDecrement(cache.StructsLoadAttributeId, amount)
}

func (cache *PlayerCache) StructsLoadIncrement(amount uint64) {
    cache.CC.SetGridAttributeIncrement(cache.StructsLoadAttributeId, amount)
}

func (cache *PlayerCache) Discharge() {
    ctxSDK := sdk.UnwrapSDKContext(cache.CC.ctx)
    cache.CC.SetGridAttribute(cache.LastActionAttributeId, uint64(ctxSDK.BlockHeight()))
}

func (cache *PlayerCache) SetActiveAddress(address string) {
    cache.ActiveAddress = address
}

func (cache *PlayerCache) SetPlanetId(planetId string) {
    if (!cache.PlayerLoaded) { cache.LoadPlayer() }
    cache.Player.PlanetId = planetId
    cache.Changed = true
}

func (cache *PlayerCache) SetFleetId(fleetId string) {
    if (!cache.PlayerLoaded) { cache.LoadPlayer() }
    cache.Player.FleetId = fleetId
    cache.Changed = true
}

func (cache *PlayerCache) SetPrimaryAddress(address string) {
    if (!cache.PlayerLoaded) { cache.LoadPlayer() }
    cache.Player.PrimaryAddress = address
    cache.Changed = true
}

// DepositRefinedAlpha()
// Turn this into a delayed commit like the rest
func (cache *PlayerCache) DepositRefinedAlpha() {
    // Got this far, let's reward the player with some fresh Alpha
    // Mint the new Alpha to the module
    newAlpha, _ := sdk.ParseCoinsNormalized("1000000ualpha")
    cache.CC.k.bankKeeper.MintCoins(cache.CC.ctx, types.ModuleName, newAlpha)
    // Transfer the refined Alpha to the player
    playerAcc, _ := sdk.AccAddressFromBech32(cache.GetPrimaryAddress())
    cache.CC.k.bankKeeper.SendCoinsFromModuleToAccount(cache.CC.ctx, types.ModuleName, playerAcc, newAlpha)
}

func (cache *PlayerCache) IsOnline() (online bool){
    if ((cache.GetLoad() + cache.GetStructsLoad()) <= (cache.GetCapacity() + cache.GetCapacitySecondary())) {
        online = true
    } else {
        online = false
    }
    return
}

func (cache *PlayerCache) IsOffline() (bool){
    return !cache.IsOnline()
}

func (cache *PlayerCache) HasPlayerAccount() (bool){
    return cache.PlayerId != ""
}

func (cache *PlayerCache) HasPlanet() (bool){
    return (cache.GetPlanetId() != "")
}

func (cache *PlayerCache) HasStoredOre() (bool) {
    return (cache.GetStoredOre() > 0)
}

/* Permissions */
func (cache *PlayerCache) CanBePlayedBy(address string) (err error) {
    return cache.CanBeAdministratedBy(address, types.PermissionPlay)
}

func (cache *PlayerCache) CanBeHashedBy(address string) (err error) {
    return cache.CanBeAdministratedBy(address, types.PermissionHash)
}

func (cache *PlayerCache) CanBeUpdatedBy(address string) (err error) {
    return cache.CanBeAdministratedBy(address, types.PermissionUpdate)
}

func (cache *PlayerCache) CanManageGridBy(address string) (err error) {
    return cache.CanBeAdministratedBy(address, types.PermissionGrid)
}

func (cache *PlayerCache) CanBeAdministratedBy(address string, permission types.Permission) (error) {

    // Make sure the address calling this has request permissions
    if (!cache.CC.PermissionHasOneOf(GetAddressPermissionIDBytes(address), permission)) {
        return types.NewPermissionError("address", address, "", "", uint64(permission), "administrate")
    }

    if (cache.GetPrimaryAddress() != address) {
        callingPlayer, err := cache.CC.GetPlayerByAddress(address)
        if (err != nil) {
            return err
        }

        if (callingPlayer.GetPlayerId() != cache.GetPlayerId()) {
            if (!cache.CC.PermissionHasOneOf(GetObjectPermissionIDBytes(cache.GetPlayerId(), callingPlayer.GetPlayerId()), permission)) {
               return types.NewPermissionError("player", callingPlayer.GetPlayerId(), "player", cache.GetPlayerId(), uint64(permission), "administrate")
            }
        }
    }

    return nil
}

func (cache *PlayerCache) ReadinessCheck() (error) {
    if (cache.IsOffline()) {
        return types.NewPlayerPowerError(cache.PlayerId, "offline")
    }
    cache.Ready = true
    return nil
}


func (cache *PlayerCache) AttemptPlanetExplore() (err error) {
    planet := cache.CC.NewPlanet(cache.GetPrimaryAddress(), cache.GetPlayerId())
    cache.SetPlanetId(planet.GetPlanetId())
    return nil
}

func (cache *PlayerCache) CanSupportLoadAddition(additionalLoad uint64) (bool) {
    // Check player Load for the passiveDraw capacity
    totalLoad := cache.GetLoad() + cache.GetStructsLoad()
    totalCapacity := cache.GetCapacity() + cache.GetCapacitySecondary()

    if (totalLoad > totalCapacity) {
        return false
    }

    return ((totalCapacity - totalLoad) >= additionalLoad)
}


func (cache *PlayerCache) SetGuild(guildId string) {
    if (!cache.PlayerLoaded) { cache.LoadPlayer() }

    cache.Player.GuildId = guildId
    cache.Changed = true
}

func (cache *PlayerCache) ClearGuild() {
    if (!cache.PlayerLoaded) { cache.LoadPlayer() }
    cache.Player.GuildId = ""
    cache.Changed = true
}

func (cache *PlayerCache) MigrateSubstation(substationId string){

    cache.DisconnectSubstation()

    if (substationId != "") {
        cache.Player.SubstationId = substationId
        cache.CC.k.SetSubstationPlayerIndex(cache.CC.ctx, cache.GetSubstationId(), cache.GetPlayerId())
        cache.GetSubstation().ConnectionCountIncrement(1)
        cache.Changed = true
    }
}

func (cache *PlayerCache) DisconnectSubstation(){
    if (!cache.PlayerLoaded) {
        cache.LoadPlayer()
    }

    if (cache.GetSubstationId() != "") {
        cache.GetSubstation().ConnectionCountDecrement(1)
        cache.CC.k.RemoveSubstationPlayerIndex(cache.CC.ctx, cache.GetSubstationId(), cache.GetPlayerId())
        cache.Player.SubstationId = ""
        cache.Changed = true
    }
}


func (cache *PlayerCache) MigrateGuild(guild *GuildCache){
    if (!cache.PlayerLoaded) {
        cache.LoadPlayer()
    }

    cache.Player.GuildId = guild.GetGuildId()
    cache.Changed = true
}

func (cache *PlayerCache) LeaveGuild(){
    if (!cache.PlayerLoaded) {
        cache.LoadPlayer()
    }

    cache.Player.GuildId = ""
    cache.Changed = true
}