package book

import (
	"errors"
	"slices"
	"sort"
)

var (
	ErrInvalidOrder     = errors.New("invalid order price, qty, or side")
	ErrDuplicateOrderID = errors.New("duplicate order id")
)

type OrderBook struct {
	bids   []*PriceLevel
	asks   []*PriceLevel
	orders map[uint64]*Order
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		orders: make(map[uint64]*Order),
	}
}

func (ob *OrderBook) AddLimit(id uint64, side Side, price, qty, ts int64) ([]Trade, error) {
	if price <= 0 || qty <= 0 || (side != SideBuy && side != SideSell) {
		return nil, ErrInvalidOrder
	}
	if _, exists := ob.orders[id]; exists {
		return nil, ErrDuplicateOrderID
	}

	incoming := &Order{
		id:        id,
		price:     price,
		qty:       qty,
		side:      side,
		timestamp: ts,
	}

	var trades []Trade
	if side == SideBuy {
		trades = ob.matchBuy(incoming)
	} else {
		trades = ob.matchSell(incoming)
	}

	if incoming.qty > 0 {
		lvl := ob.getOrCreateLevel(side, incoming.price)
		lvl.Append(incoming)
		ob.orders[incoming.id] = incoming
	}

	return trades, nil
}

func (ob *OrderBook) matchBuy(incoming *Order) []Trade {
	var trades []Trade
	for {
		if incoming.qty > 0 && len(ob.asks) > 0 && incoming.price >= ob.asks[0].price {
			bestLevel := ob.asks[0]
			maker := bestLevel.Head()
			fillQty := min(incoming.qty, maker.qty)

			trades = append(trades, Trade{
				MakerOrderID: maker.id,
				TakerOrderID: incoming.id,
				Price:        maker.price,
				Qty:          fillQty,
			})
			incoming.qty -= fillQty
			maker.qty -= fillQty
			bestLevel.totalQty -= fillQty
			if maker.qty == 0 {
				bestLevel.Remove(maker)
				delete(ob.orders, maker.id)
			}
			if bestLevel.IsEmpty() {
				ob.asks = ob.asks[1:]
			}

		} else {
			return trades
		}
	}
}

func (ob *OrderBook) matchSell(incoming *Order) []Trade {
	var trades []Trade
	for {
		if incoming.qty > 0 && len(ob.bids) > 0 && ob.bids[0].price >= incoming.price {
			highestBid := ob.bids[0]
			bidder := highestBid.head
			fillQty := min(incoming.qty, bidder.qty)

			trades = append(trades, Trade{
				MakerOrderID: bidder.id,
				TakerOrderID: incoming.id,  // taker is who is making liquidy less
				Price:        bidder.price, //incoming price could be more,
				Qty:          fillQty,
			})

			incoming.qty -= fillQty
			bidder.qty -= fillQty
			highestBid.totalQty -= fillQty
			if bidder.qty == 0 {
				highestBid.Remove(bidder)
				delete(ob.orders, bidder.id)
			}
			if highestBid.IsEmpty() {
				ob.bids = ob.bids[1:]
			}

		} else {
			return trades
		}
	}

}

func (ob *OrderBook) Cancel(id uint64) bool {
	o, ok := ob.orders[id]
	if !ok {
		return false
	}

	lvl := o.level
	side := o.side
	lvl.Remove(o)
	delete(ob.orders, id)

	if lvl.IsEmpty() {
		ob.removeLevel(side, lvl.price)
	}
	return true
}

func (ob *OrderBook) BestBid() (price, totalQty int64, ok bool) {
	if len(ob.bids) == 0 {
		return 0, 0, false
	}
	return ob.bids[0].price, ob.bids[0].totalQty, true
}

func (ob *OrderBook) BestAsk() (price, totalQty int64, ok bool) {
	if len(ob.asks) == 0 {
		return 0, 0, false
	}
	return ob.asks[0].price, ob.asks[0].totalQty, true
}

func (ob *OrderBook) getOrCreateLevel(side Side, price int64) *PriceLevel {
	if side == SideBuy {
		idx := sort.Search(len(ob.bids), func(i int) bool {
			return ob.bids[i].price <= price
		})
		if idx < len(ob.bids) && ob.bids[idx].price == price {
			return ob.bids[idx]
		}
		lvl := &PriceLevel{price: price}
		ob.bids = slices.Insert(ob.bids, idx, lvl)
		return lvl
	}

	idx := sort.Search(len(ob.asks), func(i int) bool {
		return ob.asks[i].price >= price
	})
	if idx < len(ob.asks) && ob.asks[idx].price == price {
		return ob.asks[idx]
	}
	lvl := &PriceLevel{price: price}
	ob.asks = slices.Insert(ob.asks, idx, lvl)
	return lvl
}

func (ob *OrderBook) removeLevel(side Side, price int64) {
	if side == SideBuy {
		idx := sort.Search(len(ob.bids), func(i int) bool {
			return ob.bids[i].price <= price
		})
		if idx < len(ob.bids) && ob.bids[idx].price == price {
			ob.bids = slices.Delete(ob.bids, idx, idx+1)
		}
		return
	}

	idx := sort.Search(len(ob.asks), func(i int) bool {
		return ob.asks[i].price >= price
	})
	if idx < len(ob.asks) && ob.asks[idx].price == price {
		ob.asks = slices.Delete(ob.asks, idx, idx+1)
	}
}
