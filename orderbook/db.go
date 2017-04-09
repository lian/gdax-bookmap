package orderbook

import "fmt"

func NewDbBook(id string) *DbBook {
	b := &DbBook{
		Book: New(id),
	}
	b.Book.AlwaysSort = false
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

	if data["type"].(string) != "sync" {
		if sequence <= b.Book.Sequence {
			fmt.Println("Process: Ignore old messages", sequence, b.Book.Sequence, data["type"])
			return true
		}

		if (b.Book.Sequence != 0) && (sequence != (b.Book.Sequence + 1)) {
			fmt.Println("Process: Message lost, needs to resync", sequence, b.Book.Sequence)
			return false
		}
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
			b.Book.Sequence = seq.(uint64)
			b.Book.SkipStatsUpdate = true

			if bids, ok := data["bids"].([][]interface{}); ok {
				for i := len(bids) - 1; i >= 0; i-- {
					data := bids[i]
					b.Book.Add(map[string]interface{}{
						"side":  "buy",
						"price": data[0].(float64),
						"size":  data[1].(float64),
						"id":    data[2].(string),
					})
				}
			}

			if asks, ok := data["asks"].([][]interface{}); ok {
				for i := len(asks) - 1; i >= 0; i-- {
					data := asks[i]
					b.Book.Add(map[string]interface{}{
						"side":  "sell",
						"price": data[0].(float64),
						"size":  data[1].(float64),
						"id":    data[2].(string),
					})
				}
			}
			b.Book.SkipStatsUpdate = false
		}
		break
	}
}
