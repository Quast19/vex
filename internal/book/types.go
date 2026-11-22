package book

type Side uint8

const (
	SideBuy Side = iota + 1
	SideSell
)

type Order struct {
	id        uint64
	price     int64
	qty       int64
	side      Side
	timestamp int64

	// Intrusive doubly-linked list links
	prev  *Order
	next  *Order
	level *PriceLevel
}

func (o *Order) ID() uint64       { return o.id }
func (o *Order) Price() int64     { return o.price }
func (o *Order) Qty() int64       { return o.qty }
func (o *Order) Side() Side       { return o.side }
func (o *Order) Timestamp() int64 { return o.timestamp }

type PriceLevel struct {
	price    int64
	totalQty int64
	count    int
	head     *Order
	tail     *Order
}

func (pl *PriceLevel) Price() int64    { return pl.price }
func (pl *PriceLevel) TotalQty() int64 { return pl.totalQty }
func (pl *PriceLevel) Count() int      { return pl.count }

type Trade struct {
	MakerOrderID uint64
	TakerOrderID uint64
	Price        int64
	Qty          int64
}
