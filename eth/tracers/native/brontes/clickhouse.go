package brontes

import (
	"fmt"
)

// ClickhouseDecodedCallData represents decoded function call data for ClickHouse
type ClickhouseDecodedCallData struct {
	TraceIdx     []uint64
	FunctionName []string
	CallData     [][]DecodedParams
	ReturnData   [][]DecodedParams
}

// NewClickhouseDecodedCallData creates a ClickhouseDecodedCallData from a TxTrace
func NewClickhouseDecodedCallData(value *TxTrace) *ClickhouseDecodedCallData {
	result := &ClickhouseDecodedCallData{}
	for _, trace := range value.Trace {
		if trace.DecodedData != nil {
			result.TraceIdx = append(result.TraceIdx, trace.TraceIdx)
			result.FunctionName = append(result.FunctionName, trace.DecodedData.FunctionName)
			result.CallData = append(result.CallData, trace.DecodedData.CallData)
			result.ReturnData = append(result.ReturnData, trace.DecodedData.ReturnData)
		}
	}
	return result
}

// ClickhouseLogs represents transaction logs for ClickHouse
type ClickhouseLogs struct {
	TraceIdx []uint64
	LogIdx   []uint64
	Address  []string
	Topics   [][]string
	Data     []string
}

// NewClickhouseLogs creates a ClickhouseLogs from a TxTrace
func NewClickhouseLogs(value *TxTrace) *ClickhouseLogs {
	result := &ClickhouseLogs{}
	for _, trace := range value.Trace {
		for logIdx, log := range trace.Logs {
			result.TraceIdx = append(result.TraceIdx, trace.TraceIdx)
			result.LogIdx = append(result.LogIdx, uint64(logIdx))
			result.Address = append(result.Address, log.Address.String())

			// Convert topics to strings
			topicStrings := make([]string, len(log.Topics))
			for i, topic := range log.Topics {
				topicStrings[i] = topic.String()
			}
			result.Topics = append(result.Topics, topicStrings)

			result.Data = append(result.Data, fmt.Sprintf("%x", log.Data))
		}
	}
	return result
}

// ClickhouseCreateAction represents contract creation actions for ClickHouse
type ClickhouseCreateAction struct {
	TraceIdx []uint64
	From     []string
	Gas      []uint64
	Init     []string
	Value    [][32]byte
}

// NewClickhouseCreateAction creates a ClickhouseCreateAction from a TxTrace
func NewClickhouseCreateAction(value *TxTrace) *ClickhouseCreateAction {
	result := &ClickhouseCreateAction{}
	for _, trace := range value.Trace {
		if trace.IsCreate() {
			result.TraceIdx = append(result.TraceIdx, trace.TraceIdx)
			result.From = append(result.From, trace.Trace.Action.Create.From.String())
			result.Gas = append(result.Gas, trace.Trace.Action.Create.Gas)
			result.Init = append(result.Init, fmt.Sprintf("%x", trace.Trace.Action.Create.Init))

			// Convert big.Int to [32]byte
			var valueBytes [32]byte
			trace.Trace.Action.Create.Value.FillBytes(valueBytes[:])
			result.Value = append(result.Value, valueBytes)
		}
	}
	return result
}

// ClickhouseCallAction represents contract call actions for ClickHouse
type ClickhouseCallAction struct {
	TraceIdx []uint64
	From     []string
	CallType []string
	Gas      []uint64
	Input    []string
	To       []string
	Value    [][32]byte
}

// NewClickhouseCallAction creates a ClickhouseCallAction from a TxTrace
func NewClickhouseCallAction(value *TxTrace) *ClickhouseCallAction {
	result := &ClickhouseCallAction{}
	for _, trace := range value.Trace {

		if trace.Trace.Action.Type == ActionTypeCall {
			result.TraceIdx = append(result.TraceIdx, trace.TraceIdx)
			result.From = append(result.From, trace.Trace.Action.Call.From.String())
			result.CallType = append(result.CallType, trace.Trace.Action.Call.CallType.String())
			result.Gas = append(result.Gas, trace.Trace.Action.Call.Gas)
			result.Input = append(result.Input, fmt.Sprintf("%x", trace.Trace.Action.Call.Input))
			result.To = append(result.To, trace.Trace.Action.Call.To.String())

			var valueBytes [32]byte
			trace.Trace.Action.Call.Value.FillBytes(valueBytes[:])
			result.Value = append(result.Value, valueBytes)
		}
	}
	return result
}

