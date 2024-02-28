package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"
)

func (a *Infusion) SetCommission(newCommission sdk.Dec) (
                                                newInfusionPower uint64,
                                                oldInfusionPower uint64,
                                                newCommissionPower uint64,
                                                oldCommissionPower uint64,
                                                newPlayerPower uint64,
                                                oldPlayerPower uint64,
                                                err error)  {

    oldInfusionPower       = a.Power
    oldCommissionPower     = a.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(oldInfusionPower))).RoundInt().Uint64()
    oldPlayerPower         = a.Power - oldCommissionPower


    newInfusionPower       = CalculateInfusionPower(a.DestinationType, a.Fuel)
    newCommissionPower     = newCommission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(newInfusionPower))).RoundInt().Uint64()
    newPlayerPower         = newInfusionPower - newCommissionPower


	a.Commission  = newCommission
	a.Power      = newInfusionPower

    err           = nil
	return

}

func (a *Infusion) SetFuel(newFuel uint64) (
                                    newInfusionFuel uint64,
                                    oldInfusionFuel uint64,
                                    newInfusionPower uint64,
                                    oldInfusionPower uint64,
                                    newCommissionPower uint64,
                                    oldCommissionPower uint64,
                                    newPlayerPower uint64,
                                    oldPlayerPower uint64,
                                    err error)  {

    oldInfusionFuel         = a.Fuel
    oldInfusionPower       = a.Power
    oldCommissionPower     = a.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(oldInfusionPower))).RoundInt().Uint64()
    oldPlayerPower         = a.Power - oldCommissionPower

    newInfusionFuel         = newFuel
    newInfusionPower       = CalculateInfusionPower(a.DestinationType, newInfusionFuel)
    newCommissionPower     = a.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(newInfusionPower))).RoundInt().Uint64()
    newPlayerPower         = newInfusionPower - newCommissionPower


    a.Fuel      = newFuel
	a.Power    = newInfusionPower

    err         = nil
	return

}

func (a *Infusion) SetFuelAndCommission(newFuel uint64, newCommission sdk.Dec) (
                                    newInfusionFuel uint64,
                                    oldInfusionFuel uint64,
                                    newInfusionPower uint64,
                                    oldInfusionPower uint64,
                                    newCommissionPower uint64,
                                    oldCommissionPower uint64,
                                    newPlayerPower uint64,
                                    oldPlayerPower uint64,
                                    err error)  {

    oldInfusionFuel         = a.Fuel
    oldInfusionPower       = a.Power
    oldCommissionPower     = a.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(oldInfusionPower))).RoundInt().Uint64()
    oldPlayerPower         = a.Power - oldCommissionPower

    newInfusionFuel         = newFuel
    newInfusionPower       = CalculateInfusionPower(a.DestinationType, newInfusionFuel)
    newCommissionPower     = newCommission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(newInfusionPower))).RoundInt().Uint64()
    newPlayerPower         = newInfusionPower - newCommissionPower


    a.Commission    = newCommission
    a.Fuel          = newFuel
	a.Power        = newInfusionPower

    err         = nil
	return

}

func (a *Infusion) GetPowerDistribution() (infusionPower uint64, commissionPower uint64, playerPower uint64) {
        infusionPower       = a.Power
        commissionPower     = a.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(infusionPower))).RoundInt().Uint64()
        playerPower         = infusionPower - commissionPower

        return
}

func CalculateInfusionPower(destinationType ObjectType, fuel uint64) (energy uint64) {
    switch destinationType {
        case ObjectType_reactor:
            energy = CalculateReactorPower(fuel)
        case ObjectType_struct:
            energy = CalculateStructPower(fuel)
    }

    return
}

func CreateNewInfusion(destinationType ObjectType, destinationId string, playerAddress string, playerId string, fuel uint64, commission sdk.Dec) Infusion {
	return Infusion{
		DestinationType: destinationType,
		DestinationId: destinationId,
		Commission: commission,
		Fuel: fuel,
		Power: CalculateInfusionPower(destinationType, fuel),
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

