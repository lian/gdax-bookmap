package orderbook

import "sort"

type Side uint8

const BidSide Side = 0
const AskSide Side = 1

type Order struct {
	ID    string
	Size  float64
	Price float64
	Side  Side
}

type BookLevel struct {
	Price  float64
	Orders []*Order
}

func (bl *BookLevel) Add(order *Order) {
	bl.Orders = append(bl.Orders, order)
}

func (bl *BookLevel) Remove(order *Order) {
	orders := []*Order{}
	for _, current := range bl.Orders {
		if current == order {
			continue
		}
		orders = append(orders, current)
	}
	bl.Orders = orders
}

func (bl *BookLevel) Empty() bool {
	return len(bl.Orders) == 0
}

type BookLevelList []*BookLevel

func (a BookLevelList) Len() int           { return len(a) }
func (a BookLevelList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BookLevelList) Less(i, j int) bool { return a[i].Price < a[j].Price }

type Book struct {
	ID           string
	Bid          BookLevelList
	Ask          BookLevelList
	OrderMap     map[string]*Order
	LastSequence uint64
	SyncSequence uint64
}

func NewBook(id string) *Book {
	return &Book{
		ID:       id,
		Bid:      []*BookLevel{},
		Ask:      []*BookLevel{},
		OrderMap: map[string]*Order{},
	}
}

func New(id string) *Book {
	return &Book{
		ID:       id,
		Bid:      []*BookLevel{},
		Ask:      []*BookLevel{},
		OrderMap: map[string]*Order{},
	}
}

func (b *Book) Add(data map[string]interface{}) {
	order := &Order{
		ID:    data["id"].(string),
		Size:  data["size"].(float64),
		Price: data["price"].(float64),
	}
	if data["side"].(string) == "buy" {
		order.Side = BidSide
	} else {
		order.Side = AskSide
	}

	b.OrderMap[order.ID] = order

	level, found := b.FindLevel(order)
	if !found {
		b.AddLevel(order)
	}
	level.Add(order)
}

func (b *Book) Remove(order_id string) {
	order, ok := b.OrderMap[order_id]
	if !ok {
		return
	}
	delete(b.OrderMap, order_id)

	level, _ := b.FindLevel(order)
	level.Remove(order)
	if level.Empty() {
		b.RemoveLevel(order.Side, level)
	}
}

func (b *Book) AddLevel(order *Order) {
	level := &BookLevel{Price: order.Price, Orders: []*Order{order}}
	if order.Side == BidSide {
		b.Bid = append(b.Bid, level)
		sort.Sort(b.Bid)
	} else {
		b.Ask = append(b.Ask, level)
		sort.Sort(b.Ask)
	}
}

func (b *Book) RemoveLevel(side Side, level *BookLevel) {
	levels := []*BookLevel{}
	if side == BidSide {
		for _, current := range b.Bid {
			if current.Price == level.Price {
				continue
			}
			levels = append(levels, current)
		}
		b.Bid = levels
	} else {
		for _, current := range b.Ask {
			if current.Price == level.Price {
				continue
			}
			levels = append(levels, current)
		}
		b.Ask = levels
	}
}

func (b *Book) FindLevel(order *Order) (*BookLevel, bool) {
	if order.Side == BidSide {
		for _, level := range b.Bid {
			if level.Price == order.Price {
				return level, true
			}
		}
	} else {
		for _, level := range b.Ask {
			if level.Price == order.Price {
				return level, true
			}
		}
	}

	return &BookLevel{}, false
}

func (b *Book) Match(data map[string]interface{}) {
	match := &Order{
		Size:  data["size"].(float64),
		Price: data["price"].(float64),
	}
	if data["side"].(string) == "bid" {
		match.Side = BidSide
	} else {
		match.Side = AskSide
	}

	var maker_id, taker_id string
	if id, ok := data["maker_order_id"]; ok {
		maker_id = id.(string)
	}
	if id, ok := data["taker_order_id"]; ok {
		taker_id = id.(string)
	}

	level, found := b.FindLevel(match)
	if !found {
		return
	}

	var order *Order

	for _, current := range level.Orders {
		if current.ID == maker_id || current.ID == taker_id {
			order = current
		}
	}

	if order == nil {
		return
	}

	order.Size = order.Size - match.Size

	if order.Size <= 0 {
		b.Remove(order.ID)
	}
}

type OrderState struct {
	Price float64
	Size  float64
}

func (b *Book) State() ([]OrderState, []OrderState) {
	bids := []OrderState{}
	for i := len(b.Bid) - 1; i >= 0; i-- {
		for _, order := range b.Bid[i].Orders {
			bids = append(bids, OrderState{Price: order.Price, Size: order.Size})
		}
	}

	asks := []OrderState{}
	for _, level := range b.Ask {
		for _, order := range level.Orders {
			asks = append(asks, OrderState{Price: order.Price, Size: order.Size})
		}
	}

	return bids, asks
}

func (b *Book) StateCombined() ([]OrderState, []OrderState) {
	bids := []OrderState{}

	for i := len(b.Bid) - 1; i >= 0; i-- {
		level := b.Bid[i]
		bid := OrderState{Price: level.Price}
		for _, order := range level.Orders {
			bid.Size += order.Size
		}
		bids = append(bids, bid)
	}

	asks := []OrderState{}
	for _, level := range b.Ask {
		ask := OrderState{Price: level.Price}
		for _, order := range level.Orders {
			ask.Size += order.Size
		}
		asks = append(asks, ask)
	}

	return bids, asks
}

/*
func main() {
	fmt.Println("foo")
	book := NewBook()
	book.Add(map[string]interface{}{
		"id":    "id-1",
		"price": 100.0,
		"size":  1.0,
		"side":  "bid",
	})
	book.Add(map[string]interface{}{
		"id":    "id-1-1",
		"price": 100.0,
		"size":  1.0,
		"side":  "bid",
	})
	book.Add(map[string]interface{}{
		"id":    "id-2",
		"price": 100.5,
		"size":  1.0,
		"side":  "bid",
	})
	book.Add(map[string]interface{}{
		"id":    "id-3",
		"price": 99.5,
		"size":  0.5,
		"side":  "ask",
	})
	book.Add(map[string]interface{}{
		"id":    "id-4",
		"price": 99.4,
		"size":  0.5,
		"side":  "ask",
	})

	book.Remove("id-1")
	book.Remove("id-3")
	book.Remove("id-4")

	book.Match(map[string]interface{}{
		"size":           0.5,
		"price":          100.0,
		"side":           "bid",
		"maker_order_id": "id-1-1",
	})

	book.Match(map[string]interface{}{
		"size":           0.5,
		"price":          100.0,
		"side":           "bid",
		"maker_order_id": "id-1-1",
	})

	spew.Dump(book)
}
*/
