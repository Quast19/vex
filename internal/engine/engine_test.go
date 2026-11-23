package engine

import (
	"context"
	"sync"
	"testing"

	"github.com/Quast19/vex/internal/book"
)

func TestEngine_ConcurrentOrdersAndCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := NewEngine(1024, 1024)
	eng.Start(ctx)

	const workers = 50
	var wg sync.WaitGroup
	wg.Add(workers * 2)

	// 50 concurrent buyers
	for i := 1; i <= workers; i++ {
		go func(id uint64) {
			defer wg.Done()
			_, err := eng.SubmitLimit(ctx, id, book.SideBuy, 100, 10, int64(id))
			if err != nil {
				t.Errorf("buy %d failed: %v", id, err)
			}
		}(uint64(i))
	}

	for i := 1; i <= workers; i++ {
		go func(id uint64) {
			defer wg.Done()
			_, err := eng.SubmitLimit(ctx, id, book.SideSell, 100, 10, int64(id))
			if err != nil {
				t.Errorf("sell %d failed: %v", id, err)
			}
		}(uint64(workers + i))
	}

	wg.Wait()

	// Verify event stream emitted updates without blocking
	select {
	case ev := <-eng.Events():
		if ev.BestBidPrice == 0 && ev.BestAskPrice == 0 && len(ev.Trades) == 0 {
			t.Fatalf("expected non-empty market event")
		}
	default:
		t.Fatalf("expected at least one market event on Events()")
	}
}

func TestEngine_ShutdownUnblocksCallers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	eng := NewEngine(16, 16)
	eng.Start(ctx)

	cancel()

	if _, err := eng.SubmitLimit(ctx, 1, book.SideBuy, 100, 10, 1); err != ErrEngineStopped {
		t.Fatalf("expected ErrEngineStopped, got %v", err)
	}
	if _, err := eng.SubmitCancel(ctx, 1); err != ErrEngineStopped {
		t.Fatalf("expected ErrEngineStopped on cancel, got %v", err)
	}
}
