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
		cdc                    codec.BinaryCodec
		storeKey               sdk.StoreKey
		memKey                 sdk.StoreKey
		paramstore             paramtypes.Subspace
		wasmViewKeeper         types.WasmViewKeeper
		wasmPermissionedKeeper types.WasmPermissionedKeeper
		feegrantKeeper         types.FeeGrantKeeper
		bankKeeper             types.BankKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey sdk.StoreKey,
	ps paramtypes.Subspace,
	wasmViewKeeper types.WasmViewKeeper,
	wasmPermissionedKeeper types.WasmPermissionedKeeper,
	feegrantKeeper types.FeeGrantKeeper,
	bankKeeper types.BankKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{
		cdc:                    cdc,
		storeKey:               storeKey,
		memKey:                 memKey,
		paramstore:             ps,
		wasmViewKeeper:         wasmViewKeeper,
		wasmPermissionedKeeper: wasmPermissionedKeeper,
		feegrantKeeper:         feegrantKeeper,
		bankKeeper:             bankKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// Scheduled Calls

func (k Keeper) BlockHeightForSignerContract(ctx sdk.Context, signer sdk.AccAddress, contract sdk.AccAddress) uint64 {
	store := ctx.KVStore(k.storeKey)
	bySignerContractKey := types.MakeScheduledCallBySignerContractKey(signer, contract)

	return sdk.BigEndianToUint64(store.Get(bySignerContractKey))
}

func (k Keeper) AddScheduledCall(ctx sdk.Context, signer sdk.AccAddress, contract sdk.AccAddress, callBody []byte, blockHeight uint64) {
	store := ctx.KVStore(k.storeKey)
	byHeightKey := types.MakeScheduledCallByBlockHeightKey(blockHeight, signer, contract)
	bySignerContractKey := types.MakeScheduledCallBySignerContractKey(signer, contract)

	value := &types.ScheduledCall{
		CallBody: callBody,
	}

	store.Set(bySignerContractKey, sdk.Uint64ToBigEndian(blockHeight))
	store.Set(byHeightKey, k.cdc.MustMarshal(value))
}

func (k Keeper) ReScheduleCall(ctx sdk.Context, signer sdk.AccAddress, contract sdk.AccAddress, callBody []byte, oldBlockHeight uint64, newBlockHeight uint64) {
	store := ctx.KVStore(k.storeKey)

	k.removeScheduledCallWithBlockHeight(ctx, signer, contract, oldBlockHeight)

	newByHeightKey := types.MakeScheduledCallByBlockHeightKey(newBlockHeight, signer, contract)
	newBySignerContractKey := types.MakeScheduledCallBySignerContractKey(signer, contract)

	value := &types.ScheduledCall{
		CallBody: callBody,
	}

	store.Set(newByHeightKey, sdk.Uint64ToBigEndian(newBlockHeight))
	store.Set(newBySignerContractKey, k.cdc.MustMarshal(value))
}

func (k Keeper) RemoveScheduledCall(ctx sdk.Context, signer sdk.AccAddress, contract sdk.AccAddress) {
	store := ctx.KVStore(k.storeKey)
	byNameKey := types.MakeScheduledCallBySignerContractKey(signer, contract)

	blockHeight := sdk.BigEndianToUint64(store.Get(byNameKey))
	store.Delete(byNameKey)

	byHeightKey := types.MakeScheduledCallByBlockHeightKey(blockHeight, signer, contract)
	store.Delete(byHeightKey)
}

func (k Keeper) removeScheduledCallWithBlockHeight(ctx sdk.Context, signer sdk.AccAddress, contract sdk.AccAddress, blockHeight uint64) {
	store := ctx.KVStore(k.storeKey)
	byNameKey := types.MakeScheduledCallBySignerContractKey(signer, contract)
	store.Delete(byNameKey)

	byHeightKey := types.MakeScheduledCallByBlockHeightKey(blockHeight, signer, contract)
	store.Delete(byHeightKey)
}

func (k Keeper) iterateScheduledCalls(ctx sdk.Context, cb func(height uint64, signer sdk.AccAddress, contract sdk.AccAddress, call *types.ScheduledCall) (stop bool)) {
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{types.ScheduledCallByBlockHeightKeyPrefix})
	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		keyPair := bytes.NewBuffer(bytes.TrimPrefix(iter.Key(), []byte{types.ScheduledCallByBlockHeightKeyPrefix}))
		blockHeight := sdk.BigEndianToUint64(keyPair.Next(8))
		signer := sdk.AccAddress(keyPair.Next(20))
		contract := sdk.AccAddress(keyPair.Next(32))
		var call types.ScheduledCall
		k.cdc.MustUnmarshal(iter.Value(), &call)
		if cb(blockHeight, signer, contract, &call) {
			break
		}
	}
}

func (k Keeper) ConsumeScheduledCallsByHeight(ctx sdk.Context, blockHeight uint64, cb func(signer sdk.AccAddress, contract sdk.AccAddress, call *types.ScheduledCall) (stop bool)) {
	prefixKey := types.MakeScheduledCallByBlockHeightPrefixKey(blockHeight)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixKey)
	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		keyPair := bytes.NewBuffer(bytes.TrimPrefix(iter.Key(), types.MakeScheduledCallByBlockHeightPrefixKey(blockHeight)))
		signer := sdk.AccAddress(keyPair.Next(20))
		contract := sdk.AccAddress(keyPair.Next(32))

		var call types.ScheduledCall
		k.cdc.MustUnmarshal(iter.Value(), &call)
		k.removeScheduledCallWithBlockHeight(ctx, signer, contract, blockHeight)
		if cb(signer, contract, &call) {
			break
		}
	}
}

func (k Keeper) countOfScheduledCallsAtHeight(ctx sdk.Context, blockHeight uint64) (count uint64) {
	prefixKey := types.MakeScheduledCallByBlockHeightPrefixKey(blockHeight)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixKey)
	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()
	count = 0
	for ; iter.Valid(); iter.Next() {
		count += 1
	}
	return
}
