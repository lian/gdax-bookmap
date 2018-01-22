package orderbook

import (
	"fmt"
	"time"
)

type DbBook struct {
	Book *Book
}

func NewDbBook(id string) *DbBook {
	b := &DbBook{
		Book: New(id),
	}
	return b
}

func (b *DbBook) UpdateSync(book *Book, first, last uint64) error {
	seq := book.Sequence
	next := seq + 1

	if first <= seq {
		return fmt.Errorf("Ignore old messages %d %d", last, seq)
	}

	//fmt.Println("UpdateSync", book.Synced, seq, first, last)

	if book.Synced {
		if first != next {
			fmt.Println("Message lost, wating for resync")
			book.Synced = false
		}
	} else {
		if (first <= next) && (last >= next) {
			book.Synced = true
		}
	}

	book.Sequence = last
	return nil
}

func (b *DbBook) applyLevels(data map[string]interface{}) {
	book := b.Book
	t := time.Now() // fix

	if bids, ok := data["bids"].([][]float64); ok {
		for i := len(bids) - 1; i >= 0; i-- {
			price := bids[i][0]
			quantity := bids[i][1]
			book.UpdateBidLevel(t, price, quantity)
		}
	}

	if asks, ok := data["asks"].([][]float64); ok {
		for i := len(asks) - 1; i >= 0; i-- {
			price := asks[i][0]
			quantity := asks[i][1]
			book.UpdateAskLevel(t, price, quantity)
		}
	}
}

func (b *DbBook) Process(t time.Time, data map[string]interface{}) bool {
	book := b.Book
	sequence := data["sequence"].(uint64)

	switch data["type"].(string) {
	case "diff":
		if err := b.UpdateSync(book, data["first"].(uint64), data["last"].(uint64)); err != nil {
			fmt.Println("DbBook.UpdateSync", book.ID, err)
			return false
		}

		b.applyLevels(data)
		book.Sort()

	case "sync":
		book.Clear()
		book.Sequence = sequence

		b.applyLevels(data)
		book.Sort()

	case "trade":
		book.AddTrade(t, data["side"].(uint8), data["price"].(float64), data["size"].(float64))

	default:
		fmt.Println("unkown DbBook.Process pkt")
		return false
	}

	return true
}
