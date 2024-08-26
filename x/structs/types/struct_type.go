package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"github.com/nethruster/go-fraction"
)


func (structType *StructType) VerifyWeaponSystem(weaponSystem TechWeaponSystem) (err error) {
    switch weaponSystem {
        case TechWeaponSystem_primaryWeapon:
            if (structType.PrimaryWeapon == TechActiveWeaponry_noActiveWeaponry) {
                err = sdkerrors.Wrapf(ErrObjectNotFound, "No valid primary weapon system")
            }
        case TechWeaponSystem_secondaryWeapon:
            if (structType.SecondaryWeapon == TechActiveWeaponry_noActiveWeaponry) {
                err = sdkerrors.Wrapf(ErrObjectNotFound, "No valid secondary weapon system")
            }
        default:
            err = sdkerrors.Wrapf(ErrObjectNotFound, "No valid weapon system provided")
    }
    return
}

func (structType *StructType) GetWeapon(weaponSystem TechWeaponSystem) (weapon TechActiveWeaponry) {
    switch weaponSystem {
        case TechWeaponSystem_primaryWeapon:
            weapon = structType.PrimaryWeapon

        case TechWeaponSystem_secondaryWeapon:
            weapon = structType.SecondaryWeapon
    }
    return weapon
}

func (structType *StructType) GetWeaponControl(weaponSystem TechWeaponSystem) (weaponControl TechWeaponControl) {
    switch weaponSystem {
        case TechWeaponSystem_primaryWeapon:
            weaponControl = structType.PrimaryWeaponControl

        case TechWeaponSystem_secondaryWeapon:
            weaponControl = structType.SecondaryWeaponControl
    }
    return weaponControl
}


func (structType *StructType) GetWeaponCharge(weaponSystem TechWeaponSystem) (weaponCharge uint64) {
    switch weaponSystem {
        case TechWeaponSystem_primaryWeapon:
            weaponCharge = structType.PrimaryWeaponCharge

        case TechWeaponSystem_secondaryWeapon:
            weaponCharge = structType.SecondaryWeaponCharge
    }
    return weaponCharge
}

func (structType *StructType) GetWeaponAmbits(weaponSystem TechWeaponSystem) (weaponAmbits uint64) {
    switch weaponSystem {
        case TechWeaponSystem_primaryWeapon:
            weaponAmbits = structType.PrimaryWeaponAmbits

        case TechWeaponSystem_secondaryWeapon:
            weaponAmbits = structType.SecondaryWeaponAmbits
    }
    return weaponAmbits
}

func (structType *StructType) GetWeaponTargets(weaponSystem TechWeaponSystem) (weaponTargets uint64) {
    switch weaponSystem {
        case TechWeaponSystem_primaryWeapon:
            weaponTargets = structType.PrimaryWeaponTargets

        case TechWeaponSystem_secondaryWeapon:
            weaponTargets = structType.SecondaryWeaponTargets
    }
    return weaponTargets
}

func (structType *StructType) GetWeaponShots(weaponSystem TechWeaponSystem) (weaponShots uint64) {
    switch weaponSystem {
        case TechWeaponSystem_primaryWeapon:
            weaponShots = structType.PrimaryWeaponShots

        case TechWeaponSystem_secondaryWeapon:
            weaponShots = structType.SecondaryWeaponShots
    }
    return weaponShots
}

func (structType *StructType) GetWeaponDamage(weaponSystem TechWeaponSystem) (weaponDamage uint64) {
    switch weaponSystem {
        case TechWeaponSystem_primaryWeapon:
            weaponDamage = structType.PrimaryWeaponDamage

        case TechWeaponSystem_secondaryWeapon:
            weaponDamage = structType.SecondaryWeaponDamage
    }
    return weaponDamage
}

func (structType *StructType) GetWeaponBlockable(weaponSystem TechWeaponSystem) (weaponBlockable bool) {
    switch weaponSystem {
        case TechWeaponSystem_primaryWeapon:
            weaponBlockable = structType.PrimaryWeaponBlockable

        case TechWeaponSystem_secondaryWeapon:
            weaponBlockable = structType.SecondaryWeaponBlockable
    }
    return weaponBlockable
}

func (structType *StructType) GetWeaponCounterable(weaponSystem TechWeaponSystem) (weaponCounterable bool) {
    switch weaponSystem {
        case TechWeaponSystem_primaryWeapon:
            weaponCounterable = structType.PrimaryWeaponCounterable

        case TechWeaponSystem_secondaryWeapon:
            weaponCounterable = structType.SecondaryWeaponCounterable
    }
    return weaponCounterable
}

func (structType *StructType) GetWeaponRecoilDamage(weaponSystem TechWeaponSystem) (weaponRecoilDamage uint64) {
    switch weaponSystem {
        case TechWeaponSystem_primaryWeapon:
            weaponRecoilDamage = structType.PrimaryWeaponRecoilDamage

        case TechWeaponSystem_secondaryWeapon:
            weaponRecoilDamage = structType.SecondaryWeaponRecoilDamage
    }
    return weaponRecoilDamage
}

func (structType *StructType) GetWeaponShotSuccessRate(weaponSystem TechWeaponSystem) (weaponShotSuccessRate fraction.Fraction) {
    switch weaponSystem {
        case TechWeaponSystem_primaryWeapon:
            weaponShotSuccessRate, _ = fraction.New(structType.PrimaryWeaponShotSuccessRateNumerator, structType.PrimaryWeaponShotSuccessRateDenominator)

        case TechWeaponSystem_secondaryWeapon:
            weaponShotSuccessRate, _ = fraction.New(structType.SecondaryWeaponShotSuccessRateNumerator, structType.SecondaryWeaponShotSuccessRateDenominator)
    }
    return weaponShotSuccessRate
}


