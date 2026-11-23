package engine

import (
	"context"
	"errors"

	"github.com/Quast19/vex/internal/book"
)

var ErrEngineStopped = errors.New("matching engine stopped")

type CommandType uint8

const (
	CmdAddLimit CommandType = iota + 1
	CmdCancel
)

type Response struct {
	Trades    []book.Trade
	Cancelled bool
	Err       error
}

type Command struct {
	ID        uint64
	Type      CommandType
	Price     int64
	Qty       int64
	Side      book.Side
	Timestamp int64
	ReplyCh   chan Response
}

type MarketEvent struct {
	Trades       []book.Trade
	BestBidPrice int64
	BestBidQty   int64
	HasBid       bool
	BestAskPrice int64
	BestAskQty   int64
	HasAsk       bool
}

type Engine struct {
	ob      *book.OrderBook
	cmdCh   chan Command
	eventCh chan MarketEvent
}

func NewEngine(cmdBufSize, eventBufSize int) *Engine {
	return &Engine{
		ob:      book.NewOrderBook(),
		cmdCh:   make(chan Command, cmdBufSize),
		eventCh: make(chan MarketEvent, eventBufSize),
	}
}

func (e *Engine) Events() <-chan MarketEvent {
	return e.eventCh
}

func (e *Engine) Start(ctx context.Context) {
	go e.run(ctx)
}

func (e *Engine) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cmd := <-e.cmdCh:
			e.handleCommand(cmd)
		}
	}
}

func (e *Engine) handleCommand(cmd Command) {
	switch cmd.Type {
	case CmdAddLimit:
		trades, err := e.ob.AddLimit(cmd.ID, cmd.Side, cmd.Price, cmd.Qty, cmd.Timestamp)
		cmd.ReplyCh <- Response{Trades: trades, Err: err}
		if err == nil {
			e.emitEvent(trades)
		}
	case CmdCancel:
		cancelled := e.ob.Cancel(cmd.ID)
		cmd.ReplyCh <- Response{Cancelled: cancelled}
		if cancelled {
			e.emitEvent(nil)
		}
	}
}

func (e *Engine) emitEvent(trades []book.Trade) {
	bidPrice, bidQty, hasBid := e.ob.BestBid()
	askPrice, askQty, hasAsk := e.ob.BestAsk()
	ev := MarketEvent{
		Trades:       trades,
		BestBidPrice: bidPrice,
		BestBidQty:   bidQty,
		HasBid:       hasBid,
		BestAskPrice: askPrice,
		BestAskQty:   askQty,
		HasAsk:       hasAsk,
	}
	select {
	case e.eventCh <- ev:

	default:
	}

}

func (e *Engine) SubmitLimit(ctx context.Context, id uint64, side book.Side, price, qty, ts int64) ([]book.Trade, error) {
	replyCh := make(chan Response, 1)
	cmd := Command{
		ID:        id,
		Type:      CmdAddLimit,
		Price:     price,
		Qty:       qty,
		Side:      side,
		Timestamp: ts,
		ReplyCh:   replyCh,
	}

	select {
	case <-ctx.Done():
		return nil, ErrEngineStopped
	case e.cmdCh <- cmd:
	}

	select {
	case <-ctx.Done():
		return nil, ErrEngineStopped
	case res := <-replyCh:
		return res.Trades, res.Err
	}

}

func (e *Engine) SubmitCancel(ctx context.Context, orderID uint64) (bool, error) {
	replyCh := make(chan Response, 1)
	cmd := Command{
		ID:      orderID,
		Type:    CmdCancel,
		ReplyCh: replyCh,
	}

	select {
	case <-ctx.Done():
		return false, ErrEngineStopped
	case e.cmdCh <- cmd:
	}

	select {
	case <-ctx.Done():
		return false, ErrEngineStopped
	case res := <-replyCh:
		return res.Cancelled, res.Err
	}
}
