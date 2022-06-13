package keeper

import (
	"encoding/json"
	"github.com/BurntFinance/burnt/x/schedule/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
)

func (k Keeper) EndBlocker(ctx sdk.Context) {
	gasDenom := k.GetParams(ctx).GasDenom
	// k.bankKeeper.GetDenomMetaData(ctx, gasDenom) // is this needed?
	k.ConsumeScheduledCallsByHeight(ctx, uint64(ctx.BlockHeight()), func(signer sdk.AccAddress, contract sdk.AccAddress, call *types.ScheduledCall) (stop bool) {
		payer := sdk.AccAddress(call.Payer)
		gasCtx := ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
		if !payer.Equals(signer) {
			limits, err := k.determineGasLimit(ctx, payer, signer)
			if err != nil {
				panic(err)
			}
			for _, coin := range limits {
				if coin.Denom == gasDenom {
					// todo: check for minimal available funds here
				}
			}
		}

		// todo: construct a message with call info
		var msg []byte
		msg, err := json.Marshal(map[string]interface{}{
			call.FunctionName: map[string]interface{}{},
		})
		if err != nil {
			k.Logger(ctx).Error("unable to marshal wasm call with function %s", call.FunctionName)
		}

		result, err := k.wasmKeeper.Execute(gasCtx, contract, signer, msg, nil)
		if err != nil {
			k.Logger(ctx).Error("error executing scheduled wasm call",
				"block height", ctx.BlockHeight(),
				"signer", signer,
				"contract", contract,
				"msg", msg,
				"error", err,
			)
			return false
		}
		if len(result) != 8 {
			k.Logger(ctx).Info("invalid response from contract",
				"result", result,
				"msg", msg,
				"contract", contract,
				"payer", payer,
				"signer", signer,
			)
		}
		nextBlock := sdk.BigEndianToUint64(result)

		// Deduct fees
		// We pass msgs = nil because we are generating this transaction
		gasConsumed := gasCtx.GasMeter().GasConsumed()
		gasCoins := sdk.Coins{{
			Denom:  gasDenom,
			Amount: sdk.NewIntFromUint64(gasConsumed),
		}}
		if err := k.feegrantKeeper.UseGrantedFees(ctx, payer, signer, gasCoins, nil); err != nil {
			k.Logger(ctx).Error("error using granted fees",
				"error", err,
				"payer", payer,
				"signer", signer,
				"gas coins", gasCoins,
			)
			return false
		}

		// Schedule the next execution
		if nextBlock <= uint64(ctx.BlockHeight()) {
			return true
		}
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
