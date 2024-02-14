package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"
)

func (a *Infusion) SetCommission(newCommission sdk.Dec) (
                                                newInfusionEnergy uint64,
                                                oldInfusionEnergy uint64,
                                                newCommissionEnergy uint64,
                                                oldCommissionEnergy uint64,
                                                newPlayerEnergy uint64,
                                                oldPlayerEnergy uint64,
                                                err error)  {

    oldInfusionEnergy       = a.Energy
    oldCommissionEnergy     = a.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(oldInfusionEnergy))).RoundInt().Uint64()
    oldPlayerEnergy         = a.Energy - oldCommissionEnergy


    newInfusionEnergy       = CalculateInfusionEnergy(a.DestinationType, a.Fuel)
    newCommissionEnergy     = newCommission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(newInfusionEnergy))).RoundInt().Uint64()
    newPlayerEnergy         = newInfusionEnergy - newCommissionEnergy


	a.Commission  = newCommission
	a.Energy      = newInfusionEnergy

    err           = nil
	return

}

func (a *Infusion) SetFuel(newFuel uint64) (
                                    newInfusionEnergy uint64,
                                    oldInfusionEnergy uint64,
                                    newCommissionEnergy uint64,
                                    oldCommissionEnergy uint64,
                                    newPlayerEnergy uint64,
                                    oldPlayerEnergy uint64,
                                    err error)  {

    oldInfusionEnergy       = a.Energy
    oldCommissionEnergy     = a.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(oldInfusionEnergy))).RoundInt().Uint64()
    oldPlayerEnergy         = a.Energy - oldCommissionEnergy


    newInfusionEnergy       = CalculateInfusionEnergy(a.DestinationType, a.Fuel)
    newCommissionEnergy     = a.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(newInfusionEnergy))).RoundInt().Uint64()
    newPlayerEnergy         = newInfusionEnergy - newCommissionEnergy


    a.Fuel      = newFuel
	a.Energy    = newInfusionEnergy

    err         = nil
	return

}

func (a *Infusion) SetFuelAndCommission(newFuel uint64, newCommission sdk.Dec) (
                                    newInfusionEnergy uint64,
                                    oldInfusionEnergy uint64,
                                    newCommissionEnergy uint64,
                                    oldCommissionEnergy uint64,
                                    newPlayerEnergy uint64,
                                    oldPlayerEnergy uint64,
                                    err error)  {

    oldInfusionEnergy       = a.Energy
    oldCommissionEnergy     = a.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(oldInfusionEnergy))).RoundInt().Uint64()
    oldPlayerEnergy         = a.Energy - oldCommissionEnergy


    newInfusionEnergy       = CalculateInfusionEnergy(a.DestinationType, a.Fuel)
    newCommissionEnergy     = newCommission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(newInfusionEnergy))).RoundInt().Uint64()
    newPlayerEnergy         = newInfusionEnergy - newCommissionEnergy


    a.Commission    = newCommission
    a.Fuel          = newFuel
	a.Energy        = newInfusionEnergy

    err         = nil
	return

}

func (a *Infusion) getEnergyDistribution() (infusionEnergy uint64, commissionEnergy uint64, playerEnergy uint64) {
        infusionEnergy       = a.Energy
        commissionEnergy     = a.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(infusionEnergy))).RoundInt().Uint64()
        playerEnergy         = infusionEnergy - commissionEnergy

        return
}

func CalculateInfusionEnergy(destinationType ObjectType, fuel uint64) (energy uint64) {
    switch destinationType {
        case ObjectType_reactor:
            energy = CalculateReactorEnergy(fuel)
        case ObjectType_struct:
            energy = CalculateStructEnergy(fuel)
    }

    return
}

func CreateNewInfusion(destinationType ObjectType, destinationId uint64, playerAddress string, playerId uint64, fuel uint64, commission sdk.Dec) Infusion {
	return Infusion{
		DestinationType: destinationType,
		DestinationId: destinationId,
		Commission: commission,
		Fuel: fuel,
		Energy: CalculateInfusionEnergy(destinationType, fuel),
		Address: playerAddress,
		PlayerId: playerId,
	}
}



/*
 * Only Reactors and Structs (Power Plants) can have Alpha infused
 *
 * Use this function anytime a user is providing the objectType of the source objectType
 */
func IsValidInfusionConnectionType(objectType ObjectType) bool {
	for _, a := range []ObjectType{ObjectType_reactor, ObjectType_struct} {
		if a == objectType {
			return true
		}
	}
	return false
}

