package orderbook

import (
	"sort"
	"time"

	"github.com/lian/gdax-bookmap/orderbook/product_info"
)

type BookLevel struct {
	Price       float64
	Quantity    float64
	MaxQuantity float64
	OrderCount  int
	TradeSize   float64
}

type Side uint8

const BidSide Side = 0
const AskSide Side = 1

type Trade struct {
	Price    float64
	Quantity float64
	Time     time.Time
	Side     Side
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

func (b *Book) GetSide(price float64) uint8 {
	for _, level := range b.Bid {
		if level.Price == price {
			return uint8(BidSide)
		}
	}
	for _, level := range b.Ask {
		if level.Price == price {
			return uint8(AskSide)
		}
	}
	return uint8(BidSide)
}

func (b *Book) UpdateBidLevel(t time.Time, price, quantity float64) {
	var found bool

	for i, current := range b.Bid {
		if current.Price == price {
			if quantity == 0 {
				// remove
				b.Bid[i].Quantity = 0
			} else {
				// update
				b.Bid[i].Quantity = quantity
				if quantity > b.Bid[i].MaxQuantity {
					b.Bid[i].MaxQuantity = quantity
				}
				b.Bid[i].OrderCount += 1 // remove?
			}
			found = true
			break
		}
	}

	if !found && quantity != 0 {
		// add
		b.Bid = append(b.Bid, &BookLevel{Price: price, Quantity: quantity, MaxQuantity: quantity, OrderCount: 1})
	}
}

func (b *Book) UpdateAskLevel(t time.Time, price, quantity float64) {
	var found bool

	for i, current := range b.Ask {
		if current.Price == price {
			if quantity == 0 {
				// remove
				b.Ask[i].Quantity = 0
			} else {
				// update
				b.Ask[i].Quantity = quantity
				if quantity > b.Ask[i].MaxQuantity {
					b.Ask[i].MaxQuantity = quantity
				}
				b.Ask[i].OrderCount += 1 // remove?
			}
			found = true
			break
		}
	}

	if !found && quantity != 0 {
		// add
		b.Ask = append(b.Ask, &BookLevel{Price: price, Quantity: quantity, MaxQuantity: quantity, OrderCount: 1})
	}
}

func (b *Book) Sort() {
	sort.Sort(b.Bid)
	sort.Sort(b.Ask)
}

func (b *Book) AddTrade(t time.Time, side uint8, price, quantity float64) {
	if len(b.Trades) >= 50 {
		// remove and free first item
		copy(b.Trades[0:], b.Trades[1:])
		b.Trades[len(b.Trades)-1] = nil
		b.Trades = b.Trades[:len(b.Trades)-1]
	}
	trade := &Trade{Price: price, Side: Side(side), Quantity: quantity, Time: t}
	b.Trades = append(b.Trades, trade)

	if trade.Side == BidSide {
		for _, level := range b.Bid {
			if level.Price == price {
				level.TradeSize += quantity
				break
			}
		}
	} else {
		for _, level := range b.Ask {
			if level.Price == price {
				level.TradeSize += quantity
				break
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
	//b.Sort() // called by dbBook

	stats := &BookMapStatsCopy{
		Bid: make([]OrderState, 0, len(b.Bid)),
		Ask: make([]OrderState, 0, len(b.Ask)),
	}

	for _, level := range b.Bid {
		if level.Quantity == 0 {
			continue
		}
		bid := OrderState{Price: level.Price, Size: level.Quantity, OrderCount: level.OrderCount}
		stats.Bid = append(stats.Bid, bid)
	}

	for _, level := range b.Ask {
		if level.Quantity == 0 {
			continue
		}
		ask := OrderState{Price: level.Price, Size: level.Quantity, OrderCount: level.OrderCount}
		stats.Ask = append(stats.Ask, ask)
	}

	return stats
}

func (b *Book) ResetStats() {
	bid := make([]*BookLevel, 0, len(b.Bid))
	ask := make([]*BookLevel, 0, len(b.Ask))

	for _, level := range b.Bid {
		level.MaxQuantity = level.Quantity
		level.TradeSize = 0
		if level.Quantity != 0 {
			bid = append(bid, level)
		}
	}

	for _, level := range b.Ask {
		level.MaxQuantity = level.Quantity
		level.TradeSize = 0
		if level.Quantity != 0 {
			ask = append(ask, level)
		}
	}

	b.Bid = bid
	b.Ask = ask
}

func (b *Book) StatsCopy() *BookMapStatsCopy {
	stats := &BookMapStatsCopy{
		Bid: make([]OrderState, 0, len(b.Bid)),
		Ask: make([]OrderState, 0, len(b.Ask)),
	}

	for _, level := range b.Bid {
		bid := OrderState{Price: level.Price, Size: level.MaxQuantity, OrderCount: level.OrderCount, TradeSize: level.TradeSize}
		stats.Bid = append(stats.Bid, bid)
	}

	for _, level := range b.Ask {
		ask := OrderState{Price: level.Price, Size: level.MaxQuantity, OrderCount: level.OrderCount, TradeSize: level.TradeSize}
		stats.Ask = append(stats.Ask, ask)
	}

	return stats
}

type OrderState struct {
	Price      float64
	Size       float64
	OrderCount int
	TradeSize  float64
}

type BookMapStatsCopy struct {
	Bid []OrderState
	Ask []OrderState
}
