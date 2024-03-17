package cli

import (

    "context"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"structs/x/structs/types"

)

func CmdPlayerMe() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "player-me",
		Short: "shows a specific player",
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
                playerId, err := cmd.Flags().GetString("player-id")
                if err != nil {
                    return err
                }

                params = &types.QueryGetPlayerRequest{
                    Id: playerId,
                }
                // Backwards compatibility
                if (playerId == "") {

                    if (len(args) == 1) {
                        playerId = args[0]

                       params = &types.QueryGetPlayerRequest{
                            Id: playerId,
                        }
                    } else {
                        kb := clientCtx.Keyring

                        info, err := kb.List()
                        if err != nil {
                            return err
                        }

                        // Change this entire query to return multiple players
                        for index, _ := range info {
                            keychainAddress, _ := info[index].GetAddress()
                            argAddress = keychainAddress.String()

                            addressLookupRequest := &types.QueryGetAddressRequest{ Address: argAddress,}

                            addressResults, _ := queryClient.Address(context.Background(), addressLookupRequest)

                            if (addressResults.PlayerId != "") {
                                params = &types.QueryGetPlayerRequest{ Id: addressResults.PlayerId, }
                                break;
                            }

                        }

                    }

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

	cmd.Flags().StringP("player-id", "I", "", "Lookup by Player ID")
	cmd.Flags().StringP("address", "A", "", "Lookup by associated Player Address")

	return cmd
}
