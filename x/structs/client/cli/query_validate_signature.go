package cli

import (

    //"context"
	"github.com/cosmos/cosmos-sdk/client"
	//"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"structs/x/structs/types"

)
/*
message QueryValidateSignatureRequest {
  string address        = 1;
  string message        = 2;
  string proofPubKey    = 3;
  string proofSignature = 4;
}

message QueryValidateSignatureResponse {
  bool pubkeyFormatError      = 1;
  bool signatureFormatError   = 2;
  bool addressPubkeyMismatch  = 3;
  bool signatureInvalid       = 4;
  bool valid                  = 5;
}
*/
func ValidateSignature() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate-signature [address] [message] [signature] [pubkey]",
		Short: "Validate the signature and source of a generic message",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

            params := types.QueryValidateSignatureRequest{
                  Address:          args[0],
                  Message:          args[1],
                  ProofPubKey:      args[2],
                  ProofSignature:   args[3],
            }

			res, err := queryClient.ValidateSignature(cmd.Context(), &params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	return cmd
}
