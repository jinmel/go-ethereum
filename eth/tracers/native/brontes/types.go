package brontes

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

// ---------------------------------------------------------------------
// Basic types and helpers
// ---------------------------------------------------------------------

// LogData represents log data with topics and data.
type LogData struct {
	Topics []common.Hash
	Data   []byte
}

// ---------------------------------------------------------------------
// Types for tracing and call frames
// ---------------------------------------------------------------------

// CallTrace represents a trace of a call.
type CallTrace struct {
	Depth                    int
	Success                  bool
	Caller                   common.Address
	Address                  common.Address // For CALL calls, this is the callee; for CREATE, it is the created address.
	MaybePrecompile          *bool
	SelfdestructRefundTarget *common.Address
	Kind                     CallKind
	Value                    *big.Int
	Data                     []byte
	Output                   []byte
	GasUsed                  uint64
	GasLimit                 uint64
	Reverted                 bool
	Error                    error
	Steps                    []CallTraceStep
}

func (ct *CallTrace) IsError() bool {
	return ct.Error != nil
}

func (ct *CallTrace) IsRevert() bool {
	return ct.Reverted
}

func (ct *CallTrace) AsErrorMsg(kind TraceStyle) *string {
	if !ct.IsError() {
		return nil
	}
	errMsg := ct.Error.Error()
	return &errMsg
}

// CallTraceNode represents a node in the call trace arena.
type CallTraceNode struct {
	Parent   *int
	Children []int
	Idx      int
	Trace    CallTrace
	Logs     []LogData
	Ordering []LogCallOrder
}

// ExecutionAddress returns the execution address based on the call kind.
func (ctn *CallTraceNode) ExecutionAddress() common.Address {
	if ctn.Trace.Kind.IsDelegate() {
		return ctn.Trace.Caller
	}
	return ctn.Trace.Address
}

// IsPrecompile returns true if the trace is a call to a precompile.
func (ctn *CallTraceNode) IsPrecompile() bool {
	if ctn.Trace.MaybePrecompile != nil {
		return *ctn.Trace.MaybePrecompile
	}
	return false
}

// Kind returns the kind of the call.
func (ctn *CallTraceNode) Kind() CallKind {
	return ctn.Trace.Kind
}

// IsSelfdestruct returns true if the call was a selfdestruct.
func (ctn *CallTraceNode) IsSelfdestruct() bool {
	return ctn.Trace.SelfdestructRefundTarget != nil
}

// ---------------------------------------------------------------------
// Call kinds and conversions
// ---------------------------------------------------------------------

// CallKind is an enumeration of call types.
type CallKind int

const (
	CallKindCall CallKind = iota
	CallKindStaticCall
	CallKindCallCode
	CallKindDelegateCall
	CallKindCreate
	CallKindCreate2
)

func FromCallTypeCode(typ byte) CallKind {
	callScheme := vm.OpCode(typ)
	switch callScheme {
	case vm.CALL:
		return CallKindCall
	case vm.STATICCALL:
		return CallKindStaticCall
	case vm.CALLCODE:
		return CallKindCallCode
	case vm.DELEGATECALL:
		return CallKindDelegateCall
	case vm.CREATE:
		return CallKindCreate
	case vm.CREATE2:
		return CallKindCreate2
	}
	panic("unknown call type")
}

func (ck CallKind) IsAnyCreate() bool {
	return ck == CallKindCreate || ck == CallKindCreate2
}

func (ck CallKind) IsAnyCall() bool {
	return ck == CallKindCall || ck == CallKindCallCode || ck == CallKindStaticCall || ck == CallKindDelegateCall
}

func (ck CallKind) IsDelegate() bool {
	return ck == CallKindDelegateCall || ck == CallKindCallCode
}

func (ck CallKind) IsStaticCall() bool {
	return ck == CallKindStaticCall
}

func (ck CallKind) String() string {
	switch ck {
	case CallKindCall:
		return "CALL"
	case CallKindStaticCall:
		return "STATICCALL"
	case CallKindCallCode:
		return "CALLCODE"
	case CallKindDelegateCall:
		return "DELEGATECALL"
	case CallKindCreate:
		return "CREATE"
	case CallKindCreate2:
		return "CREATE2"
	default:
		return "UNKNOWN"
	}
}

// ---------------------------------------------------------------------
// Additional supporting types
// ---------------------------------------------------------------------

// CallTraceStepStackItem represents an item on the call trace step stack.
type CallTraceStepStackItem struct {
	TraceNode   *CallTraceNode
	Step        *CallTraceStep
	CallChildID *int
}

// CallTraceStep represents a tracked execution step.
type CallTraceStep struct {
	Depth            int
	Pc               int
	Op               vm.OpCode
	Contract         common.Address
	Stack            *[]uint256.Int // nil if not captured
	PushStack        *[]uint256.Int
	Memory           RecordedMemory
	MemorySize       int
	GasRemaining     uint64
	GasRefundCounter uint64
	GasCost          uint64
	StorageChange    *StorageChange
}

// ---------------------------------------------------------------------
// Storage and memory types
// ---------------------------------------------------------------------

