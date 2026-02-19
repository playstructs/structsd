package types

import (
	"cosmossdk.io/math"
)

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
		Defusing: 0,
		Address: playerAddress,
		PlayerId: playerId,
	}
}

func (a *Infusion) Recalculate() {
    a.Power = a.Ratio * a.Fuel
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

