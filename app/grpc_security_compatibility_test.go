package app_test

import (
	"context"
	"net"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	structstypes "structs/x/structs/types"
)

// Exercise the real SDK query routing and gogo codec over HTTP/2 after a gRPC
// dependency bump. bufconn avoids stopping or sharing ports with a local node.
func TestGRPCSecurityUpdateSDKCompatibility(t *testing.T) {
	bApp, sdkCtx := setupJailGateApp(t)
	_, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: sdkCtx.BlockTime()})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.ForceServerCodec(codec.NewProtoCodec(bApp.AppCodec().InterfaceRegistry()).GRPCCodec()))
	bApp.RegisterGRPCServer(server)
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close(); <-done })
	conn, err := grpc.NewClient("passthrough:///structs-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	bank, err := banktypes.NewQueryClient(conn).Params(ctx, &banktypes.QueryParamsRequest{})
	require.NoError(t, err)
	require.NotNil(t, bank.Params)
	query := structstypes.NewQueryClient(conn)
	var headers metadata.MD
	catalog, err := query.StructTypeAll(ctx, &structstypes.QueryAllStructTypeRequest{}, grpc.Header(&headers))
	require.NoError(t, err)
	require.Len(t, catalog.StructType, 22)
	require.Equal(t, []string{"1"}, headers.Get("x-cosmos-block-height"))
	for _, kind := range catalog.StructType {
		require.EqualValues(t, 1, kind.BuildCharge)
	}
	badCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("x-cosmos-block-height", "-1"))
	_, err = query.StructTypeAll(badCtx, &structstypes.QueryAllStructTypeRequest{})
	require.Error(t, err)
	_, err = query.StructTypeAll(ctx, &structstypes.QueryAllStructTypeRequest{})
	require.NoError(t, err, "server must remain usable after a rejected request")
}
