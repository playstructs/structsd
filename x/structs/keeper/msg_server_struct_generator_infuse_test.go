package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructGeneratorInfuse(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create a player first
	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = k.AppendPlayer(ctx, player)

	// Create a planet
	planetId := k.AppendPlanet(ctx, player)

	// Create a struct type with power generation
	// Note: PowerGeneration is an enum, use a valid value
	structType := types.StructType{
		Id:              1,
		Type:            types.CommandStruct,
		Category:        types.ObjectType_player,
		PowerGeneration: 1, // Use a non-zero value to indicate power generation
	}
	k.SetStructType(ctx, structType)

	// Create a struct
	structObj := types.Struct{
		Creator:      player.Creator,
		Owner:        player.Id,
		Type:         structType.Id,
		LocationId:   planetId,
		LocationType: types.ObjectType_planet,
	}
	structObj = k.AppendStruct(ctx, structObj)

	// Mark struct as built and online
	statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structObj.Id)
	builtFlag := uint64(types.StructStateBuilt)
	k.SetStructAttributeFlagAdd(ctx, statusAttrId, builtFlag)

	// Grant permissions
	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(player.Creator)
	k.PermissionAdd(ctx, addressPermissionId, types.PermissionAssets)

	// Set up balances
	playerAcc, _ := sdk.AccAddressFromBech32(player.Creator)
	coins := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(1000)))
	k.BankKeeper().MintCoins(ctx, types.ModuleName, coins)
	k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, playerAcc, coins)

	testCases := []struct {
		name      string
		input     *types.MsgStructGeneratorInfuse
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid generator infuse",
			input: &types.MsgStructGeneratorInfuse{
				Creator:      player.Creator,
				StructId:     structObj.Id,
				InfuseAmount: "1000ualpha",
			},
			expErr: false,
		},
		{
			name: "struct not found",
			input: &types.MsgStructGeneratorInfuse{
				Creator:      player.Creator,
				StructId:     "invalid-struct",
				InfuseAmount: "1000ualpha",
			},
			expErr:    true,
			expErrMsg: "not found",
		},
		{
			name: "struct offline",
			input: &types.MsgStructGeneratorInfuse{
				Creator:      player.Creator,
				StructId:     structObj.Id,
				InfuseAmount: "1000ualpha",
			},
			expErr:    true,
			expErrMsg: "is offline",
		},
		{
			name: "no power generation",
			input: &types.MsgStructGeneratorInfuse{
				Creator:      player.Creator,
				StructId:     structObj.Id,
				InfuseAmount: "1000ualpha",
			},
			expErr:    true,
			expErrMsg: "has no generation systems",
		},
		{
			name: "no permissions",
			input: &types.MsgStructGeneratorInfuse{
				Creator:      "cosmos1noperms",
				StructId:     structObj.Id,
				InfuseAmount: "1000ualpha",
			},
			expErr:    true,
			expErrMsg: "has no assets permissions",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up struct state for each test
			if tc.name == "valid generator infuse" {
				// Ensure struct is online
				onlineFlag := uint64(types.StructStateOnline)
				k.SetStructAttributeFlagAdd(ctx, statusAttrId, onlineFlag)
			} else if tc.name == "struct offline" {
				// Ensure struct is offline
				onlineFlag := uint64(types.StructStateOnline)
				k.SetStructAttributeFlagRemove(ctx, statusAttrId, onlineFlag)
			} else if tc.name == "no power generation" {
				// Create struct type without power generation
				noGenType := types.StructType{
					Id:              2,
					Type:            types.CommandStruct,
					Category:        types.ObjectType_player,
					PowerGeneration: types.TechPowerGeneration_noPowerGeneration,
				}
				k.SetStructType(ctx, noGenType)
				structObj.Type = noGenType.Id
				k.SetStruct(ctx, structObj)
			}

			resp, err := ms.StructGeneratorInfuse(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}
		})
	}
}
