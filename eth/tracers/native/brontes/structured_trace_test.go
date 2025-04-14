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