package keeper

import (
	"bytes"
	"fmt"
	"github.com/cosmos/cosmos-sdk/store/prefix"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/BurntFinance/burnt/x/schedule/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

type (
	Keeper struct {
		cdc            codec.BinaryCodec
		storeKey       sdk.StoreKey
		memKey         sdk.StoreKey
		paramstore     paramtypes.Subspace
		wasmKeeper     types.WasmKeeper
		feegrantKeeper types.FeeGrantKeeper
		bankKeeper     types.BankKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey sdk.StoreKey,
	ps paramtypes.Subspace,
	wasmKeeper types.WasmKeeper,
	feegrantKeeper types.FeeGrantKeeper,
	bankKeeper types.BankKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{
		cdc:            cdc,
		storeKey:       storeKey,
		memKey:         memKey,
		paramstore:     ps,
		wasmKeeper:     wasmKeeper,
		feegrantKeeper: feegrantKeeper,
		bankKeeper:     bankKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// Scheduled Calls

func (k Keeper) AddScheduledCall(ctx sdk.Context, signer sdk.AccAddress, contract sdk.AccAddress, functionName string, blockHeight uint64, payer *sdk.AccAddress) {
	store := ctx.KVStore(k.storeKey)
	byHeightKey := types.MakeScheduledCallByBlockHeightKey(blockHeight, signer, contract, functionName)
	byNameKey := types.MakeScheduledCallByNameKey(signer, contract, functionName)

	value := &types.ScheduledCall{
		FunctionName: functionName,
		Payer:        signer.Bytes(),
	}

	if payer != nil {
		value.Payer = payer.Bytes()
	}

	// todo: check for funds in payer account

	store.Set(byNameKey, sdk.Uint64ToBigEndian(blockHeight))
	store.Set(byHeightKey, k.cdc.MustMarshal(value))
}

func (k Keeper) RemoveScheduledCall(ctx sdk.Context, signer sdk.AccAddress, contract sdk.AccAddress, functionName string) {
	store := ctx.KVStore(k.storeKey)
	byNameKey := types.MakeScheduledCallByNameKey(signer, contract, functionName)

	blockHeight := sdk.BigEndianToUint64(store.Get(byNameKey))
	store.Delete(byNameKey)

	byHeightKey := types.MakeScheduledCallByBlockHeightKey(blockHeight, signer, contract, functionName)
	store.Delete(byHeightKey)
}

func (k Keeper) RemoveScheduledCallWithBlockHeight(ctx sdk.Context, signer sdk.AccAddress, contract sdk.AccAddress, functionName string, blockHeight uint64) {
	store := ctx.KVStore(k.storeKey)
	byNameKey := types.MakeScheduledCallByNameKey(signer, contract, functionName)
	store.Delete(byNameKey)

	byHeightKey := types.MakeScheduledCallByBlockHeightKey(blockHeight, signer, contract, functionName)
	store.Delete(byHeightKey)
}

func (k Keeper) ConsumeScheduledCallsByHeight(ctx sdk.Context, blockHeight uint64, cb func(signer sdk.AccAddress, contract sdk.AccAddress, call *types.ScheduledCall) (stop bool)) {
	prefixKey := types.MakeScheduledCallByBlockHeightPrefixKey(blockHeight)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixKey)
	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		keyPair := bytes.NewBuffer(bytes.TrimPrefix(iter.Key(), types.MakeScheduledCallByBlockHeightPrefixKey(blockHeight)))
		signer := sdk.AccAddress(keyPair.Next(20))
		contract := sdk.AccAddress(keyPair.Next(20))

		var call types.ScheduledCall
		k.cdc.MustUnmarshal(iter.Value(), &call)
		k.RemoveScheduledCallWithBlockHeight(ctx, signer, contract, call.FunctionName, blockHeight)
		if cb(signer, contract, &call) {
			break
		}
	}
}
