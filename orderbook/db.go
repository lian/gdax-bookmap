package orderbook

import (
	"fmt"
	"strconv"
)

func NewDbBook(id string) *DbBook {
	b := &DbBook{
		Book: New(id),
	}
	return b
}

type DbBook struct {
	Book *Book
}

func (b *DbBook) Process(data map[string]interface{}) bool {
	var sequence uint64

	if val, ok := data["sequence"].(float64); ok {
		sequence = uint64(val)
	} else {
		sequence = data["sequence"].(uint64)
	}

	if sequence <= b.Book.Sequence {
		fmt.Println("Process: Ignore old messages", sequence, b.Book.Sequence, data["type"])
		return true
	}

	if (b.Book.Sequence != 0) && (sequence != (b.Book.Sequence + 1)) {
		fmt.Println("Process: Message lost, needs to resync", sequence, b.Book.Sequence)
		return false
	}

	b.Book.Sequence = sequence

	b.HandleMessage(data)

	return true
}

func (b *DbBook) HandleMessage(data map[string]interface{}) {
	switch data["type"].(string) {
	case "open":
		b.Book.Add(data)
		break
	case "done":
		b.Book.Remove(data["id"].(string))
		break
	case "match":
		b.Book.Match(data, false)
		break
	case "change":
		b.Book.Match(data, true)
		break
	case "sync":
		if seq, ok := data["sequence"]; ok {
			b.Book.Clear()
			b.Book.Sequence = uint64(seq.(float64))
			b.Book.SkipStatsUpdate = true
			//b.Book.Sequence = uint64(seq.(float64)) + 1 // TODO: fix

			if bids, ok := data["bids"].([]interface{}); ok {
				for i := len(bids) - 1; i >= 0; i-- {
					data := bids[i].([]interface{})
					price, _ := strconv.ParseFloat(data[0].(string), 64)
					size, _ := strconv.ParseFloat(data[1].(string), 64)
					b.Book.Add(map[string]interface{}{
						"id":    data[2].(string),
						"side":  "buy",
						"price": price,
						"size":  size,
					})
				}
			}

			if asks, ok := data["asks"].([]interface{}); ok {
				for i := len(asks) - 1; i >= 0; i-- {
					data := asks[i].([]interface{})
					price, _ := strconv.ParseFloat(data[0].(string), 64)
					size, _ := strconv.ParseFloat(data[1].(string), 64)
					b.Book.Add(map[string]interface{}{
						"id":    data[2].(string),
						"side":  "sell",
						"price": price,
						"size":  size,
					})
				}
			}
			b.Book.SkipStatsUpdate = false
			//b.Book.ResetStats() // improve
		}
		break
	}
}
