package brontes

import (
	"errors"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/forks"
	"github.com/holiman/uint256"
)

type StackSnapshotType int

const (
	StackSnapshotTypeNone StackSnapshotType = iota
	StackSnapshotTypePushes
	StackSnapshotTypeFull
)

type TracingInspectorConfig struct {
	RecordSteps            bool
	RecordMemorySnapshots  bool
	RecordStackSnapshots   StackSnapshotType
	RecordStateDiff        bool
	ExcludePrecompileCalls bool
	RecordCallReturnData   bool
	RecordLogs             bool
}

// As is in the brontes code.
var DefaultTracingInspectorConfig = TracingInspectorConfig{
	RecordSteps:            false,
	RecordMemorySnapshots:  false,
	RecordStackSnapshots:   StackSnapshotTypeNone,
	RecordStateDiff:        false,
	ExcludePrecompileCalls: true,
	RecordCallReturnData:   true,
	RecordLogs:             true,
}

type StackStep struct {
	TraceIdx int
	StepIdx  int
}

type BrontesInspector struct {
	Config             TracingInspectorConfig
	Traces             *CallTraceArena
	TraceStack         []int
	StepStack          []StackStep
	LastCallReturnData *[]byte
	SpecId             *forks.Fork
	ActivePrecompiles  map[common.Address]struct{}
	Transaction        *types.Transaction
	VMContext          *tracing.VMContext
	From               common.Address
}

func NewBrontesInspector(
	config TracingInspectorConfig,
	chainConfig *params.ChainConfig,
	env *tracing.VMContext,
	tx *types.Transaction,
	from common.Address,
) *BrontesInspector {
	activePrecompiles := make(map[common.Address]struct{})
	rules := chainConfig.Rules(env.BlockNumber, env.Random != nil, env.Time, env.ArbOSVersion)
	precompiles := vm.ActivePrecompiles(rules)
	for _, precompile := range precompiles {
		activePrecompiles[precompile] = struct{}{}
	}
	specId := chainConfig.LatestFork(env.Time, env.ArbOSVersion)

	return &BrontesInspector{
		Config:             config,
		Traces:             NewCallTraceArena(),
		TraceStack:         make([]int, 0),
		StepStack:          make([]StackStep, 0),
		LastCallReturnData: nil,
		SpecId:             &specId,
		ActivePrecompiles:  activePrecompiles,
		VMContext:          env,
		Transaction:        tx,
		From:               from,
	}
}

func (insp *BrontesInspector) IsDeep() bool {
	return len(insp.TraceStack) != 0
}

func (insp *BrontesInspector) IsPrecompile(address common.Address) bool {
	_, ok := insp.ActivePrecompiles[address]
	return ok
}

func (insp *BrontesInspector) ActiveTrace() *CallTraceNode {
	if len(insp.TraceStack) == 0 {
		return nil
	}
	idx := insp.TraceStack[len(insp.TraceStack)-1]
	return &insp.Traces.Arena[idx]
}

// lastTraceIdx returns the index of the last trace.
func (b *BrontesInspector) lastTraceIdx() int {
	if len(b.TraceStack) == 0 {
		panic("can't start step without starting a trace first")
	}
	return b.TraceStack[len(b.TraceStack)-1]
}

func (b *BrontesInspector) popTraceIdx() int {
	if len(b.TraceStack) == 0 {
		panic("more traces were filled than started")
	}
	idx := b.TraceStack[len(b.TraceStack)-1]
	b.TraceStack = b.TraceStack[:len(b.TraceStack)-1]
	return idx
}

// startTraceOnCall starts tracking a new call trace.
func (b *BrontesInspector) startTraceOnCall(address common.Address, inputData []byte, value *big.Int, kind CallKind, depth int, caller common.Address, gasLimit uint64, maybePrecompile *bool) {
	log.Info("startTraceOnCall", "address", address, "inputData", inputData, "value", value, "kind", kind, "depth", depth, "caller", caller, "gasLimit", gasLimit, "maybePrecompile", maybePrecompile)
	var pushKind PushTraceKind
	if maybePrecompile != nil && *maybePrecompile {
		pushKind = PushTraceKindPushOnly
	} else {
		pushKind = PushTraceKindPushAndAttachToParent
	}
	trace := CallTrace{
		Depth:           depth,
		Address:         address,
		Kind:            kind,
		Data:            inputData,
		Value:           value,
		Caller:          caller,
		MaybePrecompile: maybePrecompile,
		GasLimit:        gasLimit,
	}
	traceIdx := b.Traces.PushTrace(0, pushKind, trace)
	b.TraceStack = append(b.TraceStack, traceIdx)
}

func (b *BrontesInspector) fillTraceOnCallEnd(gasUsed uint64, err error, reverted bool, output []byte) {
	traceIdx := b.popTraceIdx()
	trace := &b.Traces.Arena[traceIdx].Trace

	if traceIdx == 0 {
		// TODO: handle root call
	} else {
		trace.GasUsed = gasUsed
	}

	trace.Success = !reverted
	trace.Output = output

	b.LastCallReturnData = &output

	// if createdAddress != nil {
	// 	trace.Address = *createdAddress
	// }
}

