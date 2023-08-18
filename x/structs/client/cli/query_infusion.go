package cli

import (
	"context"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"structs/x/structs/types"
)

func CmdListInfusion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-infusion",
		Short: "list all infusion",
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

			params := &types.QueryAllInfusionRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.InfusionAll(context.Background(), params)
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

func CmdShowInfusion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-infusion [destination type] [destination id] [address]",
		Short: "shows a infusion",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

            destinationType := args[0]

			destinationId, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

            address := args[2]

			params := &types.QueryGetInfusionRequest{
				DestinationType: destinationType,
				DestinationId: destinationId,
				Address: address,
			}

			res, err := queryClient.Infusion(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