// StorageChangeReason indicates why a storage slot was modified.
type StorageChangeReason int

const (
	StorageChangeReasonSLOAD StorageChangeReason = iota
	StorageChangeReasonSSTORE
)

// StorageChange represents a change to contract storage.
type StorageChange struct {
	Key      *big.Int
	Value    *big.Int
	HadValue *big.Int
	Reason   StorageChangeReason
}

// RecordedMemory wraps captured execution memory.
type RecordedMemory struct {
	Data []byte
}

func NewRecordedMemory(mem []byte) RecordedMemory {
	return RecordedMemory{Data: mem}
}

func (rm *RecordedMemory) AsBytes() []byte {
	return rm.Data
}

func (rm *RecordedMemory) Resize(size int) {
	if len(rm.Data) < size {
		newData := make([]byte, size)
		copy(newData, rm.Data)
		rm.Data = newData
	} else {
		rm.Data = rm.Data[:size]
	}
}

func (rm *RecordedMemory) Len() int {
	return len(rm.Data)
}

func (rm *RecordedMemory) IsEmpty() bool {
	return len(rm.Data) == 0
}

func (rm *RecordedMemory) MemoryChunks() []string {
	return convertMemory(rm.AsBytes())
}

// TransactionTrace represents a parity transaction trace.
type TransactionTrace struct {
	Action       *Action       `json:"action"`
	Error        *string      `json:"error,omitempty"`
	Result       *TraceOutput `json:"result,omitempty"`
	Subtraces    uint64       `json:"subtraces"`
	TraceAddress []uint64     `json:"traceAddress"`
}

func (t *TransactionTrace) IsStaticCall() bool {
	if t.Action.Type == ActionKindCall && t.Action.Call.CallType == CallKindStaticCall {
		return true
	}
	return false
}

func (t *TransactionTrace) IsCreate() bool {
	return t.Action.Type == ActionKindCreate
}

func (t *TransactionTrace) IsDelegateCall() bool {
	if t.Action.Type == ActionKindCall && t.Action.Call.CallType == CallKindDelegateCall {
		return true
	}
	return false
}

type ActionType int

const (
	ActionKindCall = iota
	ActionKindCreate
	ActionKindSelfDestruct
	ActionKindReward
)

// Action represents a call action (or create/selfdestruct).
type Action struct {
	Type         ActionType
	Call         *CallAction
	Create       *CreateAction
	SelfDestruct *SelfdestructAction
	Reward       *RewardAction
}

func (a *Action) GetFromAddr() common.Address {
	switch a.Type {
	case ActionKindCall:
		return a.Call.From
	case ActionKindCreate:
		return a.Create.From
	case ActionKindSelfDestruct:
		return a.SelfDestruct.Address
	case ActionKindReward:
		return a.Reward.Author
	}
	panic("unknown action type")
}

func (a *Action) GetToAddr() common.Address {
	switch a.Type {
	case ActionKindCall:
		return a.Call.To
	case ActionKindCreate:
		return common.Address{}
	case ActionKindSelfDestruct:
		return a.SelfDestruct.Address
	case ActionKindReward:
		return common.Address{}
	}
	panic("unknown action type")
}

func (a *Action) GetMsgValue() []byte {
	switch a.Type {
	case ActionKindCall:
		return a.Call.Value.Bytes()
	case ActionKindCreate:
		return a.Create.Value.Bytes()
	case ActionKindSelfDestruct:
		return []byte{}
	case ActionKindReward:
		return a.Reward.Value.Bytes()
	}
	panic("unknown action type")
}

func (a *Action) GetCallData() []byte {
	switch a.Type {
	case ActionKindCall:
		return a.Call.Input
	case ActionKindCreate:
		return a.Create.Init
	case ActionKindSelfDestruct:
		return []byte{}
	case ActionKindReward:
		return []byte{}
	}
	panic("unknown action type")
}

type RewardType string

const (
	RewardTypeBlock RewardType = "block"
	RewardTypeUncle RewardType = "uncle"
)

// CallAction represents a call action.
type CallAction struct {
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	Value    *big.Int       `json:"value"`
	Gas      uint64         `json:"gas"`
	Input    []byte         `json:"input"`
	CallType CallKind       `json:"callType"`
}

func (ca *CallAction) GetFromAddr() common.Address {
	return ca.From
}

func (ca *CallAction) ActionType() ActionType {
	return ActionKindCall
}

func (ca *CallAction) GetToAddr() common.Address {
	return ca.To
}

func (ca *CallAction) GetMsgValue() []byte {
	return ca.Value.Bytes()
}

func (ca *CallAction) GetCallData() []byte {
	return ca.Input
}

// CallOutput represents the output of a call.
type CallOutput struct {
	GasUsed uint64 `json:"gasUsed"`
	Output  []byte `json:"output"`
}

// CreateAction represents a contract creation action.
type CreateAction struct {
	From  common.Address `json:"from"`
	Value *big.Int       `json:"value"`
	Gas   uint64         `json:"gas"`
	Init  []byte         `json:"init"`
}

func (ca *CreateAction) GetFromAddr() common.Address {
	return ca.From
}

