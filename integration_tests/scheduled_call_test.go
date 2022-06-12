package integration_tests

import (
	"context"
	"encoding/json"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types"
	"strconv"
)

func instantiateTickerContract(codeId uint64, label string, sender types.AccAddress, count int32) (wasmtypes.MsgInstantiateContract, error) {
	msg, err := json.Marshal(map[string]interface{}{
		"count": count,
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

func incrementMsg(contract string, sender types.AccAddress) (wasmtypes.MsgExecuteContract, error) {
	msg, err := json.Marshal(map[string]interface{}{
		"increment": map[string]interface{}{},
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

type CountResponse struct {
	Count int32 `json:"count"`
}

func queryCount(ctx *client.Context, addr string) (int, error) {
	queryClient := wasmtypes.NewQueryClient(ctx)

	queryData, err := json.Marshal(map[string]interface{}{
		"get_count": map[string]interface{}{},
	})
	if err != nil {
		return 0, err
	}
	contractQueryData := wasmtypes.QuerySmartContractStateRequest{
		Address:   addr,
		QueryData: queryData,
	}
	qres, err := queryClient.SmartContractState(context.Background(), &contractQueryData)
	if err != nil {
		return 0, err
	}
	var countResponse CountResponse
	err = json.Unmarshal(qres.Data, &countResponse)
	if err != nil {
		return 0, err
	}

	return int(countResponse.Count), nil
}

const StartCount = 1337

func (s *IntegrationTestSuite) TestScheduledCall() {
	s.Run("Bring up chain, and test the schedule module", func() {
		msg, err := storeCode("contracts/compiled/ticker.wasm", s.chain.validators[0].keyInfo.GetAddress())
		s.Require().NoError(err)

		val := s.chain.validators[0]
		keyring, err := val.keyring()
		s.Require().NoError(err)

		clientCtx, err := s.chain.clientContext("tcp://localhost:26657", &keyring, "val", val.keyInfo.GetAddress())
		s.Require().NoError(err)

		s.T().Log("Uploading ticker contract...")
		res, err := s.chain.sendMsgs(*clientCtx, &msg)
		s.Require().NoError(err)
		s.Require().Zero(res.Code)
		s.T().Log("ticker contract uploaded successfully")

		events := res.Logs[0].Events
		event := events[len(events)-1]
		attrsLen := len(event.Attributes)
		attr := event.Attributes[attrsLen-1]
		s.Require().Equal("code_id", attr.Key)

		codeIdStr := attr.Value
		codeId, err := strconv.Atoi(codeIdStr)
		s.Require().NoError(err)
		s.T().Logf("Found code ID %d for ticker contract", codeId)

		s.T().Log("Instantiating ticker contract...")
		instantiateMsg, err := instantiateTickerContract(uint64(codeId), "test ticker", val.keyInfo.GetAddress(), StartCount)
		s.Require().NoError(err)
		res, err = s.chain.sendMsgs(*clientCtx, &instantiateMsg)
		s.Require().NoError(err)
		s.Require().Zero(res.Code)
		event = res.Logs[0].Events[0]
		s.Require().NotNil(event)
		attr = event.Attributes[0]
		s.Require().Equal("_contract_address", attr.Key)
		contractAddress := attr.Value
		s.T().Logf("ticker contract instantiated at address: %s", contractAddress)

		// baseline tests to make sure the contract behaves as expected
		s.T().Log("Querying contract for count")
		count, err := queryCount(clientCtx, contractAddress)
		s.Require().NoError(err)
		s.Require().Equal(StartCount, count)

		incrementMsg, err := incrementMsg(contractAddress, val.keyInfo.GetAddress())
		s.Require().NoError(err)
		res, err = s.chain.sendMsgs(*clientCtx, &incrementMsg)
		s.Require().NoError(err)
		s.Require().Zero(res.Code)

		count, err = queryCount(clientCtx, contractAddress)
		s.Require().NoError(err)
		s.Require().Equal(StartCount+1, count)
	})
}
