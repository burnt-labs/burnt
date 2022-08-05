package keeper

import (
	"encoding/json"
	"github.com/BurntFinance/burnt/x/schedule/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
)

func (k Keeper) EndBlocker(ctx sdk.Context) {
	callCount := k.countOfScheduledCallsAtHeight(ctx, uint64(ctx.BlockHeight()))
	k.Logger(ctx).Info("iterating scheduled calls",
		"height", ctx.BlockHeight(),
		"count", callCount)

	params := k.GetParams(ctx)
	feeReceiver := sdk.AccAddress(params.FeeReceiver)
	k.ConsumeScheduledCallsByHeight(ctx, uint64(ctx.BlockHeight()), func(signer sdk.AccAddress, contract sdk.AccAddress, call *types.ScheduledCall) (stop bool) {
		k.Logger(ctx).Info("consuming scheduled call",
			"signer", signer,
			"contract", contract,
			"call", call)

		// verify the signer is still the owner
		ownerQueryMsg, err := json.Marshal(map[string]interface{}{
			"is_owner": map[string]interface{}{
				"address": signer,
			},
		})
		ownerQueryRes, err := k.wasmViewKeeper.QuerySmart(ctx, contract, ownerQueryMsg)
		if err != nil {
			k.Logger(ctx).Error("error querying smart contract for owner",
				"error", err)
			return false
		}
		var isOwner isOwnerResponse
		err = json.Unmarshal(ownerQueryRes, &isOwner)
		if err != nil {
			k.Logger(ctx).Error("error parsing owner response from contract",
				"error", err)
			return false
		}
		if !isOwner.IsOwner {
			k.Logger(ctx).Info("contract is no longer owned by signer",
				"contract", contract,
				"signer", signer)
			return false
		}

		contractBalance := k.bankKeeper.GetBalance(ctx, contract, params.MinimumBalance.Denom)
		if contractBalance.IsLT(params.MinimumBalance) {
			k.Logger(ctx).Info("contract did not maintain the minimum balance, skipping it",
				"contract", contract,
				"balance", contractBalance,
				"minimum", params.MinimumBalance)
			return false
		}

		contractGasMeter := sdk.NewGasMeter(contractBalance.Amount.Uint64())
		gasCtx := ctx.WithGasMeter(contractGasMeter)
		result, err := k.wasmPermissionedKeeper.Execute(gasCtx, contract, contract, call.CallBody, nil)
		if err != nil {
			k.Logger(ctx).
		}
		// error gets checked after consuming gas

		gasConsumed := gasCtx.GasMeter().GasConsumed()
		gasCoin := sdk.Coin{
			Denom:  params.MinimumBalance.Denom,
			Amount: sdk.NewIntFromUint64(gasConsumed),
		}

		// always charge the contract for the gas, either its gas used or total remaining balance, whichever is less
		if gasCoin.IsLT(contractBalance) {
			if sendErr := k.bankKeeper.SendCoins(gasCtx, contract, feeReceiver, sdk.Coins{gasCoin}); err != nil {
				k.Logger(ctx).Error("error sending gas from contract to receiver",
					"contract", contract,
					"receiver", feeReceiver,
					"call", call.CallBody,
					"error", sendErr)
			}
		} else {
			k.Logger(ctx).Info("contract did not have a balance to pay for used gas, collecting all",
				"contract", contract,
				"gas used", gasCoin,
				"contract balance", contractBalance)
			if sendErr := k.bankKeeper.SendCoins(gasCtx, contract, feeReceiver, sdk.Coins{contractBalance}); err != nil {
				k.Logger(ctx).Error("error sending gas from contract to receiver",
					"contract", contract,
					"receiver", feeReceiver,
					"call", call.CallBody,
					"error", sendErr)
			}
		}

		// continue checking if call errored
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
			return false
		}
		nextBlock := sdk.BigEndianToUint64(result)

		// check to make sure contract still has minimum balance
		contractBalance = k.bankKeeper.GetBalance(ctx, contract, params.MinimumBalance.Denom)
		if contractBalance.IsLT(params.MinimumBalance) {
			k.Logger(ctx).Info("contract no longer has the minimum balance, not scheduling it's following scheduled call",
				"contract", contract,
				"balance", contractBalance,
				"minimum", params.MinimumBalance)
			return false
		}

		// Schedule the next execution
		if nextBlock <= uint64(ctx.BlockHeight()) {
			k.Logger(ctx).Info("contract is trying to schedule a call in the past, skipping it",
				"contract", contract,
				"next block", nextBlock,
				"current block", ctx.BlockHeight())
			return false
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