// Hooks for OnOpcode
func (b *BrontesInspector) startStep(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	traceIdx := b.lastTraceIdx()
	traceNode := &b.Traces.Arena[traceIdx]

	stepIdx := len(traceNode.Trace.Steps)
	b.StepStack = append(b.StepStack, StackStep{TraceIdx: traceIdx, StepIdx: stepIdx})

	var recordedMemory RecordedMemory
	if b.Config.RecordMemorySnapshots {
		recordedMemory = RecordedMemory{Data: scope.MemoryData()}
	}

	var stackData []uint256.Int
	if b.Config.RecordStackSnapshots == StackSnapshotTypeFull {
		stackData = scope.StackData()
	}

	// Leaving out Stack and Memory snapshots empty for now.
	// GasRefundCounter is also set to 0 by default.
	step := CallTraceStep{
		Depth:            depth,
		Pc:               int(pc),
		Op:               vm.OpCode(op),
		Contract:         scope.Address(),
		Stack:            &stackData,
		PushStack:        nil,
		MemorySize:       0,
		Memory:           recordedMemory,
		GasRemaining:     gas,
		GasRefundCounter: 0,
		GasCost:          cost,
		StorageChange:    nil,
	}

	traceNode.Trace.Steps = append(traceNode.Trace.Steps, step)
}

func (b *BrontesInspector) IntoTraceResults(tx *types.Transaction, receipt *types.Receipt, txIndex int) (*TxTrace, error) {
	blockNumber := b.VMContext.BlockNumber
	trace, err := b.buildTrace(tx.Hash(), blockNumber)
	if err != nil {
		return nil, err
	}

	// Create a new big.Int for the effective price (initially 0)
	effectivePrice := big.NewInt(0)

	return &TxTrace{
		BlockNumber:    blockNumber.Uint64(),
		Trace:          *trace,
		TxHash:         b.Transaction.Hash(),
		TxIndex:        txIndex,
		GasUsed:        new(big.Int).SetUint64(receipt.GasUsed),
		EffectivePrice: effectivePrice,
		IsSuccess:      receipt.Status == types.ReceiptStatusSuccessful,
	}, nil
}

func (b *BrontesInspector) IterTraceableNodes() []CallTraceNode {
	nodes := b.Traces.Nodes()
	traceableNodes := make([]CallTraceNode, 0)
	for _, node := range nodes {
		if node.Trace.MaybePrecompile != nil && *node.Trace.MaybePrecompile {
			continue
		}
		traceableNodes = append(traceableNodes, node)
	}
	return traceableNodes
}

func (b *BrontesInspector) TraceAddress(nodes []CallTraceNode, idx int) []uint {
	if idx == 0 {
		return []uint{}
	}

	graph := make([]uint, 0)
	node := nodes[idx]

	if node.Trace.MaybePrecompile != nil && *node.Trace.MaybePrecompile {
		return graph
	}

	for node.Parent != nil {
		childIdx := node.Idx
		parentIdx := *node.Parent
		parentNode := nodes[parentIdx]

		found := false

		// check if childIdx is in parentNode.Children
		var callIdx uint
		for i, child := range parentNode.Children {
			if child == childIdx {
				callIdx = uint(i)
				found = true
				break
			}
		}

		if !found {
			panic("non precompile child call exists in parent")
		}

		graph = append(graph, callIdx)
		node = parentNode
	}

	slices.Reverse(graph)
	return graph
}

func findMsgSender(traces []TransactionTraceWithLogs, trace *TransactionTrace) common.Address {
	var msgSender common.Address

	if trace.Action.Type == ActionTypeCall {

		callAction := trace.Action.Call

		if callAction.CallType == CallKindDelegateCall {
			var prevTrace *TransactionTraceWithLogs
			for i := len(traces) - 1; i >= 0; i-- {
				n := &traces[i]
				if n.Trace.Action.Type == ActionTypeCall {
					if n.Trace.Action.Call.CallType != CallKindDelegateCall {
						prevTrace = n
						break
					}
				}

				if n.Trace.Action.Type == ActionTypeCreate {
					prevTrace = n
					break
				}
			}

			if prevTrace == nil {
				panic("no previous trace found for delegate call")
			}
			msgSender = prevTrace.MsgSender
		} else {
			msgSender = callAction.From
		}
	} else {
		// For non-call actions (create, selfdestruct, etc.)
		msgSender = trace.Action.GetFromAddr()
	}
	return msgSender
}

