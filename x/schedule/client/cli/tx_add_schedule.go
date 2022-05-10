package cli

import (
	"strconv"

	"github.com/BurntFinance/burnt/x/schedule/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdAddSchedule() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-schedule [contract] [function-name] [payer]",
		Short: "Broadcast message add_schedule",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argContract, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}
			argFunctionName := args[1]
			argPayer, err := sdk.AccAddressFromBech32(args[2])
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgAddSchedule(
				clientCtx.GetFromAddress(),
				argContract,
				argFunctionName,
				argPayer,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}