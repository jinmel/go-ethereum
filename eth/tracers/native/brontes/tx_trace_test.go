package brontes

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestPrintTxTrace(t *testing.T) {
	// Create a sample transaction trace
	txTrace := &TxTrace{
		BlockNumber:    12345,
		TxHash:         common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		GasUsed:        big.NewInt(21000),
		EffectivePrice: big.NewInt(20000000000), // 20 Gwei
		TxIndex:        0,
		IsSuccess:      true,
		Trace: []TransactionTraceWithLogs{
			{
				TraceIdx:  0,
				MsgSender: common.HexToAddress("0x1111111111111111111111111111111111111111"),
				Logs: []types.Log{
					{
						Address: common.HexToAddress("0x2222222222222222222222222222222222222222"),
						Topics: []common.Hash{
							common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333"),
						},
						Data: []byte{0x01, 0x02, 0x03},
					},
				},
				Trace: TransactionTrace{
					Type: ActionTypeCall,
					Action: &Action{
						Call: &CallAction{
							From:     common.HexToAddress("0x1111111111111111111111111111111111111111"),
							To:       common.HexToAddress("0x2222222222222222222222222222222222222222"),
							Input:    hexutil.Bytes{0x01, 0x02, 0x03},
							Value:    big.NewInt(1000000000000000000), // 1 ETH
							Gas:      21000,
							CallType: CallKindCall,
						},
					},
					Result: &TraceOutput{
						Type: TraceOutputTypeCall,
						Call: &CallOutput{
							GasUsed: 21000,
							Output:  hexutil.Bytes{0x04, 0x05, 0x06},
						},
					},
					Subtraces:    0,
					TraceAddress: []uint{},
				},
			},
		},
	}

	// Convert to JSON for pretty printing
	jsonData, err := json.MarshalIndent(txTrace, "", "  ")
	if err != nil {
		t.Fatalf("Error marshaling txTrace: %v", err)
	}

	// Print the JSON
	fmt.Println("Sample TxTrace:")
	fmt.Println(string(jsonData))
}

func TestPrintTxTraceWithReward(t *testing.T) {
	// Create a sample transaction trace with a reward action
	txTrace := &TxTrace{
		BlockNumber:    12345,
		TxHash:         common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		GasUsed:        big.NewInt(21000),
		EffectivePrice: big.NewInt(20000000000), // 20 Gwei
		TxIndex:        0,
		IsSuccess:      true,
		Trace: []TransactionTraceWithLogs{
			{
				TraceIdx:  0,
				MsgSender: common.HexToAddress("0x1111111111111111111111111111111111111111"),
				Logs:      []types.Log{},
				Trace: TransactionTrace{
					Type: ActionTypeReward,
					Action: &Action{
						Type: ActionTypeReward,
						Reward: &RewardAction{
							Author:     common.HexToAddress("0x4444444444444444444444444444444444444444"),
							RewardType: RewardTypeBlock,
							Value:      big.NewInt(2000000000000000000), // 2 ETH
						},
					},
					Result:       nil,
					Subtraces:    0,
					TraceAddress: []uint{},
				},
			},
		},
	}

	// Convert to JSON for pretty printing
	jsonData, err := json.MarshalIndent(txTrace, "", "  ")
	if err != nil {
		t.Fatalf("Error marshaling txTrace: %v", err)
	}

	// Print the JSON
	fmt.Println("Sample TxTrace with Reward:")
	fmt.Println(string(jsonData))
}
