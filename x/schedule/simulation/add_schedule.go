package simulation

import (
	"math/rand"

	"github.com/BurntFinance/burnt/x/schedule/keeper"
	"github.com/BurntFinance/burnt/x/schedule/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

func SimulateMsgAddSchedule(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgAddSchedule{
			Signer: simAccount.Address.String(),
		}

		// TODO: Handling the AddSchedule simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "AddSchedule simulation not implemented"), nil, nil
	}
}
