package keeper

import (
	"encoding/json"
	"github.com/BurntFinance/burnt/x/schedule/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	"strconv"
)

func (k Keeper) EndBlocker(ctx sdk.Context) {
	params := k.GetParams(ctx)
	feeReceiver := sdk.AccAddress(params.FeeReceiver)
	k.ConsumeScheduledCallsByHeight(ctx, uint64(ctx.BlockHeight()), func(signer sdk.AccAddress, contract sdk.AccAddress, call *types.ScheduledCall) (stop bool) {

		// verify the signer is still the owner
		ownerQueryMsg, err := json.Marshal(map[string]interface{}{
			"is_owner": signer.Bytes(),
		})
		ownerQueryRes, err := k.wasmViewKeeper.QuerySmart(ctx, contract, ownerQueryMsg)
		if err != nil {
			return false
		}
		isOwner, err := strconv.ParseBool(string(ownerQueryRes))
		if err != nil {
			return false
		}
		if !isOwner {
			return false
		}

		contractBalance := k.bankKeeper.GetBalance(ctx, contract, params.GasDenom)
		contractGasMeter := sdk.NewGasMeter(contractBalance.Amount.Uint64())
		gasCtx := ctx.WithGasMeter(contractGasMeter)
		result, err := k.wasmPermissionedKeeper.Execute(gasCtx, contract, contract, call.CallBody, nil)

		// always charge the contract for the gas
		gasConsumed := gasCtx.GasMeter().GasConsumed()
		gasCoins := sdk.Coins{{
			Denom:  params.GasDenom,
			Amount: sdk.NewIntFromUint64(gasConsumed),
		}}
		if sendErr := k.bankKeeper.SendCoins(gasCtx, contract, feeReceiver, gasCoins); err != nil {
			k.Logger(ctx).Error("error sending gas from contract to receiver",
				"contract", contract,
				"receiver", feeReceiver,
				"call", call.CallBody,
				"error", sendErr)
		}

		if err != nil {
			k.Logger(ctx).Error("error executing scheduled wasm call",
				"block height", ctx.BlockHeight(),
				"signer", signer,
				"contract", contract,
				"msg", call.CallBody,
				"error", err,
			)
			return false
		}
		if len(result) != 8 {
			k.Logger(ctx).Info("invalid response from contract",
				"result", result,
				"msg", call.CallBody,
				"contract", contract,
				"signer", signer,
			)
		}
		nextBlock := sdk.BigEndianToUint64(result)

		// Deduct fees
		// We pass msgs = nil because we are generating this transaction
		//gasConsumed := gasCtx.GasMeter().GasConsumed()
		//gasCoins := sdk.Coins{{
		//	Denom:  gasDenom,
		//	Amount: sdk.NewIntFromUint64(gasConsumed),
		//}}

		//if err := k.feegrantKeeper.UseGrantedFees(ctx, payer, signer, gasCoins, nil); err != nil {
		//	k.Logger(ctx).Error("error using granted fees",
		//		"error", err,
		//		"payer", payer,
		//		"signer", signer,
		//		"gas coins", gasCoins,
		//	)
		//	return false
		//}

		// Schedule the next execution
		if nextBlock <= uint64(ctx.BlockHeight()) {
			return true
		}
		k.AddScheduledCall(ctx, signer, contract, call.CallBody, nextBlock)

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
			return nil, types.ErrUnmetMinimumBalance
		}
	}

	return nil, types.ErrUnmetMinimumBalance
}
