package book

import "testing"

func TestAddLimit_ExactAndPartialMatch(t *testing.T) {
	ob := NewOrderBook()

	// 1. Resting Sell: 10 shares @ 100
	trades, err := ob.AddLimit(1, SideSell, 100, 10, 1)
	if err != nil || len(trades) != 0 {
		t.Fatalf("expected 0 trades and nil err, got trades=%v err=%v", trades, err)
	}

	// 2. Incoming Buy: 6 shares @ 105 (executes at maker price 100)
	trades, err = ob.AddLimit(2, SideBuy, 105, 6, 2)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].MakerOrderID != 1 || trades[0].TakerOrderID != 2 || trades[0].Price != 100 || trades[0].Qty != 6 {
		t.Fatalf("unexpected trade: %+v", trades[0])
	}

	// 3. BestAsk remains 100 with qty 4
	price, qty, ok := ob.BestAsk()
	if !ok || price != 100 || qty != 4 {
		t.Fatalf("expected BestAsk=(100, 4, true), got (%d, %d, %v)", price, qty, ok)
	}
}

func TestAddLimit_MultiLevelSweepAndRest(t *testing.T) {
	ob := NewOrderBook()

	// Resting Asks: 5 @ 100, 5 @ 101, 5 @ 105
	_, _ = ob.AddLimit(1, SideSell, 100, 5, 1)
	_, _ = ob.AddLimit(2, SideSell, 101, 5, 2)
	_, _ = ob.AddLimit(3, SideSell, 105, 5, 3)

	// Incoming Buy: 12 @ 102 -> sweeps 100 and 101, rests 2 @ 102
	trades, err := ob.AddLimit(4, SideBuy, 102, 12, 4)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}
	if trades[0].Price != 100 || trades[0].Qty != 5 || trades[1].Price != 101 || trades[1].Qty != 5 {
		t.Fatalf("unexpected trades: %+v", trades)
	}

	bidPrice, bidQty, ok := ob.BestBid()
	if !ok || bidPrice != 102 || bidQty != 2 {
		t.Fatalf("expected BestBid=(102, 2, true), got (%d, %d, %v)", bidPrice, bidQty, ok)
	}
	askPrice, askQty, ok := ob.BestAsk()
	if !ok || askPrice != 105 || askQty != 5 {
		t.Fatalf("expected BestAsk=(105, 5, true), got (%d, %d, %v)", askPrice, askQty, ok)
	}
}

func TestFIFO_And_CancelMiddleOrder(t *testing.T) {
	ob := NewOrderBook()

	// Three buys at same price 100 (FIFO queue: 1 -> 2 -> 3)
	_, _ = ob.AddLimit(1, SideBuy, 100, 5, 1)
	_, _ = ob.AddLimit(2, SideBuy, 100, 5, 2)
	_, _ = ob.AddLimit(3, SideBuy, 100, 5, 3)

	// Cancel middle order #2
	if !ob.Cancel(2) {
		t.Fatalf("expected Cancel(2) to return true")
	}
	if ob.Cancel(2) {
		t.Fatalf("expected duplicate Cancel(2) to return false")
	}

	// Incoming Sell: 8 @ 100 -> must fill #1 (5) then #3 (3)
	trades, err := ob.AddLimit(4, SideSell, 100, 8, 4)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}
	if trades[0].MakerOrderID != 1 || trades[0].Qty != 5 {
		t.Fatalf("expected first fill from #1 qty 5, got %+v", trades[0])
	}
	if trades[1].MakerOrderID != 3 || trades[1].Qty != 3 {
		t.Fatalf("expected second fill from #3 qty 3, got %+v", trades[1])
	}

	// Remaining BestBid is 100 with qty 2 (from order #3)
	price, qty, ok := ob.BestBid()
	if !ok || price != 100 || qty != 2 {
		t.Fatalf("expected BestBid=(100, 2, true), got (%d, %d, %v)", price, qty, ok)
	}

	// Cancel remaining order #3 -> book must become completely empty
	if !ob.Cancel(3) {
		t.Fatalf("expected Cancel(3) to return true")
	}
	if _, _, ok := ob.BestBid(); ok {
		t.Fatalf("expected empty bids after cancelling last order")
	}
}

func TestValidationErrors(t *testing.T) {
	ob := NewOrderBook()

	if _, err := ob.AddLimit(1, SideBuy, 0, 10, 1); err != ErrInvalidOrder {
		t.Fatalf("expected ErrInvalidOrder for zero price, got %v", err)
	}
	if _, err := ob.AddLimit(1, SideBuy, 100, -5, 1); err != ErrInvalidOrder {
		t.Fatalf("expected ErrInvalidOrder for negative qty, got %v", err)
	}

	_, _ = ob.AddLimit(1, SideBuy, 100, 10, 1)
	if _, err := ob.AddLimit(1, SideSell, 105, 10, 2); err != ErrDuplicateOrderID {
		t.Fatalf("expected ErrDuplicateOrderID, got %v", err)
	}
}
