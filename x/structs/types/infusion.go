package types

import (
	"cosmossdk.io/math"
)

func (a *Infusion) SetCommission(newCommission math.LegacyDec) (
                                                newInfusionPower uint64,
                                                oldInfusionPower uint64,
                                                newCommissionPower uint64,
                                                oldCommissionPower uint64,
                                                newPlayerPower uint64,
                                                oldPlayerPower uint64,
                                                newRatio uint64,
                                                oldRatio uint64,
                                                err error)  {

    oldRatio               = a.Ratio
    oldInfusionPower       = a.Power
    oldCommissionPower     = a.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(oldInfusionPower))).RoundInt().Uint64()
    oldPlayerPower         = a.Power - oldCommissionPower


    newRatio                    = a.Ratio
    newInfusionPower            = CalculateInfusionPower(newRatio, a.Fuel)
    newCommissionPower          = newCommission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(newInfusionPower))).RoundInt().Uint64()
    newPlayerPower              = newInfusionPower - newCommissionPower


	a.Commission  = newCommission
	a.Power       = newInfusionPower
    a.Ratio       = newRatio

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
                                    newRatio uint64,
                                    oldRatio uint64,
                                    err error)  {

    oldRatio               = a.Ratio
    oldInfusionFuel        = a.Fuel
    oldInfusionPower       = a.Power
    oldCommissionPower     = a.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(oldInfusionPower))).RoundInt().Uint64()
    oldPlayerPower         = a.Power - oldCommissionPower

    newInfusionFuel             = newFuel
    newRatio                    = a.Ratio
    newInfusionPower            = CalculateInfusionPower(newRatio, newInfusionFuel)
    newCommissionPower          = a.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(newInfusionPower))).RoundInt().Uint64()
    newPlayerPower              = newInfusionPower - newCommissionPower

    a.Fuel     = newFuel
	a.Power    = newInfusionPower
    a.Ratio    = newRatio

    err         = nil
	return
}

func (a *Infusion) SetFuelAndCommission(newFuel uint64, newCommission math.LegacyDec) (
                                    newInfusionFuel uint64,
                                    oldInfusionFuel uint64,
                                    newInfusionPower uint64,
                                    oldInfusionPower uint64,
                                    newCommissionPower uint64,
                                    oldCommissionPower uint64,
                                    newPlayerPower uint64,
                                    oldPlayerPower uint64,
                                    newRatio uint64,
                                    oldRatio uint64,
                                    err error)  {

    oldRatio               = a.Ratio
    oldInfusionFuel        = a.Fuel
    oldInfusionPower       = a.Power
    oldCommissionPower     = a.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(oldInfusionPower))).RoundInt().Uint64()
    oldPlayerPower         = a.Power - oldCommissionPower

    newInfusionFuel             = newFuel
    newRatio                    = a.Ratio
    newInfusionPower            = CalculateInfusionPower(newRatio, newInfusionFuel)
    newCommissionPower          = newCommission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(newInfusionPower))).RoundInt().Uint64()
    newPlayerPower              = newInfusionPower - newCommissionPower


    a.Commission    = newCommission
    a.Fuel          = newFuel
	a.Power         = newInfusionPower
    a.Ratio         = newRatio

    err         = nil
	return

}

func (a *Infusion) GetPowerDistribution() (infusionPower uint64, commissionPower uint64, playerPower uint64) {
        infusionPower       = a.Power
        commissionPower     = a.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(infusionPower))).RoundInt().Uint64()
        playerPower         = infusionPower - commissionPower

        return
}

func CalculateInfusionPower(ratio uint64, fuel uint64) (uint64) {
    return ratio * fuel
}

func CreateNewInfusion(destinationType ObjectType, destinationId string, playerAddress string, playerId string, fuel uint64, commission math.LegacyDec, ratio uint64) Infusion {

	power := CalculateInfusionPower(ratio, fuel)

	return Infusion{
		DestinationType: destinationType,
		DestinationId: destinationId,
		Commission: commission,
		Fuel: fuel,
		Power: power,
		Ratio: ratio,
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