func (b *BrontesInspector) buildTrace(txHash common.Hash, blockNumber *big.Int) (*[]TransactionTraceWithLogs, error) {
	if len(b.Traces.Nodes()) == 0 {
		return nil, errors.New("no traces found")
	}

	traces := make([]TransactionTraceWithLogs, len(b.Traces.Nodes()))
	for _, node := range b.IterTraceableNodes() {
		traceAddress := b.TraceAddress(b.Traces.Nodes(), node.Idx)
		trace := b.buildTxTrace(&node, traceAddress)
		logs := make([]types.Log, 0, len(node.Logs))
		for _, logData := range node.Logs {
			logs = append(logs, types.Log{
				Address: node.Trace.Address,
				Data:    logData.Data,
				Topics:  logData.Topics,
			})
		}
		msgSender := findMsgSender(traces, trace)

		traces = append(traces, TransactionTraceWithLogs{
			Trace:       *trace,
			Logs:        logs,
			MsgSender:   msgSender,
			DecodedData: nil,
			TraceIdx:    uint64(node.Idx),
		})

		// TODO: handle selfdestruct. Figure out how to get the result of instructions(opcode) after the execution.
		// We need an additional hook for this (OnOpcodeEnd?)
	}
	return &traces, nil
}

func (b *BrontesInspector) buildTxTrace(node *CallTraceNode, traceAddress []uint) *TransactionTrace {
	action := b.ParityAction(node)
	var result *TraceOutput
	if node.Trace.IsError() && !node.Trace.IsRevert() {
		result = nil
	} else {
		result = b.ParityTraceOutput(node)
	}
	instructionErrorMsg := b.AsErrorMsg(node)

	return &TransactionTrace{
		Type:         action.Type,
		Action:       action,
		Error:        &instructionErrorMsg,
		Result:       result,
		TraceAddress: traceAddress,
		Subtraces:    uint(len(node.Children)),
	}
}

func (b *BrontesInspector) ParityAction(node *CallTraceNode) *Action {
	if node.Trace.Kind.IsAnyCall() {
		inner := &CallAction{
			From:     node.Trace.Caller,
			To:       node.Trace.Address,
			Value:    node.Trace.Value,
			Gas:      node.Trace.GasLimit,
			Input:    node.Trace.Data,
			CallType: node.Trace.Kind,
		}
		return &Action{
			Type: ActionTypeCall,
			Call: inner,
		}
	} else if node.Trace.Kind.IsAnyCreate() {
		inner := &CreateAction{
			From:  node.Trace.Caller,
			Value: node.Trace.Value,
			Gas:   node.Trace.GasLimit,
			Init:  node.Trace.Data,
		}
		return &Action{
			Type:   ActionTypeCreate,
			Create: inner,
		}
	}
	panic("unknown action type")
}

func (b *BrontesInspector) ParityTraceOutput(node *CallTraceNode) *TraceOutput {
	if node.Trace.Kind.IsAnyCall() {
		return &TraceOutput{
			Type: TraceOutputTypeCall,
			Call: &CallOutput{
				GasUsed: node.Trace.GasUsed,
				Output:  node.Trace.Output,
			},
		}
	} else if node.Trace.Kind.IsAnyCreate() {
		return &TraceOutput{
			Type: TraceOutputTypeCreate,
			Create: &CreateOutput{
				GasUsed: node.Trace.GasUsed,
				Code:    node.Trace.Output,
				Address: node.Trace.Address,
			},
		}
	}

	panic("unknown trace output type")
}

func (b *BrontesInspector) AsErrorMsg(node *CallTraceNode) string {
	if !node.Trace.IsError() {
		return ""
	}

	// Since we don't have the Trace.Status field, let's just return a generic error message.
	return "Instruction failed"
}

// for both call(), create() and selfdestruct()
// NOTE: The to, from and value that are different to every callKind are handled by the tracer library.
// Any other type of of call
func (b *BrontesInspector) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	callKind := FromCallTypeCode(typ)
	log.Info("OnEnter", "callKind", callKind)
	op := vm.OpCode(typ)
	if op == vm.CREATE || op == vm.CREATE2 {
		b.startTraceOnCall(to, input, value, callKind, depth, from, gas, nil)
	} else if op == vm.SELFDESTRUCT {
		traceIdx := b.lastTraceIdx()
		trace := &b.Traces.Arena[traceIdx].Trace
		trace.SelfdestructRefundTarget = &to
	} else if op == vm.CALL || op == vm.CALLCODE || op == vm.DELEGATECALL || op == vm.STATICCALL {
		// handle Call
		var maybePrecompile *bool
		if b.Config.ExcludePrecompileCalls {
			temp := b.IsPrecompile(to)
			maybePrecompile = &temp
		}
		b.startTraceOnCall(to, input, value, callKind, depth, from, gas, maybePrecompile)
	}
	// we only handle call and create and selfdestruct
}

// call/create end
func (b *BrontesInspector) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	b.fillTraceOnCallEnd(gasUsed, err, reverted, output)
}

// step
func (b *BrontesInspector) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	if b.Config.RecordSteps {
		b.startStep(pc, op, gas, cost, scope, rData, depth, err)
	}
}

// log
func (b *BrontesInspector) OnLog(log *types.Log) {
	traceIdx := b.lastTraceIdx()
	traceNode := &b.Traces.Arena[traceIdx]
	traceNode.Ordering = append(traceNode.Ordering, NewLogCallOrderLog(len(traceNode.Logs)))
	traceNode.Logs = append(traceNode.Logs, LogData{
		Topics: log.Topics,
		Data:   log.Data,
	})
}
