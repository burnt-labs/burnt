package keeper

import (
	"encoding/json"
	"github.com/BurntFinance/burnt/x/schedule/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
)

func (k Keeper) executeMsgWithGasLimit(ctx sdk.Context, contract sdk.AccAddress, msg []byte, gasLimit uint64) (gasConsumed uint64, nextBlock uint64, err error) {
	contractGasMeter := sdk.NewGasMeter(gasLimit)
	gasCtx := ctx.WithGasMeter(contractGasMeter)

	// catch out of gas panic and just charge the entire gas limit
	defer func() {
		if r := recover(); r != nil {
			// if it's not an OutOfGas error, raise it again
			if _, ok := r.(sdk.ErrorOutOfGas); !ok {
				// log it to get the original stack trace somewhere (as panic(r) keeps message but stacktrace to here
				k.Logger(ctx).Error("scheduled call throwing panic",
					"error", r)
				panic(r)
			}
			//ctx.GasMeter().ConsumeGas(gasLimit, "Sub-Message OutOfGas panic")
			k.Logger(ctx).Debug("scheduled call hit gas limit",
				"gas consumed", gasCtx.GasMeter().GasConsumed(),
				"gas limit", gasLimit,
				"contract", contract)
			err = sdkerrors.Wrap(sdkerrors.ErrOutOfGas, "scheduled call hit gas limit")
			gasConsumed = gasLimit
			nextBlock = 0
		}
	}()

	result, err := k.wasmPermissionedKeeper.Execute(gasCtx, contract, contract, msg, nil)
	nextBlock = sdk.BigEndianToUint64(result)
	gasConsumed = gasCtx.GasMeter().GasConsumed()

	return
}

func (k Keeper) EndBlocker(ctx sdk.Context) {
	params := k.GetParams(ctx)
	feeReceiver := sdk.AccAddress(params.FeeReceiver)
	k.ConsumeScheduledCallsByHeight(ctx, uint64(ctx.BlockHeight()), func(signer sdk.AccAddress, contract sdk.AccAddress, call *types.ScheduledCall) (stop bool) {
		k.Logger(ctx).Debug("consuming scheduled call",
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
			k.Logger(ctx).Debug("contract is no longer owned by signer",
				"contract", contract,
				"signer", signer)
			return false
		}

		contractBalance := k.bankKeeper.GetBalance(ctx, contract, params.MinimumBalance.Denom)
		if contractBalance.IsLT(params.MinimumBalance) {
			k.Logger(ctx).Debug("contract did not maintain the minimum balance, skipping it",
				"contract", contract,
				"balance", contractBalance,
				"minimum", params.MinimumBalance)
			return false
		}

		gasConsumed, nextBlock, err := k.executeMsgWithGasLimit(ctx, contract, call.CallBody, contractBalance.Amount.Uint64())
		// error gets checked after consuming gas

		gasCoin := sdk.Coin{
			Denom:  params.MinimumBalance.Denom,
			Amount: sdk.NewIntFromUint64(gasConsumed),
		}

		if sendErr := k.bankKeeper.SendCoins(ctx, contract, feeReceiver, sdk.Coins{gasCoin}); sendErr != nil {
			k.Logger(ctx).Error("error sending gas from contract to receiver",
				"contract", contract,
				"gas consumed", gasConsumed,
				"receiver", feeReceiver,
				"call", call.CallBody,
				"error", sendErr)
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

		// check to make sure contract still has minimum balance
		contractBalance = k.bankKeeper.GetBalance(ctx, contract, params.MinimumBalance.Denom)
		if contractBalance.IsLT(params.MinimumBalance) {
			k.Logger(ctx).Debug("contract no longer has the minimum balance, will not schedule it's following scheduled call",
				"contract", contract,
				"balance", contractBalance,
				"minimum", params.MinimumBalance)
			return false
		}

		// Schedule the next execution
		if nextBlock <= uint64(ctx.BlockHeight()) {
			k.Logger(ctx).Debug("contract is trying to schedule a call in the past, skipping it",
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
