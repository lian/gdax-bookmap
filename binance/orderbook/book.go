package orderbook

import (
	"time"

	"github.com/lian/gdax-bookmap/orderbook/product_info"
)

type Side uint8

const BidSide Side = 0
const AskSide Side = 1

type BookLevel struct {
	Price    float64
	Quantity float64
}

type Trade struct {
	Price    float64
	Quantity float64
	Time     time.Time
	Side     Side
}

type Book struct {
	ID          string
	ProductInfo product_info.Info
	Bid         []*BookLevel
	Ask         []*BookLevel
	Trades      []*Trade
	Sequence    uint64
	Synced      bool
}

func New(id string) *Book {
	return &Book{
		ID:          id,
		ProductInfo: FetchProductInfo(id),
		Bid:         []*BookLevel{},
		Ask:         []*BookLevel{},
		Trades:      []*Trade{},
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

func (b *Book) AddTrade(t time.Time, side uint8, price, quantity float64) {
	if len(b.Trades) >= 50 {
		// remove and free first item
		copy(b.Trades[0:], b.Trades[1:])
		b.Trades[len(b.Trades)-1] = nil
		b.Trades = b.Trades[:len(b.Trades)-1]
	}
	b.Trades = append(b.Trades, &Trade{Side: Side(side), Price: price, Quantity: quantity, Time: t})
}

func (b *Book) Clear() {
	b.Bid = []*BookLevel{}
	b.Ask = []*BookLevel{}
}
