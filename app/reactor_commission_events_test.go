package app_test

import (
	"testing"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	"structs/app"
	"structs/x/structs/types"
)

func setupCommissionEventApp(t *testing.T) (*app.App, sdk.Context, sdk.ValAddress, string) {
	t.Helper()
	bApp, ctx := setupJailGateApp(t)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	primary, _, _ := newTransferPlayer(t, bApp, ctx)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, primary, validator, math.NewInt(1000000))
	id := jailGateReactorId(t, bApp, ctx, validator)

	return bApp, ctx.WithEventManager(sdk.NewEventManager()), validator, id
}

func slashCommissionEventValidator(t *testing.T, bApp *app.App, ctx sdk.Context, validator sdk.ValAddress) {
	t.Helper()
	value, err := bApp.StakingKeeper.GetValidator(ctx, validator)
	require.NoError(t, err)
	consensus, err := value.GetConsAddr()
	require.NoError(t, err)
	burned, err := bApp.StakingKeeper.Slash(ctx, consensus, ctx.BlockHeight(), value.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx)), math.LegacyNewDecWithPrec(5, 2))
	require.NoError(t, err)
	require.True(t, burned.IsPositive())
}

func commissionEvents(t *testing.T, events []abci.Event) (resets int, revisions []*types.EventReactorCommissionRevision) {
	t.Helper()
	for _, event := range events {
		// FinalizeBlock appends an SDK mode label that is not JSON-encoded
		// protobuf payload. Preserve every actual message attribute.
		payload := abci.Event{Type: event.Type}
		for _, attribute := range event.Attributes {
			if attribute.Key != "mode" {
				payload.Attributes = append(payload.Attributes, attribute)
			}
		}
		switch event.Type {
		case "structs.structs.EventReactorCommissionRevisionReset":
			_, err := sdk.ParseTypedEvent(payload)
			require.NoError(t, err)
			resets++
		case "structs.structs.EventReactorCommissionRevision":
			decoded, err := sdk.ParseTypedEvent(payload)
			require.NoError(t, err)
			revisions = append(revisions, decoded.(*types.EventReactorCommissionRevision))
		}
	}
	return
}

func TestCommissionRevisionEventsTrackSlashRepairAndStakeShareQuery(t *testing.T) {
	bApp, ctx, validator, id := setupCommissionEventApp(t)
	query := func(issued, certified uint64) {
		t.Helper()
		before := len(ctx.EventManager().Events())
		response, err := bApp.StructsKeeper.ReactorStakeShare(ctx, &types.QueryReactorStakeShareRequest{ReactorId: id})
		require.NoError(t, err)
		require.Equal(t, issued, response.SlashRevision)
		require.Equal(t, certified, response.ReconciledSlashRevision)
		require.Equal(t, issued != certified, response.CommissionRepairPending)
		var decoded types.QueryReactorStakeShareResponse
		require.NoError(t, bApp.AppCodec().UnmarshalJSON(bApp.AppCodec().MustMarshalJSON(response), &decoded))
		require.Equal(t, *response, decoded)
		require.Len(t, ctx.EventManager().Events(), before, "query is read-only")
	}
	query(0, 0)
	bApp.StructsKeeper.EnqueueReactorRestart(ctx, id)
	_, revisions := commissionEvents(t, ctx.EventManager().ABCIEvents())
	require.Empty(t, revisions)
	slashCommissionEventValidator(t, bApp, ctx, validator)
	query(1, 0)
	rows := bApp.StructsKeeper.GetAllReactorInfusions(ctx, id)
	require.NotEmpty(t, rows)
	original := rows[0]
	broken := original
	broken.DestinationType = types.ObjectType(500)
	bApp.StructsKeeper.SetInfusion(ctx, broken)
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	query(1, 0)
	_, revisions = commissionEvents(t, ctx.EventManager().ABCIEvents())
	require.Len(t, revisions, 1, "failed repair cannot publish a certificate")
	bApp.StructsKeeper.SetInfusion(ctx, original)
	slashCommissionEventValidator(t, bApp, ctx, validator)
	query(2, 0)
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	query(2, 2)
	resets, revisions := commissionEvents(t, ctx.EventManager().ABCIEvents())
	require.Zero(t, resets, "ordinary operation never replaces the global index")
	require.Equal(t, []*types.EventReactorCommissionRevision{
		{ReactorId: id, SlashRevision: 1},
		{ReactorId: id, SlashRevision: 2},
		{ReactorId: id, SlashRevision: 2, ReconciledSlashRevision: 2},
	}, revisions)
	lastAccounting, certification := -1, -1
	for i, event := range ctx.EventManager().ABCIEvents() {
		if event.Type == "structs.structs.EventInfusion" || event.Type == "structs.structs.EventGrid" {
			lastAccounting = i
		}
		if event.Type == "structs.structs.EventReactorCommissionRevision" {
			certification = i
		}
	}
	require.NotEqual(t, -1, lastAccounting)
	require.Greater(t, certification, lastAccounting, "certification follows committed infusion/grid events")
	before := len(ctx.EventManager().Events())
	bApp.StructsKeeper.EnqueueReactorRestart(ctx, id)
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	require.Len(t, ctx.EventManager().Events(), before, "unchanged manual repair emits no duplicate transitions")

	// Malformed private state must be an explicit query failure, not a clean
	// default that tells a client commission is usable.
	ctx.KVStore(bApp.GetKey(types.StoreKey)).Set([]byte("Reactor/commissionRevision/"+id), []byte{1})
	_, err := bApp.StructsKeeper.ReactorStakeShare(ctx, &types.QueryReactorStakeShareRequest{ReactorId: id})
	require.ErrorContains(t, err, "invalid reactor commission revision")
	require.Len(t, ctx.EventManager().Events(), before)
}

