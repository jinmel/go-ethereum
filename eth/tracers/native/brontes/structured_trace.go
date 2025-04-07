package brontes

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type TraceActions interface {
	GetCallFrameInfo() CallFrameInfo
	GetFromAddr() common.Address
	GetToAddr() common.Address
	GetMsgSender() common.Address
	GetCallData() []byte
	GetReturnCallData() []byte
	IsStaticCall() bool
	IsCreate() bool
	ActionType() ActionType
	GetCreateOutput() common.Address
	IsDelegateCall() bool
}

type DecodedParams struct {
	FieldName string `json:"field_name"`
	FieldType string `json:"field_type"`
	Value     string `json:"value"`
}

type DecodedCallData struct {
	FunctionName string          `json:"function_name"`
	CallData     []DecodedParams `json:"call_data"`
	ReturnData   []DecodedParams `json:"return_data"`
}

type CallFrameInfo struct {
	TraceIdx      uint64
	CallData      []byte
	ReturnData    []byte
	TargetAddress common.Address
	FromAddress   common.Address
	Logs          []types.Log
	DelegateLogs  []types.Log
	MsgSender     common.Address
	MsgValue      []byte
}

type CallInfo struct {
	TraceIdx      uint64
	TargetAddress common.Address
	FromAddress   common.Address
	MsgSender     common.Address
	MsgValue      *big.Int
}

type TransactionTraceWithLogs struct {
	Trace       TransactionTrace
	Logs        []types.Log
	MsgSender   common.Address
	TraceIdx    uint64
	DecodedData *DecodedCallData
}

func (t *TransactionTraceWithLogs) IsStaticCall() bool {
	return t.Trace.IsStaticCall()
}

func (t *TransactionTraceWithLogs) IsCreate() bool {
	return t.Trace.IsCreate()
}

func (t *TransactionTraceWithLogs) IsDelegateCall() bool {
	return t.Trace.IsDelegateCall()
}

func (t *TransactionTraceWithLogs) GetCreateOutput() common.Address {
	if t.Trace.Result.Type == TraceOutputTypeCreate && t.Trace.Result.Create != nil {
		return common.Address(t.Trace.Result.Create.Address)
	}
	return common.Address{} // default address
}

func (t *TransactionTraceWithLogs) ActionType() ActionType {
	return t.Trace.Action.Type
}

func (t *TransactionTraceWithLogs) GetFromAddr() common.Address {
	return t.Trace.Action.GetFromAddr()
}

func (t *TransactionTraceWithLogs) GetMsgSender() common.Address {
	return t.MsgSender
}

func (t *TransactionTraceWithLogs) GetToAddr() common.Address {
	return t.Trace.Action.GetToAddr()
}

func (t *TransactionTraceWithLogs) GetCallData() []byte {
	return t.Trace.Action.GetCallData()
}

func (t *TransactionTraceWithLogs) GetReturnCallData() []byte {
	if t.Trace.Result == nil {
		return nil
	}

	if t.Trace.Result.Call != nil {
		return t.Trace.Result.Call.Output
	}

	return nil
}

func (t *TransactionTraceWithLogs) GetMsgValue() []byte {
	return t.Trace.Action.GetMsgValue()
}

func (t *TransactionTraceWithLogs) GetCallFrameInfo() CallFrameInfo {
	return CallFrameInfo{
		TraceIdx:      t.TraceIdx,
		CallData:      t.GetCallData(),
		ReturnData:    t.GetReturnCallData(),
		TargetAddress: t.GetToAddr(),
		FromAddress:   t.GetFromAddr(),
		Logs:          t.Logs,
		DelegateLogs:  make([]types.Log, 0),
		MsgSender:     t.MsgSender,
		MsgValue:      t.GetMsgValue(),
	}
}

type TxTrace struct {
	BlockNumber    uint64                     `json:"block_number"`
	Trace          []TransactionTraceWithLogs `json:"trace"`
	TxHash         common.Hash                `json:"tx_hash"`
	GasUsed        *big.Int                   `json:"gas_used"`
	EffectivePrice *big.Int                   `json:"effective_price"`
	IsSuccess      bool                       `json:"is_success"`
}
