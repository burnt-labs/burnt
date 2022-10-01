package types

import (
	"bytes"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spaolacci/murmur3"
)

const (
	// ModuleName defines the module name
	ModuleName = "schedule"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for slashing
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_schedule"
)

const (
	_ = byte(iota)

	// ScheduledCallByBlockHeightKeyPrefix <prefix><block_height><signer><contract><function_hash> -> <function_name, payer>
	ScheduledCallByBlockHeightKeyPrefix
	// ScheduledCallByNameKeyPrefix <prefix><signer><contract><function_hash> -> <block_height>
	ScheduledCallByNameKeyPrefix
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

func stringToHash(s string) []byte {
	h64 := murmur3.New64()
	h64.Write([]byte(s))

	return sdk.Uint64ToBigEndian(h64.Sum64())
}

func MakeScheduledCallByBlockHeightPrefixKey(blockHeight uint64) []byte {
	return bytes.Join([][]byte{{ScheduledCallByBlockHeightKeyPrefix}, sdk.Uint64ToBigEndian(blockHeight)}, []byte{})
}

func MakeScheduledCallByBlockHeightKey(blockHeight uint64, signer sdk.AccAddress, contract sdk.AccAddress) []byte {
	return bytes.Join([][]byte{MakeScheduledCallByBlockHeightPrefixKey(blockHeight), signer.Bytes(), contract.Bytes()}, []byte{})
}

func MakeScheduledCallByNameKey(signer sdk.AccAddress, contract sdk.AccAddress) []byte {
	return bytes.Join([][]byte{{ScheduledCallByNameKeyPrefix}, signer.Bytes(), contract.Bytes()}, []byte{})
}
