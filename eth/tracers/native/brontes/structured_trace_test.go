package brontes

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func TestTxTraceJSONMarshaling(t *testing.T) {
	// Create a sample TxTrace
	txHash := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	txTrace := &TxTrace{
		BlockNumber: 12345,
		Trace: []TransactionTraceWithLogs{
			{
				TraceIdx:  1,
				MsgSender: common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Logs: []types.Log{
					{
						Address: common.HexToAddress("0x0987654321098765432109876543210987654321"),
						Topics:  []common.Hash{common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")},
						Data:    []byte("test data"),
						TxHash:  txHash,
					},
				},
			},
		},
		TxHash:         txHash,
		GasUsed:        big.NewInt(21000),
		EffectivePrice: big.NewInt(1000000000),
		IsSuccess:      true,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(txTrace)
	assert.NoError(t, err)

	// Unmarshal back to struct
	var unmarshaledTxTrace TxTrace
	err = json.Unmarshal(jsonData, &unmarshaledTxTrace)
	assert.NoError(t, err)

	// Compare the original and unmarshaled structs
	assert.Equal(t, txTrace.BlockNumber, unmarshaledTxTrace.BlockNumber)
	assert.Equal(t, txTrace.TxHash, unmarshaledTxTrace.TxHash)
	assert.Equal(t, txTrace.GasUsed.String(), unmarshaledTxTrace.GasUsed.String())
	assert.Equal(t, txTrace.EffectivePrice.String(), unmarshaledTxTrace.EffectivePrice.String())
	assert.Equal(t, txTrace.IsSuccess, unmarshaledTxTrace.IsSuccess)
	assert.Equal(t, len(txTrace.Trace), len(unmarshaledTxTrace.Trace))
	if len(txTrace.Trace) > 0 {
		assert.Equal(t, txTrace.Trace[0].TraceIdx, unmarshaledTxTrace.Trace[0].TraceIdx)
		assert.Equal(t, txTrace.Trace[0].MsgSender, unmarshaledTxTrace.Trace[0].MsgSender)
		assert.Equal(t, len(txTrace.Trace[0].Logs), len(unmarshaledTxTrace.Trace[0].Logs))
		if len(txTrace.Trace[0].Logs) > 0 {
			assert.Equal(t, txTrace.Trace[0].Logs[0].TxHash, unmarshaledTxTrace.Trace[0].Logs[0].TxHash)
		}
	}
}

func TestTransactionTraceWithLogsJSONMarshaling(t *testing.T) {
	// Create a sample TransactionTraceWithLogs
	txHash := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	trace := &TransactionTraceWithLogs{
		TraceIdx:  1,
		MsgSender: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Logs: []types.Log{
			{
				Address:     common.HexToAddress("0x0987654321098765432109876543210987654321"),
				Topics:      []common.Hash{common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")},
				Data:        []byte("test data"),
				TxHash:      txHash,
				BlockNumber: 12345,
				TxIndex:     0,
				BlockHash:   common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234"),
				Index:       0,
				Removed:     false,
			},
		},
		Trace: TransactionTrace{
			Action: &Action{
				Type: ActionType(0), // Call type
				Call: &CallAction{
					From:  common.HexToAddress("0x1234567890123456789012345678901234567890"),
					To:    common.HexToAddress("0x0987654321098765432109876543210987654321"),
					Value: big.NewInt(1000000000),
					Input: []byte("test input"),
				},
			},
			Result: &TraceOutput{
				Type: TraceOutputType(0), // Call type
				Call: &CallOutput{
					Output: []byte("test output"),
				},
			},
		},
		DecodedData: &DecodedCallData{
			FunctionName: "testFunction",
			CallData: []DecodedParams{
				{
					FieldName: "param1",
					FieldType: "uint256",
					Value:     "1000000000",
				},
			},
			ReturnData: []DecodedParams{
				{
					FieldName: "result",
					FieldType: "bool",
					Value:     "true",
				},
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(trace)
	assert.NoError(t, err)

	// Unmarshal back to struct
	var unmarshaledTrace TransactionTraceWithLogs
	err = json.Unmarshal(jsonData, &unmarshaledTrace)
	assert.NoError(t, err)

	// Compare the original and unmarshaled structs
	assert.Equal(t, trace.TraceIdx, unmarshaledTrace.TraceIdx)
	assert.Equal(t, trace.MsgSender, unmarshaledTrace.MsgSender)
	assert.Equal(t, len(trace.Logs), len(unmarshaledTrace.Logs))
	if len(trace.Logs) > 0 {
		assert.Equal(t, trace.Logs[0].Address, unmarshaledTrace.Logs[0].Address)
		assert.Equal(t, trace.Logs[0].TxHash, unmarshaledTrace.Logs[0].TxHash)
		assert.Equal(t, trace.Logs[0].BlockNumber, unmarshaledTrace.Logs[0].BlockNumber)
		assert.Equal(t, trace.Logs[0].Data, unmarshaledTrace.Logs[0].Data)
	}
	assert.Equal(t, trace.Trace.Action.Type, unmarshaledTrace.Trace.Action.Type)
	if trace.DecodedData != nil {
		assert.Equal(t, trace.DecodedData.FunctionName, unmarshaledTrace.DecodedData.FunctionName)
		assert.Equal(t, len(trace.DecodedData.CallData), len(unmarshaledTrace.DecodedData.CallData))
	}
}
