package app_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestAllocationPowerKeepsSourcePlayerSharesCurrent(t *testing.T) {
	for _, kind := range []types.AllocationType{types.AllocationType_static, types.AllocationType_dynamic, types.AllocationType_automated, types.AllocationType_providerAgreement} {
		t.Run(kind.String(), func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			k := bApp.StructsKeeper
			newPlayer := func() types.Player {
				return appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
			}
			owner, first, second := newPlayer(), newPlayer(), newPlayer()
			const source, destination = "4-968", "4-969"
			for _, id := range []string{source, destination} {
				k.SetSubstation(ctx, types.Substation{Id: id, Owner: owner.Id})
			}
			k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner.Id), 1000)
			cc := k.NewCurrentContext(ctx)
			_, err := cc.NewAllocation(types.AllocationType_dynamic, owner.Id, source, owner.Creator, owner.Id, 1000)
			require.NoError(t, err)
			cc.GetPlayer(first.Id).MigrateSubstation(source)
			cc.GetPlayer(second.Id).MigrateSubstation(source)
			cc.CommitAll()
			checkShare := func(power uint64) {
				t.Helper()
				current := k.NewCurrentContext(ctx)
				want := (900 - power) / 2
				require.Equal(t, want, current.GetSubstation(source).GetConnectionCapacity())
				require.Equal(t, want, current.GetPlayer(first.Id).GetCapacitySecondary())
				require.Equal(t, want, current.GetPlayer(second.Id).GetCapacitySecondary())
				require.Equal(t, power, current.GetSubstation(source).GetLoad())
				require.Equal(t, power, current.GetSubstation(destination).GetCapacity())
			}
			checkShare(0)
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			cc = k.NewCurrentContext(ctx)
			allocation, err := cc.NewAllocation(kind, source, destination, owner.Creator, owner.Id, 300)
			require.NoError(t, err)
			cc.CommitAll()
			power := uint64(300)
			if kind == types.AllocationType_automated {
				power = 900
			}
			checkShare(power)
			shareEvents := 0
			for _, event := range ctx.EventManager().ABCIEvents() {
				if event.Type == "structs.structs.EventGrid" {
					parsed, err := sdk.ParseTypedEvent(event)
					require.NoError(t, err)
					row := parsed.(*types.EventGrid).GridRecord
					if row.AttributeId == keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, source) {
						require.Equal(t, (900-power)/2, row.Value)
						shareEvents++
					}
				}
			}
			require.Equal(t, 1, shareEvents)
			if kind == types.AllocationType_dynamic {
				for _, next := range []uint64{500, 100} {
					cc = k.NewCurrentContext(ctx)
					cached, found := cc.GetAllocation(allocation.ID())
					require.True(t, found)
					_, err = cached.SetDynamicPower(next)
					require.NoError(t, err)
					cc.CommitAll()
					checkShare(next)
				}
			}
			cc = k.NewCurrentContext(ctx)
			cached, found := cc.GetAllocation(allocation.ID())
			require.True(t, found)
			require.NoError(t, cached.Destroy())
			cc.CommitAll()
			checkShare(0)
		})
	}
}
