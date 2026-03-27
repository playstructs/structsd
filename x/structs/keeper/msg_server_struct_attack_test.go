package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructAttack(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	attackerPlayer := types.Player{
		Creator:        "cosmos1attacker",
		PrimaryAddress: "cosmos1attacker",
	}
	attackerPlayer = testAppendPlayer(k, sdkCtx, attackerPlayer)

	attackerCapAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, attackerPlayer.Id)
	k.SetGridAttribute(sdkCtx, attackerCapAttrId, uint64(100000))
	attackerLastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, attackerPlayer.Id)
	k.SetGridAttribute(sdkCtx, attackerLastActionAttrId, uint64(0))

	targetPlayer := types.Player{
		Creator:        "cosmos1target",
		PrimaryAddress: "cosmos1target",
	}
	targetPlayer = testAppendPlayer(k, sdkCtx, targetPlayer)
	targetCapAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, targetPlayer.Id)
	k.SetGridAttribute(sdkCtx, targetCapAttrId, uint64(100000))

	planet := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator:   targetPlayer.Creator,
		Owner:     targetPlayer.Id,
		LandSlots: 4,
		Land:      []string{"", "", "", ""},
	})
	targetPlayer.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, targetPlayer)

	cmdStructType := types.StructType{
		Id:       1,
		Type:     types.CommandStruct,
		Category: types.ObjectType_fleet,
	}
	k.SetStructType(sdkCtx, cmdStructType)

	attackStructType := types.StructType{
		Id:                   2,
		Type:                 "Gunship",
		Category:             types.ObjectType_fleet,
		PrimaryWeapon:        1,
		PrimaryWeaponCharge:  10,
		PrimaryWeaponTargets: 1,
		PrimaryWeaponAmbits:  0xFFFF,
		PrimaryWeaponDamage:  5,
		PrimaryWeaponBlockable: true,
		PossibleAmbit:        1 << uint64(types.Ambit_space),
	}
	k.SetStructType(sdkCtx, attackStructType)

	targetStructType := types.StructType{
		Id:            3,
		Type:          "Turret",
		Category:      types.ObjectType_planet,
		PossibleAmbit: 1 << uint64(types.Ambit_land),
	}
	k.SetStructType(sdkCtx, targetStructType)

	fleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:      attackerPlayer.Id,
		LocationId: planet.Id,
		Status:     types.FleetStatus_away,
	})

	cmdStruct := types.Struct{
		Creator:        attackerPlayer.Creator,
		Owner:          attackerPlayer.Id,
		Type:           cmdStructType.Id,
		LocationId:     fleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_space,
	}
	cmdStruct = testAppendStruct(k, sdkCtx, cmdStruct)
	cmdStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, cmdStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdStatusAttrId, uint64(types.StructStateOnline))

	fleet.CommandStruct = cmdStruct.Id
	k.SetFleet(sdkCtx, fleet)
	attackerPlayer.FleetId = fleet.Id
	k.SetPlayer(sdkCtx, attackerPlayer)

	attackerStruct := types.Struct{
		Creator:        attackerPlayer.Creator,
		Owner:          attackerPlayer.Id,
		Type:           attackStructType.Id,
		LocationId:     fleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_space,
	}
	attackerStruct = testAppendStruct(k, sdkCtx, attackerStruct)
	atkStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, attackerStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, atkStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, atkStatusAttrId, uint64(types.StructStateOnline))

	targetStruct := types.Struct{
		Creator:        targetPlayer.Creator,
		Owner:          targetPlayer.Id,
		Type:           targetStructType.Id,
		LocationId:     planet.Id,
		LocationType:   types.ObjectType_planet,
		OperatingAmbit: types.Ambit_land,
	}
	targetStruct = testAppendStruct(k, sdkCtx, targetStruct)
	tgtStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, targetStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, tgtStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, tgtStatusAttrId, uint64(types.StructStateOnline))

	t.Run("valid attack", func(t *testing.T) {
		k.SetGridAttribute(sdkCtx, attackerLastActionAttrId, uint64(0))
		resp, err := ms.StructAttack(wctx, &types.MsgStructAttack{
			Creator:           attackerPlayer.Creator,
			OperatingStructId: attackerStruct.Id,
			WeaponSystem:      "primaryWeapon",
			TargetStructId:    []string{targetStruct.Id},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("struct not found", func(t *testing.T) {
		_, err := ms.StructAttack(wctx, &types.MsgStructAttack{
			Creator:           attackerPlayer.Creator,
			OperatingStructId: "invalid-struct",
			WeaponSystem:      "primaryWeapon",
			TargetStructId:    []string{targetStruct.Id},
		})
		require.Error(t, err)
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructAttack(wctx, &types.MsgStructAttack{
			Creator:           "cosmos1noperms",
			OperatingStructId: attackerStruct.Id,
			WeaponSystem:      "primaryWeapon",
			TargetStructId:    []string{targetStruct.Id},
		})
		require.Error(t, err)
	})
}
