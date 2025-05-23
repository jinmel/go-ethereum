package native

import (
	"encoding/json"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/native/brontes"
	ethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

func init() {
	tracers.DefaultDirectory.Register("brontesTracer", newBrontesTracer, false)
}

type brontesTracer struct {
	ctx         *tracers.Context
	inspector   *brontes.BrontesInspector
	chainConfig *params.ChainConfig
	receipt     *types.Receipt
	tx          *types.Transaction
	// for stopping the tracer
	interrupt atomic.Bool
	reason    error
}

func newBrontesTracerObject(ctx *tracers.Context, _ json.RawMessage, chainConfig *params.ChainConfig) (*brontesTracer, error) {
	return &brontesTracer{
		ctx:         ctx,
		chainConfig: chainConfig,
	}, nil
}

func newBrontesTracer(ctx *tracers.Context, cfg json.RawMessage, chainConfig *params.ChainConfig) (*tracers.Tracer, error) {
	t, err := newBrontesTracerObject(ctx, cfg, chainConfig)
	if err != nil {
		return nil, err
	}
	return &tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnTxStart: t.OnTxStart,
			OnTxEnd:   t.OnTxEnd,
			OnEnter:   t.OnEnter,
			OnExit:    t.OnExit,
			OnOpcode:  t.OnOpcode,
			OnLog:     t.OnLog,
		},
		GetResult: t.GetResult,
		Stop:      t.Stop,
	}, nil
}

// step
func (t *brontesTracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	if t.interrupt.Load() {
		return
	}
	t.inspector.OnOpcode(pc, op, gas, cost, scope, rData, depth, err)
}

func (*brontesTracer) OnFault(pc uint64, op byte, gas, cost uint64, _ tracing.OpContext, depth int, err error) {
}

// Step in
func (t *brontesTracer) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	if t.interrupt.Load() {
		return
	}
	ethlog.Debug("BrontesTracer: OnEnter", "depth", depth, "typ", typ, "from", from.Hex(), "to", to.Hex(), "input", input, "gas", gas, "value", value)
	err := t.inspector.OnEnter(depth, typ, from, to, input, gas, value)
	if err != nil {
		ethlog.Error("BrontesTracer: OnEnter", "error", err)
		t.interrupt.Store(true)
	}
}

// Step out
func (t *brontesTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if t.interrupt.Load() {
		return
	}
	ethlog.Debug("BrontesTracer: OnExit", "depth", depth, "output", output, "gasUsed", gasUsed, "err", err, "reverted", reverted)
	t.inspector.OnExit(depth, output, gasUsed, err, reverted)
}

func (t *brontesTracer) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	// Initialize the BrontesInspector
	t.inspector = brontes.NewBrontesInspector(brontes.DefaultTracingInspectorConfig, t.chainConfig, env, tx, from)
	t.tx = tx
}

func (t *brontesTracer) OnTxEnd(receipt *types.Receipt, err error) {
	if receipt != nil {
		ethlog.Debug("BrontesTracer: Transaction ended", "txHash", receipt.TxHash.Hex(), "err", err)
	}
	t.receipt = receipt
}

func (t *brontesTracer) OnLog(log *types.Log) {
	if t.interrupt.Load() {
		return
	}
	t.inspector.OnLog(log)
}

func (t *brontesTracer) GetResult() (json.RawMessage, error) {
	result, err := t.inspector.IntoTraceResults(t.tx, t.receipt, t.ctx.TxIndex)
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *brontesTracer) Stop(err error) {
	t.reason = err
	t.interrupt.Store(true)
}
