package orderbook

import (
	"fmt"
	"time"

	"github.com/lian/gdax-bookmap/orderbook/product_info"
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

type LevelDiff struct {
	Price float64
	Size  float64
}

func (bl *BookLevel) Add(order *Order) {
	bl.Orders = append(bl.Orders, order)
}

func (bl *BookLevel) Remove(order *Order) {
	if len(bl.Orders) == 1 {
		bl.Orders = []*Order{}
		return
	}

	var i int
	var found bool

	for n, current := range bl.Orders {
		if current == order {
			i = n
			found = true
			break
		}
	}

	if found {
		copy(bl.Orders[i:], bl.Orders[i+1:])
		bl.Orders[len(bl.Orders)-1] = nil
		bl.Orders = bl.Orders[:len(bl.Orders)-1]
	}
}

func (bl *BookLevel) Size() float64 {
	var size float64
	for _, o := range bl.Orders {
		size += o.Size
	}
	return size
}

func (bl *BookLevel) Empty() bool {
	return len(bl.Orders) == 0
}

type BookLevelDiff struct {
	Bid []*LevelDiff
	Ask []*LevelDiff
}

type Book struct {
	ID          string
	ProductInfo product_info.Info
	Bid         map[float64]*BookLevel
	Ask         map[float64]*BookLevel
	OrderMap    map[string]*Order
	Sequence    uint64
	Trades      []*Order
	Diff        *BookLevelDiff
}

func New(id string) *Book {
	b := &Book{
		ID:          id,
		ProductInfo: FetchProductInfo(id),
		Bid:         map[float64]*BookLevel{},
		Ask:         map[float64]*BookLevel{},
		OrderMap:    map[string]*Order{},
		Trades:      []*Order{},
	}
	b.ResetDiff()
	return b
}

func (b *Book) ResetDiff() {
	b.Diff = nil
	b.Diff = &BookLevelDiff{
		Bid: []*LevelDiff{},
		Ask: []*LevelDiff{},
	}
}

func (b *Book) Clear() {
	b.Bid = map[float64]*BookLevel{}
	b.Ask = map[float64]*BookLevel{}
	b.OrderMap = map[string]*Order{}
	b.ResetDiff()
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
		fmt.Println("BOOK ERROR: ignore duplicated order_id", order)
		return
	}

	b.OrderMap[order.ID] = order

	level, found := b.FindLevel(order)
	if !found {
		level = b.AddLevel(order)
	}
	level.Add(order)

	// update diff stats
	size := level.Size()
	if order.Side == BidSide {
		var found bool
		for _, state := range b.Diff.Bid {
			if state.Price == order.Price {
				state.Size = size
				found = true
				break
			}
		}
		if !found {
			b.Diff.Bid = append(b.Diff.Bid, &LevelDiff{Price: order.Price, Size: size})
		}
	} else {
		var found bool
		for _, state := range b.Diff.Ask {
			if state.Price == order.Price {
				state.Size = size
				found = true
				break
			}
		}
		if !found {
			b.Diff.Ask = append(b.Diff.Ask, &LevelDiff{Price: order.Price, Size: size})
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

	// update diff stats
	size := level.Size()
	if order.Side == BidSide {
		var found bool
		for _, state := range b.Diff.Bid {
			if state.Price == order.Price {
				state.Size = size
				found = true
				break
			}
		}
		if !found {
			b.Diff.Bid = append(b.Diff.Bid, &LevelDiff{Price: order.Price, Size: size})
		}
	} else {
		var found bool
		for _, state := range b.Diff.Ask {
			if state.Price == order.Price {
				state.Size = size
				found = true
				break
			}
		}
		if !found {
			b.Diff.Ask = append(b.Diff.Ask, &LevelDiff{Price: order.Price, Size: size})
		}
	}

	if level.Empty() {
		b.RemoveLevel(order.Side, level)
	}

}

func (b *Book) AddLevel(order *Order) *BookLevel {
	level := &BookLevel{Price: order.Price, Orders: []*Order{}}
	if order.Side == BidSide {
		b.Bid[order.Price] = level
	} else {
		b.Ask[order.Price] = level
	}
	return level
}

func (b *Book) RemoveLevel(side Side, level *BookLevel) {
	if side == BidSide {
		if _, ok := b.Bid[level.Price]; ok {
			delete(b.Bid, level.Price)
		}
	} else {
		if _, ok := b.Ask[level.Price]; ok {
			delete(b.Ask, level.Price)
		}
	}
}

func (b *Book) FindLevel(order *Order) (*BookLevel, bool) {
	if order.Side == BidSide {
		if level, ok := b.Bid[order.Price]; ok {
			return level, true
		}
	} else {
		if level, ok := b.Ask[order.Price]; ok {
			return level, true
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

	// update diff stats if change
	size := level.Size()
	if change {
		if match.Side == BidSide {
			var found bool
			for _, state := range b.Diff.Bid {
				if state.Price == order.Price {
					state.Size = size
					found = true
					break
				}
			}
			if !found {
				b.Diff.Bid = append(b.Diff.Bid, &LevelDiff{Price: order.Price, Size: size})
			}
		} else {
			var found bool
			for _, state := range b.Diff.Ask {
				if state.Price == order.Price {
					state.Size = size
					found = true
					break
				}
			}
			if !found {
				b.Diff.Ask = append(b.Diff.Ask, &LevelDiff{Price: order.Price, Size: size})
			}
		}
	}

	if !change {
		b.AddTrade(match)
	}
}

func (b *Book) AddTrade(match *Order) {
	if len(b.Trades) >= 50 {
		copy(b.Trades[0:], b.Trades[1:])
		b.Trades[len(b.Trades)-1] = nil
		b.Trades = b.Trades[:len(b.Trades)-1]
	}
	b.Trades = append(b.Trades, match)
}
