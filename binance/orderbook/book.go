package orderbook

import (
	"sort"
	"time"

	"github.com/lian/gdax-bookmap/orderbook/product_info"
)

type BookLevel struct {
	Price    float64
	Quantity float64
}

type Trade struct {
	Price    float64
	Quantity float64
	Time     time.Time
}

type BookLevelList []*BookLevel

func (a BookLevelList) Len() int           { return len(a) }
func (a BookLevelList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BookLevelList) Less(i, j int) bool { return a[i].Price < a[j].Price }

type Book struct {
	ID          string
	Name        string
	Bid         BookLevelList
	Ask         BookLevelList
	Trades      []*Trade
	Sequence    uint64
	Synced      bool
	ProductInfo product_info.Info
	Stats       *BookMapStats
}

func New(name string) *Book {
	return &Book{
		ID:     name,
		Name:   name,
		Bid:    []*BookLevel{},
		Ask:    []*BookLevel{},
		Trades: []*Trade{},
	}
}

func NewProductBook(id string) *Book {
	b := New(id)
	b.InitProductInfo()
	return b
}

func (b *Book) InitProductInfo() {
	b.ProductInfo = FetchProductInfo(b.ID)
}

func (b *Book) UpdateBidLevel(t time.Time, price, quantity float64) {
	var found bool

	for i, current := range b.Bid {
		if current.Price == price {
			if quantity == 0 {
				// remove
				b.Bid[i] = b.Bid[len(b.Bid)-1]
				b.Bid[len(b.Bid)-1] = nil
				b.Bid = b.Bid[:len(b.Bid)-1]
			} else {
				// update
				b.Bid[i].Quantity = quantity
			}
			found = true
			break
		}
	}

	if !found && quantity != 0 {
		// add
		b.Bid = append(b.Bid, &BookLevel{Price: price, Quantity: quantity})
	}
}

func (b *Book) UpdateAskLevel(t time.Time, price, quantity float64) {
	var found bool

	for i, current := range b.Ask {
		if current.Price == price {
			if quantity == 0 {
				// remove
				b.Ask[i] = b.Ask[len(b.Ask)-1]
				b.Ask[len(b.Ask)-1] = nil
				b.Ask = b.Ask[:len(b.Ask)-1]
			} else {
				// update
				b.Ask[i].Quantity = quantity
			}
			found = true
			break
		}
	}

	if !found && quantity != 0 {
		// add
		b.Ask = append(b.Ask, &BookLevel{Price: price, Quantity: quantity})
	}
}

func (b *Book) Sort() {
	sort.Sort(b.Bid)
	sort.Sort(b.Ask)
}

func (b *Book) AddTrade(t time.Time, price, quantity float64) {
	if len(b.Trades) >= 50 {
		// remove and free first item
		copy(b.Trades[0:], b.Trades[1:])
		b.Trades[len(b.Trades)-1] = nil
		b.Trades = b.Trades[:len(b.Trades)-1]
	}
	b.Trades = append(b.Trades, &Trade{Price: price, Quantity: quantity, Time: t})

	if b.Stats != nil {
		var found bool
		for _, state := range b.Stats.Bid {
			if state.Price == price {
				state.TradeSize += quantity
				found = true
				break
			}
		}
		if !found {
			for _, state := range b.Stats.Ask {
				if state.Price == price {
					state.TradeSize += quantity
					break
				}
			}
		}
	}
}

func (b *Book) LastPrice() float64 {
	var lastPrice float64
	i := len(b.Trades)
	if i > 0 {
		lastPrice = b.Trades[i-1].Price
	} else {
		lastPrice = b.CenterPrice()
	}
	return lastPrice
}

func (b *Book) Empty() bool {
	return len(b.Ask) == 0 && len(b.Ask) == 0
}

func (b *Book) CenterPrice() float64 {
	if b.Empty() {
		return 0.0
	}
	spread := b.Spread()
	return b.Ask[0].Price + (spread / 2)
}

func (b *Book) Spread() float64 {
	var spread float64
	if len(b.Bid) > 0 && len(b.Ask) > 0 {
		spread = b.Ask[0].Price - b.Bid[len(b.Bid)-1].Price
	}
	return spread
}

func (b *Book) Clear() {
	b.Bid = []*BookLevel{}
	b.Ask = []*BookLevel{}
}

func (b *Book) StateAsStats() *BookMapStatsCopy {
	b.Sort()

	stats := &BookMapStatsCopy{
		Bid: make([]OrderState, 0, len(b.Bid)),
		Ask: make([]OrderState, 0, len(b.Ask)),
	}

	for i, _ := range b.Bid {
		//for i := len(b.Bid) - 1; i >= 0; i-- {
		level := b.Bid[i]
		if level == nil { // fix: duo to unsafe access in opengl/orderbook/book.go
			continue
		}
		bid := OrderState{Price: level.Price, Size: level.Quantity, OrderCount: 1}
		stats.Bid = append(stats.Bid, bid)
	}

	for _, level := range b.Ask {
		if level == nil { // fix: duo to unsafe access in opengl/orderbook/book.go
			continue
		}
		ask := OrderState{Price: level.Price, Size: level.Quantity, OrderCount: 1}
		stats.Ask = append(stats.Ask, ask)
	}

	return stats
}

func (b *Book) ResetStats() {
	b.Stats = nil
	b.Stats = &BookMapStats{
		Bid: OrderStateList{},
		Ask: OrderStateList{},
	}

	for _, level := range b.Bid {
		bid := &OrderState{Price: level.Price, Size: level.Quantity}
		b.Stats.Bid = append(b.Stats.Bid, bid)
	}

	for _, level := range b.Ask {
		ask := &OrderState{Price: level.Price, Size: level.Quantity}
		b.Stats.Ask = append(b.Stats.Ask, ask)
	}
}

// TODO: improve. prolly memory/gc hungy
func (b *Book) StatsCopy() *BookMapStatsCopy {
	b.Stats.Sort()

	s := &BookMapStatsCopy{
		Bid: make([]OrderState, 0, len(b.Stats.Bid)),
		Ask: make([]OrderState, 0, len(b.Stats.Ask)),
	}

	for _, state := range b.Stats.Bid {
		s.Bid = append(s.Bid, *state)
	}

	for _, state := range b.Stats.Ask {
		s.Ask = append(s.Ask, *state)
	}

	return s
}

//func (b *Book) ResetStats() {
//}

type OrderState struct {
	Price      float64
	Size       float64
	OrderCount int
	TradeSize  float64
}

type OrderStateList []*OrderState

func (a OrderStateList) Len() int           { return len(a) }
func (a OrderStateList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a OrderStateList) Less(i, j int) bool { return a[i].Price < a[j].Price }

type BookMapStats struct {
	Bid OrderStateList
	Ask OrderStateList
}

type BookMapStatsCopy struct {
	Bid []OrderState
	Ask []OrderState
}

func (stats *BookMapStats) Sort() {
	sort.Sort(stats.Bid)
	sort.Sort(stats.Ask)
}
