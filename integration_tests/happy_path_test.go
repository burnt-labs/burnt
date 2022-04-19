package integration_tests

import (
	"encoding/json"
	"fmt"
	wasmutils "github.com/CosmWasm/wasmd/x/wasm/client/utils"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/types"
	"io/ioutil"
	"strconv"
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

func instantiateContract(codeId uint64, label string, sender types.AccAddress) (wasmtypes.MsgInstantiateContract, error) {
	msg, err := json.Marshal(map[string]interface{}{
		"minter": sender.String(),
		"name":   label,
		"symbol": "skronk",
	})
	if err != nil {
		return wasmtypes.MsgInstantiateContract{}, nil
	}
	return wasmtypes.MsgInstantiateContract{
		Sender: sender.String(),
		Admin:  sender.String(),
		CodeID: codeId,
		Label:  label,
		Msg:    msg,
		Funds:  nil,
	}, nil
}

func mintTokenMsg(contract string, sender types.AccAddress, id string, recipient types.AccAddress) (wasmtypes.MsgExecuteContract, error) {
	msg, err := json.Marshal(map[string]interface{}{
		"mint": map[string]interface{}{
			"owner":    recipient.String(),
			"token_id": id,
		},
	})
	if err != nil {
		return wasmtypes.MsgExecuteContract{}, nil
	}
	return wasmtypes.MsgExecuteContract{
		Sender:   sender.String(),
		Contract: contract,
		Msg:      msg,
		Funds:    nil,
	}, nil
}

func (s *IntegrationTestSuite) TestHappyPath() {
	s.Run("Bring up chain, and test the happy path", func() {
		msg, err := storeCode("contracts/compiled/cw721_metadata_onchain.wasm", s.chain.validators[0].keyInfo.GetAddress())
		s.Require().NoError(err)

		val := s.chain.validators[0]
		keyring, err := val.keyring()
		s.Require().NoError(err)

		clientCtx, err := s.chain.clientContext("tcp://localhost:26657", &keyring, "val", val.keyInfo.GetAddress())
		s.Require().NoError(err)

		res, err := s.chain.sendMsgs(*clientCtx, &msg)
		s.Require().NoError(err)
		s.Require().Zero(res.Code)

		events := res.Logs[0].Events
		event := events[len(events)-1]
		attrsLen := len(event.Attributes)
		attr := event.Attributes[attrsLen-1]
		s.Require().Equal("code_id", attr.Key)

		codeIdStr := attr.Value
		codeId, err := strconv.Atoi(codeIdStr)
		s.Require().NoError(err)

		instantiateMsg, err := instantiateContract(uint64(codeId), "Skronk Token Internazionale", val.keyInfo.GetAddress())
		res, err = s.chain.sendMsgs(*clientCtx, &instantiateMsg)
		s.Require().NoError(err)
		s.Require().Zero(res.Code)
		event = res.Logs[0].Events[0]
		s.Require().NotNil(event)
		attr = event.Attributes[0]
		s.Require().Equal("_contract_address", attr.Key)
		contractAddress := attr.Value
		s.T().Logf("Contract address: %s", contractAddress)

		for i, otherVal := range s.chain.validators {
			id := fmt.Sprintf("badonk-%d", i)
			mintMsg, err := mintTokenMsg(contractAddress, val.keyInfo.GetAddress(), id, otherVal.keyInfo.GetAddress())
			res, err = s.chain.sendMsgs(*clientCtx, &mintMsg)
			s.Require().NoError(err)
			s.Require().Zero(res.Code)
			s.T().Log(res.RawLog)
		}
		// TODO(@bigs, @ash)
		// 3. Send one token to each of validators 1-3
		// 4. Query contract state to confirm balance of all validators and that tokens are correct
	})
}
