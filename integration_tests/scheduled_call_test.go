package integration_tests

import (
	"context"
	"encoding/json"
	scheduletypes "github.com/BurntFinance/burnt/x/schedule/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types"
	"strconv"
	"time"
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

func currentBlockHeight(clientCtx *client.Context) (uint64, error) {
	node, err := clientCtx.GetNode()
	if err != nil {
		return 0, err
	}
	status, err := node.Status(context.Background())
	if err != nil {
		return 0, err
	}
	return uint64(status.SyncInfo.LatestBlockHeight), nil
}

func queryScheduledCalls(clientCtx *client.Context) ([]*scheduletypes.QueryScheduledCall, error) {
	queryClient := scheduletypes.NewQueryClient(clientCtx)
	res, err := queryClient.ScheduledCalls(context.Background(), &scheduletypes.QueryScheduledCallsRequest{})
	if err != nil {
		return nil, err
	}

	return res.Calls, nil
}

const StartCount = 1337

var CurrentCount = StartCount

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
		CurrentCount += 1

		count, err = queryCount(clientCtx, contractAddress)
		s.Require().NoError(err)
		s.Require().Equal(CurrentCount, count)

		// query current block height
		blockHeight, err := currentBlockHeight(clientCtx)
		s.Require().NoError(err)
		s.T().Logf("current block height %d", blockHeight)

		// schedule the call
		scheduledBlockHeight := blockHeight + 10
		scheduleMsg := scheduletypes.MsgAddSchedule{
			Signer:       val.keyInfo.GetAddress().String(),
			Contract:     contractAddress,
			FunctionName: "scheduled_increment",
			Payer:        val.keyInfo.GetAddress().String(),
			BlockHeight:  scheduledBlockHeight,
		}
		res, err = s.chain.sendMsgs(*clientCtx, &scheduleMsg)
		s.Require().NoError(err)
		s.T().Logf("scheduled call for height %d", scheduledBlockHeight)

		// verify the call was scheduled
		s.Require().Eventuallyf(func() bool {
			height, err := currentBlockHeight(clientCtx)
			s.Require().NoError(err)
			calls, err := queryScheduledCalls(clientCtx)
			s.Require().NoError(err)
			for _, call := range calls {
				if call.Contract == contractAddress && call.Height == scheduledBlockHeight {
					return true
				}
			}
			s.T().Logf("queried scheduled calls at block %d, got %v", height, calls)
			return false
		}, time.Second*30, time.Second*5, "never found scheduled call")

		// watch the blocks and check if the count has updated
		s.Require().Eventuallyf(func() bool {
			blockHeight, err = currentBlockHeight(clientCtx)
			s.Require().NoError(err)
			if blockHeight < scheduledBlockHeight {
				count, err = queryCount(clientCtx, contractAddress)
				s.Require().NoError(err)
				s.Require().Equal(count, CurrentCount, "count was updated before schedule")
			} else if blockHeight > scheduledBlockHeight {
				count, err = queryCount(clientCtx, contractAddress)
				s.Require().NoError(err)
				s.Require().Equal(count, CurrentCount+1, "count was not updated after schedule")
				return true
			}

			return false
		}, time.Minute*2, time.Second*10, "never found an incremented count after the scheduled height")
	})
}