func TestCommissionRevisionSnapshotReplacesStateAndRespectsRollback(t *testing.T) {
	bApp, ctx, validator, id := setupCommissionEventApp(t)
	slashCommissionEventValidator(t, bApp, ctx, validator)
	parent := ctx.WithEventManager(sdk.NewEventManager())
	abandonedSlash, _ := parent.CacheContext()
	slashCommissionEventValidator(t, bApp, abandonedSlash, validator)
	_, abandonedEvents := commissionEvents(t, abandonedSlash.EventManager().ABCIEvents())
	require.Len(t, abandonedEvents, 1)
	require.Equal(t, uint64(2), abandonedEvents[0].SlashRevision)
	require.Empty(t, parent.EventManager().Events())
	originalRevision, err := bApp.StructsKeeper.GetReactorCommissionRevision(parent, id)
	require.NoError(t, err)
	require.Equal(t, uint64(1), originalRevision.Issued)
	discarded, _ := parent.CacheContext()
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(discarded)
	_, childEvents := commissionEvents(t, discarded.EventManager().ABCIEvents())
	require.Len(t, childEvents, 1)
	require.Equal(t, uint64(1), childEvents[0].ReconciledSlashRevision)
	require.Empty(t, parent.EventManager().Events())
	response, err := bApp.StructsKeeper.ReactorStakeShare(parent, &types.QueryReactorStakeShareRequest{ReactorId: id})
	require.NoError(t, err)
	require.True(t, response.SlashRevision != response.ReconciledSlashRevision, "discarded repair cannot publish state or events")

	replay := parent.WithBlockHeight(1).WithEventManager(sdk.NewEventManager())
	bApp.StructsKeeper.EventAllGenesis(replay)
	resets, revisions := commissionEvents(t, replay.EventManager().ABCIEvents())
	require.Equal(t, 1, resets)
	require.Equal(t, []*types.EventReactorCommissionRevision{{ReactorId: id, SlashRevision: 1}}, revisions)
	indexed := map[string][2]uint64{"3-obsolete": {99, 0}, id: {9, 9}}
	for _, event := range replay.EventManager().ABCIEvents() {
		switch event.Type {
		case "structs.structs.EventReactorCommissionRevisionReset":
			clear(indexed)
		case "structs.structs.EventReactorCommissionRevision":
			decoded, err := sdk.ParseTypedEvent(event)
			require.NoError(t, err)
			row := decoded.(*types.EventReactorCommissionRevision)
			indexed[row.ReactorId] = [2]uint64{row.SlashRevision, row.ReconciledSlashRevision}
		}
	}
	require.Equal(t, map[string][2]uint64{id: {1, 0}}, indexed)
	later := parent.WithBlockHeight(2).WithEventManager(sdk.NewEventManager())
	bApp.StructsKeeper.EventAllGenesis(later)
	resets, revisions = commissionEvents(t, later.EventManager().ABCIEvents())
	require.Zero(t, resets)
	require.Empty(t, revisions)

	// Snapshot validation is complete before its reset, even if the invalid
	// row follows otherwise valid records in the store.
	store := parent.KVStore(bApp.GetKey(types.StoreKey))
	store.Set([]byte("Reactor/commissionRevision/3-9999999"), []byte{1})
	err = bApp.StructsKeeper.EmitReactorCommissionRevisionSnapshot(later)
	require.Error(t, err)
	require.Empty(t, later.EventManager().Events(), "invalid snapshot cannot clear a consumer's last good state")
}
