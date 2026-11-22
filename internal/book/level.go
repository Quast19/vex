package book

func (pl *PriceLevel) Append(o *Order) {
	o.prev = pl.tail
	o.next = nil
	o.level = pl

	if pl.tail != nil {
		pl.tail.next = o
	} else {
		pl.head = o
	}
	pl.tail = o

	pl.totalQty += o.qty
	pl.count++
}

func (pl *PriceLevel) Remove(o *Order) {
	if o.prev != nil {
		o.prev.next = o.next
	} else {
		pl.head = o.next
	}

	if o.next != nil {
		o.next.prev = o.prev
	} else {
		pl.tail = o.prev
	}

	pl.totalQty -= o.qty
	pl.count--

	o.prev = nil
	o.next = nil
	o.level = nil
}

func (pl *PriceLevel) Head() *Order {
	return pl.head
}

func (pl *PriceLevel) IsEmpty() bool {
	return pl.count == 0
}
