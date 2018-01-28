package websocket

import (
	"bytes"
	"encoding/binary"

	"github.com/lian/gdax-bookmap/exchanges/gdax/orderbook"
	db_orderbook "github.com/lian/gdax-bookmap/orderbook"
)

func PackDiff(first, last uint64, diff *orderbook.BookLevelDiff) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, db_orderbook.DiffPacket)
	binary.Write(buf, binary.LittleEndian, uint64(first)) // sequence
	binary.Write(buf, binary.LittleEndian, uint64(first)) // first
	binary.Write(buf, binary.LittleEndian, uint64(last))  // last

	binary.Write(buf, binary.LittleEndian, uint64(len(diff.Bid)))
	for _, state := range diff.Bid {
		binary.Write(buf, binary.LittleEndian, state.Price) // price
		binary.Write(buf, binary.LittleEndian, state.Size)  // size
	}

	binary.Write(buf, binary.LittleEndian, uint64(len(diff.Ask)))
	for _, state := range diff.Ask {
		binary.Write(buf, binary.LittleEndian, state.Price) // price
		binary.Write(buf, binary.LittleEndian, state.Size)  // size
	}

	return buf.Bytes()
}

func PackSync(book *orderbook.Book) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, db_orderbook.SyncPacket)
	binary.Write(buf, binary.LittleEndian, uint64(book.Sequence))

	binary.Write(buf, binary.LittleEndian, uint64(len(book.Bid)))
	for _, level := range book.Bid {
		binary.Write(buf, binary.LittleEndian, level.Price) // price
		var size float64
		for _, order := range level.Orders {
			size += order.Size
		}
		binary.Write(buf, binary.LittleEndian, size) // size
	}

	binary.Write(buf, binary.LittleEndian, uint64(len(book.Ask)))
	for _, level := range book.Ask {
		binary.Write(buf, binary.LittleEndian, level.Price) // price
		var size float64
		for _, order := range level.Orders {
			size += order.Size
		}
		binary.Write(buf, binary.LittleEndian, size) // size
	}

	return buf.Bytes()
}

func PackTrade(trade *orderbook.Order) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, db_orderbook.TradePacket)
	binary.Write(buf, binary.LittleEndian, uint64(0))         // seq
	binary.Write(buf, binary.LittleEndian, uint8(trade.Side)) // side
	binary.Write(buf, binary.LittleEndian, trade.Price)       // price
	binary.Write(buf, binary.LittleEndian, trade.Size)        // size
	return buf.Bytes()
}
