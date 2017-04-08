package orderbook

import (
	"sort"
	"time"
)

type Side uint8

const BidSide Side = 0
const AskSide Side = 1

type Order struct {
	ID    string
	Size  float64
	Price float64
	Side  Side
	Time  time.Time
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

type OrderStateList []*OrderState

func (a OrderStateList) Len() int           { return len(a) }
func (a OrderStateList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a OrderStateList) Less(i, j int) bool { return a[i].Price < a[j].Price }

type BookMapStats struct {
	From time.Time
	To   time.Time
	Bid  OrderStateList
	Ask  OrderStateList
}

type BookMapStatsCopy struct {
	From time.Time
	To   time.Time
	Bid  []OrderState
	Ask  []OrderState
}

func (stats *BookMapStats) Sort() {
	sort.Sort(stats.Bid)
	sort.Sort(stats.Ask)
}

type Book struct {
	ID              string
	Bid             BookLevelList
	Ask             BookLevelList
	OrderMap        map[string]*Order
	Sequence        uint64
	Trades          []*Order
	TradesUpdated   chan string
	Stats           *BookMapStats
	SkipStatsUpdate bool
}

func New(id string) *Book {
	b := &Book{
		ID:       id,
		Bid:      []*BookLevel{},
		Ask:      []*BookLevel{},
		OrderMap: map[string]*Order{},
		Trades:   []*Order{},
	}
	//b.ResetStats()
	return b
}

// TODO: improve. prolly memory/gc hungy
func (b *Book) StatsCopy() *BookMapStatsCopy {
	b.Stats.Sort()

	s := &BookMapStatsCopy{
		From: b.Stats.From,
		To:   b.Stats.To,
		Bid:  make([]OrderState, 0, len(b.Stats.Bid)),
		Ask:  make([]OrderState, 0, len(b.Stats.Ask)),
	}

	for _, state := range b.Stats.Bid {
		s.Bid = append(s.Bid, *state)
	}

	for _, state := range b.Stats.Ask {
		s.Ask = append(s.Ask, *state)
	}

	return s
}

func (b *Book) CenterPrice() float64 {
	if b.Empty() {
		return 0.0
	}
	spread := b.Spread()
	return b.Ask[0].Price + (spread / 2)
}

func (b *Book) Empty() bool {
	return len(b.Ask) == 0 && len(b.Ask) == 0
}

func (b *Book) Sort() {
	sort.Sort(b.Bid)
	sort.Sort(b.Ask)
}

func (b *Book) Spread() float64 {
	b.Sort()
	var spread float64
	if len(b.Bid) > 0 && len(b.Ask) > 0 {
		spread = b.Ask[0].Price - b.Bid[len(b.Bid)-1].Price
	}
	return spread
}

func (b *Book) ResetStats() {
	b.Stats = nil
	b.Stats = &BookMapStats{
		Bid: OrderStateList{},
		Ask: OrderStateList{},
	}

	for _, level := range b.Bid {
		bid := &OrderState{Price: level.Price}
		for _, order := range level.Orders {
			bid.Size += order.Size
			bid.OrderCount += 1
		}
		b.Stats.Bid = append(b.Stats.Bid, bid)
	}

	for _, level := range b.Ask {
		ask := &OrderState{Price: level.Price}
		for _, order := range level.Orders {
			ask.Size += order.Size
			ask.OrderCount += 1
		}
		b.Stats.Ask = append(b.Stats.Ask, ask)
	}
}

func (b *Book) Clear() {
	b.Bid = []*BookLevel{}
	b.Ask = []*BookLevel{}
	b.OrderMap = map[string]*Order{}
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

	if _, ok := b.OrderMap[order.ID]; ok {
		//fmt.Println("BOOK ERROR: ignore duplicated order_id", order)
		return
	}

	b.OrderMap[order.ID] = order

	level, found := b.FindLevel(order)
	if !found {
		b.AddLevel(order)
	} else {
		level.Add(order)

		if b.Stats != nil && !b.SkipStatsUpdate {
			if order.Side == BidSide {
				for _, state := range b.Stats.Bid {
					if state.Price == order.Price {
						state.Size = state.Size + order.Size
						state.OrderCount += 1
						break
					}
				}
			} else {
				for _, state := range b.Stats.Ask {
					if state.Price == order.Price {
						state.Size = state.Size + order.Size
						state.OrderCount += 1
						break
					}
				}
			}
		}
	}
}

func (b *Book) Remove(order_id string) {
	order, ok := b.OrderMap[order_id]
	if !ok {
		//fmt.Println("BOOK wanted to remove order but was not found", order_id)
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
	} else {
		b.Ask = append(b.Ask, level)
	}

	if b.Stats != nil && !b.SkipStatsUpdate { // SkipStatsUpdate here makes no sense
		state := &OrderState{Price: order.Price, Size: order.Size, OrderCount: 1}
		if order.Side == BidSide {
			b.Stats.Bid = append(b.Stats.Bid, state)
		} else {
			b.Stats.Ask = append(b.Stats.Ask, state)
		}
	}
}

func (b *Book) RemoveLevel(side Side, level *BookLevel) {
	if side == BidSide {
		levels := make([]*BookLevel, 0, len(b.Bid)-1)
		for _, current := range b.Bid {
			if current.Price == level.Price {
				continue
			}
			levels = append(levels, current)
		}
		b.Bid = levels
	} else {
		levels := make([]*BookLevel, 0, len(b.Ask)-1)
		for _, current := range b.Ask {
			if current.Price == level.Price {
				continue
			}
			levels = append(levels, current)
		}
		b.Ask = levels
	}
}

func (b *Book) StateAsStats() *BookMapStatsCopy {
	sort.Sort(b.Bid)
	sort.Sort(b.Ask)

	stats := &BookMapStatsCopy{
		Bid: make([]OrderState, 0, len(b.Bid)),
		Ask: make([]OrderState, 0, len(b.Ask)),
	}

	//for i := len(b.Bid) - 1; i >= 0; i-- {
	//	level := b.Bid[i]
	for _, level := range b.Bid {
		bid := OrderState{Price: level.Price}
		for _, order := range level.Orders {
			bid.Size += order.Size
			bid.OrderCount += 1
		}
		stats.Bid = append(stats.Bid, bid)
	}

	for _, level := range b.Ask {
		ask := OrderState{Price: level.Price}
		for _, order := range level.Orders {
			ask.Size += order.Size
			ask.OrderCount += 1
		}
		stats.Ask = append(stats.Ask, ask)
	}

	return stats
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

func (b *Book) Match(data map[string]interface{}, change bool) {
	match := &Order{
		Size:  data["size"].(float64),
		Price: data["price"].(float64),
	}
	if data["side"].(string) == "buy" {
		match.Side = BidSide
	} else {
		match.Side = AskSide
	}
	if _, ok := data["time"]; ok {
		match.Time, _ = time.Parse("2006-01-02T15:04:05.999999Z07:00", data["time"].(string))
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

	if change && b.Stats != nil {
		if match.Side == BidSide {
			for _, state := range b.Stats.Bid {
				if state.Price == order.Price {
					state.Size = state.Size - match.Size
					break
				}
			}
		} else {
			for _, state := range b.Stats.Ask {
				if state.Price == order.Price {
					state.Size = state.Size - match.Size
					break
				}
			}
		}
	}

	if !change && b.Stats != nil {
		if match.Side == BidSide {
			for _, state := range b.Stats.Bid {
				if state.Price == order.Price {
					state.TradeSize += match.Size
					//state.TradeCount += 1
					break
				}
			}
		} else {
			for _, state := range b.Stats.Ask {
				if state.Price == order.Price {
					state.TradeSize += match.Size
					//state.TradeCount += 1
					break
				}
			}
		}
	}

	if !change {
		b.AddTrade(match)
	}
}

func (b *Book) AddTrade(match *Order) {
	//fmt.Println("trade", match)
	if len(b.Trades) >= 50 {
		b.Trades = append(b.Trades[1:], match)
	} else {
		b.Trades = append(b.Trades, match)
	}
	if b.TradesUpdated != nil {
		b.TradesUpdated <- b.ID
	}
}

type OrderState struct {
	Price      float64
	Size       float64
	OrderCount int
	TradeSize  float64
}

func (b *Book) State() ([]OrderState, []OrderState) {
	sort.Sort(b.Bid)
	sort.Sort(b.Ask)

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
	sort.Sort(b.Bid)
	sort.Sort(b.Ask)

	bids := []OrderState{}

	for i := len(b.Bid) - 1; i >= 0; i-- {
		level := b.Bid[i]
		bid := OrderState{Price: level.Price}
		for _, order := range level.Orders {
			bid.Size += order.Size
			bid.OrderCount += 1
		}
		bids = append(bids, bid)
	}

	asks := []OrderState{}
	for _, level := range b.Ask {
		ask := OrderState{Price: level.Price}
		for _, order := range level.Orders {
			ask.Size += order.Size
			ask.OrderCount += 1
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
