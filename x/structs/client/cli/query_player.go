package cli

import (

    "context"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/cast"
	"structs/x/structs/types"
)

func CmdListPlayer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-player",
		Short: "list all player",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllPlayerRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.PlayerAll(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddPaginationFlagsToCmd(cmd, cmd.Use)
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdShowPlayer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-player",
		Short: "shows my player",
		Args:  cobra.RangeArgs(0,1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

            var params *types.QueryGetPlayerRequest

			queryClient := types.NewQueryClient(clientCtx)

			argAddress, err := cmd.Flags().GetString("address")
            if err != nil {
                return err
            }

            if (argAddress != "") {
                addressLookupRequest := &types.QueryGetAddressRequest{
                    Address: argAddress,
                }

                addressResults, AddressLookupErr := queryClient.Address(context.Background(), addressLookupRequest)
                if AddressLookupErr != nil {
                    return AddressLookupErr
                }

                params = &types.QueryGetPlayerRequest{
                    Id: addressResults.PlayerId,
                }

            } else {
                playerId, err := cmd.Flags().GetUint64("player-id")
                if err != nil {
                    return err
                }

                // Backwards compatibility
                if ((playerId == 0) && (len(args) == 1)) {
                    playerId, err = cast.ToUint64E(args[0])
                    if err != nil {
                        return err
                    }
                }

                params = &types.QueryGetPlayerRequest{
                    Id: playerId,
                }
            }



			res, err := queryClient.Player(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	cmd.Flags().Uint64P("player-id", "I", 0, "Lookup by Player ID")
	cmd.Flags().StringP("address", "A", "", "Lookup by associated Player Address")

	return cmd
}
