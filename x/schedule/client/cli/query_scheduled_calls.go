package cli

import (
	"context"
	"github.com/burnt-labs/burnt/x/schedule/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

func CmdQueryScheduledCalls() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scheduled-calls",
		Short: "returns all scheduled calls",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.ScheduledCalls(context.Background(), &types.QueryScheduledCallsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
