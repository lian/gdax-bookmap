package orderbook

import (
	"math"
	"time"

	"github.com/lian/gdax-bookmap/orderbook/product_info"
)

type LevelDiff struct {
	Price float64
	Size  float64
}

type BookLevelDiff struct {
	Bid []*LevelDiff
	Ask []*LevelDiff
}

type Side uint8

const BidSide Side = 0
const AskSide Side = 1

type BookLevel struct {
	Price float64
	Size  float64
}

type Trade struct {
	Price float64
	Size  float64
	Time  time.Time
	Side  Side
}

type Book struct {
	ID          string
	ProductInfo product_info.Info
	Bid         []*BookLevel
	Ask         []*BookLevel
	Trades      []*Trade
	Sequence    uint64
	Synced      bool
	Diff        *BookLevelDiff
}

func New(id string) *Book {
	book := &Book{
		ID:          id,
		ProductInfo: FetchProductInfo(id),
		Bid:         []*BookLevel{},
		Ask:         []*BookLevel{},
		Trades:      []*Trade{},
	}
	book.ResetDiff()
	return book
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

func (b *Book) UpdateBidLevel(t time.Time, price, size float64) {
	var found bool

	for i, current := range b.Bid {
		if current.Price == price {
			if size == 0 {
				// remove
				b.Bid[i] = b.Bid[len(b.Bid)-1]
				b.Bid[len(b.Bid)-1] = nil
				b.Bid = b.Bid[:len(b.Bid)-1]
			} else {
				// update
				b.Bid[i].Size = size
			}
			found = true
			break
		}
	}

	if !found && size != 0 {
		// add
		b.Bid = append(b.Bid, &BookLevel{Price: price, Size: size})
	}

	// update diff stats
	found = false
	for _, state := range b.Diff.Bid {
		if state.Price == price {
			state.Size = size
			found = true
			break
		}
	}
	if !found {
		b.Diff.Bid = append(b.Diff.Bid, &LevelDiff{Price: price, Size: size})
	}
}

func (b *Book) UpdateAskLevel(t time.Time, price, size float64) {
	var found bool

	for i, current := range b.Ask {
		if current.Price == price {
			if size == 0 {
				// remove
				b.Ask[i] = b.Ask[len(b.Ask)-1]
				b.Ask[len(b.Ask)-1] = nil
				b.Ask = b.Ask[:len(b.Ask)-1]
			} else {
				// update
				b.Ask[i].Size = size
			}
			found = true
			break
		}
	}

	if !found && size != 0 {
		// add
		b.Ask = append(b.Ask, &BookLevel{Price: price, Size: size})
	}

	// update diff stats
	found = false
	for _, state := range b.Diff.Ask {
		if state.Price == price {
			state.Size = size
			found = true
			break
		}
	}
	if !found {
		b.Diff.Ask = append(b.Diff.Ask, &LevelDiff{Price: price, Size: size})
	}
}

// :.(
func (b *Book) FixBookLevels() {
	now := time.Now()

	lowestAsk := math.MaxFloat64
	for _, level := range b.Ask {
		if level.Price < lowestAsk {
			lowestAsk = level.Price
		}
	}

	highestBid := 0.0
	for _, level := range b.Bid {
		if level.Price > highestBid {
			highestBid = level.Price
		}
	}

	deleteBids := []float64{}
	for _, level := range b.Bid {
		if level.Price > lowestAsk {
			deleteBids = append(deleteBids, level.Price)
		}
	}

	for _, price := range deleteBids {
		b.UpdateBidLevel(now, price, 0)
	}

	deleteAsks := []float64{}
	for _, level := range b.Ask {
		if level.Price < highestBid {
			deleteAsks = append(deleteAsks, level.Price)
		}
	}

	for _, price := range deleteAsks {
		b.UpdateAskLevel(now, price, 0)
	}

	//fmt.Println("FixBookLevels", b.ID, "took", time.Since(now))
}

func (b *Book) AddTrade(t time.Time, side uint8, price, size float64) {
	if len(b.Trades) >= 50 {
		// remove and free first item
		copy(b.Trades[0:], b.Trades[1:])
		b.Trades[len(b.Trades)-1] = nil
		b.Trades = b.Trades[:len(b.Trades)-1]
	}
	b.Trades = append(b.Trades, &Trade{Side: Side(side), Price: price, Size: size, Time: t})
}

func (b *Book) Clear() {
	b.Bid = []*BookLevel{}
	b.Ask = []*BookLevel{}
	b.ResetDiff()
}

func (b *Book) ResetDiff() {
	b.Diff = nil
	b.Diff = &BookLevelDiff{
		Bid: []*LevelDiff{},
		Ask: []*LevelDiff{},
	}
}
