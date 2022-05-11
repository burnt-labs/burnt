package keeper

import (
	"github.com/BurntFinance/burnt/x/schedule/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
)

func (k Keeper) EndBlocker(ctx sdk.Context) {
	k.ConsumeScheduledCallsByHeight(ctx, uint64(ctx.BlockHeight()), func(signer sdk.AccAddress, contract sdk.AccAddress, call *types.ScheduledCall) (stop bool) {
		payer := sdk.AccAddress(call.Payer)
		gasCtx := ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
		if !payer.Equals(signer) {
			limits, err := k.determineGasLimit(ctx, payer, signer)
			if err != nil {
				panic(err)
			}
			for _, coin := range limits {
				//if coin.Denom ==
			}
		}
		// construct a message with call info
		msg := []byte{}
		result, err := k.wasmKeeper.Execute(gasCtx, contract, signer, msg, nil)
		if err != nil {
			// error, log and deschedule
		}
		if len(result) != 8 {
			// invalid return from contract, log and deschedule
		}
		nextBlock := sdk.BigEndianToUint64(result)

		// Deduct fees
		// We pass msgs = nil because we are generating this transaction from within our
		k.feegrantKeeper.UseGrantedFees(ctx, payer, signer, gasCtx.GasMeter().GasConsumed(), nil)

		// Schedule the next execution
		k.AddScheduledCall(ctx, signer, contract, call.FunctionName, nextBlock, &payer)

		return false
	})
}

func (k Keeper) determineGasLimit(ctx sdk.Context, granter, grantee sdk.AccAddress) (sdk.Coins, error) {
	allowance, err := k.feegrantKeeper.GetAllowance(ctx, granter, grantee)
	if err != nil {
		return nil, err
	}
	for allowance != nil {
		switch allowance.(type) {
		case *feegrant.BasicAllowance:
			all := allowance.(*feegrant.BasicAllowance)
			return all.GetSpendLimit(), nil
		case *feegrant.AllowedMsgAllowance:
			all := allowance.(*feegrant.AllowedMsgAllowance)
			allowance, err = all.GetAllowance()
			if err != nil {
				return nil, err
			}
		case *feegrant.PeriodicAllowance:
			all := allowance.(*feegrant.PeriodicAllowance)
			periodCoins := all.PeriodCanSpend
			basicCoins := all.Basic.SpendLimit
			return periodCoins.Add(basicCoins...), nil
		default:
			return nil, types.ErrInvalidAllowance
		}
	}

	return nil, types.ErrInvalidAllowance
}
