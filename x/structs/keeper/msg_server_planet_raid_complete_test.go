package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgPlanetRaidComplete(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1_000_000_000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	// Attacker player
	attackerAcc := sdk.AccAddress("attacker1234567890123456789012345678")
	attacker := types.Player{
		Creator:        attackerAcc.String(),
		PrimaryAddress: attackerAcc.String(),
	}
	attacker = testAppendPlayer(k, sdkCtx, attacker)
	attackerCapAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, attacker.Id)
	k.SetGridAttribute(sdkCtx, attackerCapAttrId, uint64(100000))

	// Attacker's home planet
	homePlanet := testAppendPlanet(k, sdkCtx, types.Planet{Creator: attacker.Creator, Owner: attacker.Id})
	attacker.PlanetId = homePlanet.Id
	k.SetPlayer(sdkCtx, attacker)

	// Target player with planet
	targetAcc := sdk.AccAddress("target12345678901234567890123456789")
	target := types.Player{
		Creator:        targetAcc.String(),
		PrimaryAddress: targetAcc.String(),
	}
	target = testAppendPlayer(k, sdkCtx, target)
	targetCapAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, target.Id)
	k.SetGridAttribute(sdkCtx, targetCapAttrId, uint64(100000))

	targetPlanet := testAppendPlanet(k, sdkCtx, types.Planet{Creator: target.Creator, Owner: target.Id})
	target.PlanetId = targetPlanet.Id
	k.SetPlayer(sdkCtx, target)

	// Give target player some ore to steal
	targetOreAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, target.Id)
	k.SetGridAttribute(sdkCtx, targetOreAttrId, uint64(50))

	// Set BlockStartRaid on the target planet
	blockStartRaidAttrId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockStartRaid, targetPlanet.Id)
	k.SetPlanetAttribute(sdkCtx, blockStartRaidAttrId, uint64(1))

	// Create attacker's fleet at the target planet
	fleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:        attacker.Id,
		LocationId:   targetPlanet.Id,
		LocationType: types.ObjectType_planet,
		Status:       types.FleetStatus_away,
	})
	attacker.FleetId = fleet.Id
	k.SetPlayer(sdkCtx, attacker)

	t.Run("valid raid complete", func(t *testing.T) {
		hashTemplate := fmt.Sprintf("%s@%sRAID1NONCE%%s", fleet.Id, targetPlanet.Id)
		nonce, proof := testFindProof(hashTemplate, 1)

		resp, err := ms.PlanetRaidComplete(wctx, &types.MsgPlanetRaidComplete{
			Creator: attacker.Creator,
			FleetId: fleet.Id,
			Nonce:   nonce,
			Proof:   proof,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("fleet not found", func(t *testing.T) {
		_, err := ms.PlanetRaidComplete(wctx, &types.MsgPlanetRaidComplete{
			Creator: attacker.Creator,
			FleetId: "invalid-fleet",
			Nonce:   "test-nonce",
			Proof:   "test-proof",
		})
		require.Error(t, err)
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.PlanetRaidComplete(wctx, &types.MsgPlanetRaidComplete{
			Creator: "cosmos1noperms",
			FleetId: fleet.Id,
			Nonce:   "test-nonce",
			Proof:   "test-proof",
		})
		require.Error(t, err)
	})
}