// ClickhouseSelfDestructAction represents self-destruct actions for ClickHouse
type ClickhouseSelfDestructAction struct {
	TraceIdx      []uint64
	Address       []string
	Balance       [][32]byte
	RefundAddress []string
}

// NewClickhouseSelfDestructAction creates a ClickhouseSelfDestructAction from a TxTrace
func NewClickhouseSelfDestructAction(value *TxTrace) *ClickhouseSelfDestructAction {
	result := &ClickhouseSelfDestructAction{}
	for _, trace := range value.Trace {
		if trace.Trace.Action.Type == ActionTypeSelfDestruct {
			result.TraceIdx = append(result.TraceIdx, trace.TraceIdx)
			result.Address = append(result.Address, trace.Trace.Action.SelfDestruct.Address.String())
			result.RefundAddress = append(result.RefundAddress, trace.Trace.Action.SelfDestruct.RefundAddress.String())

			// Convert big.Int to [32]byte
			var balanceBytes [32]byte
			trace.Trace.Action.SelfDestruct.Balance.FillBytes(balanceBytes[:])
			result.Balance = append(result.Balance, balanceBytes)
		}
	}
	return result
}

// ClickhouseRewardAction represents reward actions for ClickHouse
type ClickhouseRewardAction struct {
	TraceIdx   []uint64
	Author     []string
	Value      [][32]byte
	RewardType []string
}

// NewClickhouseRewardAction creates a ClickhouseRewardAction from a TxTrace
func NewClickhouseRewardAction(value *TxTrace) *ClickhouseRewardAction {
	result := &ClickhouseRewardAction{}
	for _, trace := range value.Trace {
		if trace.Trace.Action.Type == ActionTypeReward {
			result.TraceIdx = append(result.TraceIdx, trace.TraceIdx)
			result.Author = append(result.Author, trace.Trace.Action.Reward.Author.String())

			// Convert RewardType to string
			var rewardTypeStr string
			if trace.Trace.Action.Reward.RewardType == RewardTypeBlock {
				rewardTypeStr = "Block"
			} else {
				rewardTypeStr = "Uncle"
			}
			result.RewardType = append(result.RewardType, rewardTypeStr)

			// Convert big.Int to [32]byte
			var valueBytes [32]byte
			trace.Trace.Action.Reward.Value.FillBytes(valueBytes[:])
			result.Value = append(result.Value, valueBytes)
		}
	}
	return result
}

// ClickhouseCallOutput represents call outputs for ClickHouse
type ClickhouseCallOutput struct {
	TraceIdx []uint64
	GasUsed  []uint64
	Output   []string
}

// NewClickhouseCallOutput creates a ClickhouseCallOutput from a TxTrace
func NewClickhouseCallOutput(value *TxTrace) *ClickhouseCallOutput {
	result := &ClickhouseCallOutput{}
	for _, trace := range value.Trace {
		if trace.Trace.Result != nil && trace.Trace.Result.Type == TraceOutputTypeCall && trace.Trace.Result.Call != nil {
			callOutput := trace.Trace.Result.Call
			result.TraceIdx = append(result.TraceIdx, trace.TraceIdx)
			result.GasUsed = append(result.GasUsed, callOutput.GasUsed)
			result.Output = append(result.Output, fmt.Sprintf("%x", callOutput.Output))
		}
	}
	return result
}

// ClickhouseCreateOutput represents contract creation outputs for ClickHouse
type ClickhouseCreateOutput struct {
	TraceIdx []uint64
	Address  []string
	Code     []string
	GasUsed  []uint64
}

// NewClickhouseCreateOutput creates a ClickhouseCreateOutput from a TxTrace
func NewClickhouseCreateOutput(value *TxTrace) *ClickhouseCreateOutput {
	result := &ClickhouseCreateOutput{}
	for _, trace := range value.Trace {
		if trace.Trace.Result != nil && trace.Trace.Result.Type == TraceOutputTypeCreate && trace.Trace.Result.Create != nil {
			createOutput := trace.Trace.Result.Create
			result.TraceIdx = append(result.TraceIdx, trace.TraceIdx)
			result.Address = append(result.Address, createOutput.Address.String())
			result.Code = append(result.Code, fmt.Sprintf("%x", createOutput.Code))
			result.GasUsed = append(result.GasUsed, createOutput.GasUsed)
		}
	}
	return result
}