func (ca *CreateAction) ActionType() ActionType {
	return ActionKindCall
}

func (ca *CreateAction) GetToAddr() common.Address {
	return common.Address{}
}

func (ca *CreateAction) GetMsgValue() []byte {
	return ca.Value.Bytes()
}

func (ca *CreateAction) GetCallData() []byte {
	return ca.Init
}

type RewardAction struct {
	Author     common.Address `json:"author"`
	RewardType RewardType     `json:"rewardType"`
	Value      *big.Int       `json:"value"`
}

func (ra *RewardAction) GetFromAddr() common.Address {
	return ra.Author
}

func (ra *RewardAction) ActionType() ActionType {
	return ActionKindReward
}

func (ra *RewardAction) GetToAddr() common.Address {
	return common.Address{}
}

func (ra *RewardAction) GetMsgValue() []byte {
	return ra.Value.Bytes()
}

func (ra *RewardAction) GetCallData() []byte {
	return []byte{}
}

// CreateOutput represents the output of a contract creation.
type CreateOutput struct {
	GasUsed uint64         `json:"gasUsed"`
	Code    []byte         `json:"code"`
	Address common.Address `json:"address"`
}

// SelfdestructAction represents a selfdestruct action.
type SelfdestructAction struct {
	Address       common.Address `json:"address"`
	RefundAddress common.Address `json:"refundAddress"`
	Balance       *big.Int       `json:"balance"`
}

func (sa *SelfdestructAction) GetFromAddr() common.Address {
	return sa.Address
}

func (sa *SelfdestructAction) ActionType() ActionType {
	return ActionKindSelfDestruct
}

func (sa *SelfdestructAction) GetToAddr() common.Address {
	return sa.Address
}

func (sa *SelfdestructAction) GetMsgValue() []byte {
	return []byte{}
}

func (sa *SelfdestructAction) GetCallData() []byte {
	return []byte{}
}

type TraceOutputType int

const (
	TraceOutputTypeCall TraceOutputType = iota
	TraceOutputTypeCreate
)

// TraceOutput represents the output in a trace (either call or create).
type TraceOutput struct {
	Type   TraceOutputType
	Call   *CallOutput
	Create *CreateOutput
}

// MarshalJSON implements the json.Marshaler interface
func (to *TraceOutput) MarshalJSON() ([]byte, error) {
	if to.Type == TraceOutputTypeCall {
		return json.Marshal(to.Call)
	} else if to.Type == TraceOutputTypeCreate {
		return json.Marshal(to.Create)
	}
	return nil, fmt.Errorf("unknown trace output type: %d", to.Type)
}

// LogCallOrderType distinguishes between a log index and a call (trace node) index.
type LogCallOrderType int

const (
	// LogCallOrderLog indicates that the ordering holds the index of a corresponding log.
	LogCallOrderLog LogCallOrderType = iota
	// LogCallOrderCall indicates that the ordering holds the index of a corresponding trace node.
	LogCallOrderCall
)

// LogCallOrder represents the ordering for calls and logs.
// It contains a type tag (LogCallOrderLog or LogCallOrderCall) and an associated index.
type LogCallOrder struct {
	Type  LogCallOrderType
	Index int
}

func NewLogCallOrderCall(i int) LogCallOrder {
	return LogCallOrder{Type: LogCallOrderCall, Index: i}
}

func NewLogCallOrderLog(i int) LogCallOrder {
	return LogCallOrder{Type: LogCallOrderLog, Index: i}
}

type TransactionInfo struct {
	Hash        *common.Hash
	Index       *uint64
	BlockHash   *common.Hash
	BlockNumber *uint64
	BaseFee     *big.Int
}

type ExecutionStatus int

const (
	ExecutionSuccess = iota
	ExecutionRevert
	ExecutionHalt
)

type SuccessReason int

const (
	SuccessReasonStop = iota
	SuccessReasonReturn
	SuccessReasonSelfDestructj
)

type ExeuctionResultSuccess struct {
	Reason      SuccessReason
	GasUsed     uint64
	GasRefunded uint64
	Logs        []LogData
	Output      TraceOutput
}

type ExeuctionResultRevert struct {
	GasUsed uint64
	Output  []byte
}

type HaltReason int

// TODO: There are more than 10 reasons for a halt, but let's not take care of it now since we are not interested to them at the moment.
const (
	HaltReasonFail = iota
)

type ExeuctionResultHalt struct {
	Reason  HaltReason
	GasUsed uint64
}

type ExecutionResult struct {
	Status  ExecutionStatus
	Success *ExeuctionResultSuccess
	Revert  *ExeuctionResultRevert
	Halt    *ExeuctionResultHalt
}

func (er *ExecutionResult) GasUsed() uint64 {
	switch er.Status {
	case ExecutionSuccess:
		return er.Success.GasUsed
	case ExecutionRevert:
		return er.Revert.GasUsed
	case ExecutionHalt:
		return er.Halt.GasUsed
	}
	panic("unknown execution result status")
}

func (er *ExecutionResult) IsSuccess() bool {
	return er.Status == ExecutionSuccess
}
