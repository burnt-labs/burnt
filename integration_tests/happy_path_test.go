package integration_tests

import (
	"fmt"
	wasmutils "github.com/CosmWasm/wasmd/x/wasm/client/utils"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/types"
	"io/ioutil"
)

func storeCode(path string, sender types.AccAddress) (wasmtypes.MsgStoreCode, error) {
	wasm, err := ioutil.ReadFile(path)
	if err != nil {
		return wasmtypes.MsgStoreCode{}, err
	}
	if wasmutils.IsWasm(wasm) {
		wasm, err = wasmutils.GzipIt(wasm)
		if err != nil {
			return wasmtypes.MsgStoreCode{}, err
		}
	} else if !wasmutils.IsGzip(wasm) {
		return wasmtypes.MsgStoreCode{}, fmt.Errorf("invalid input file. Use wasm binary or gzip")
	}

	return wasmtypes.MsgStoreCode{
		Sender:                sender.String(),
		WASMByteCode:          wasm,
		InstantiatePermission: &wasmtypes.AllowEverybody,
	}, nil
}

func (s *IntegrationTestSuite) TestHappyPath() {
	s.Run("Bring up chain, and test the happy path", func() {
		msg, err := storeCode("../contracts/compiled/cw721_metadata_onchain.wasm", s.chain.validators[0].keyInfo.GetAddress())
		s.Require().NoError(err)
		val := s.chain.validators[0]
		keyring, err := val.keyring()
		s.Require().NoError(err)
		clientCtx, err := s.chain.clientContext("tcp://localhost:26657", &keyring, "val", val.keyInfo.GetAddress())
		s.Require().NoError(err)
		_, err = s.chain.sendMsgs(*clientCtx, &msg)
		s.Require().NoError(err)
	})
}
