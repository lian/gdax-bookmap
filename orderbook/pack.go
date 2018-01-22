package orderbook

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"time"
)

const (
	SyncPacket  uint8 = iota
	DiffPacket  uint8 = iota
	TradePacket uint8 = iota
)

func UnpackTimeKey(key []byte) time.Time {
	i, _ := strconv.ParseInt(string(key), 10, 64)
	t := time.Unix(0, i)
	return t
}

func PackTimeKey(t time.Time) []byte {
	return []byte(fmt.Sprintf("%d", t.UnixNano()))
}

func PackUnixNanoKey(nano int64) []byte {
	return []byte(fmt.Sprintf("%d", nano))
}

func UnpackSequence(data []byte) uint64 {
	buf := bytes.NewBuffer(data)
	var packetType uint8
	binary.Read(buf, binary.LittleEndian, &packetType)
	var sequence uint64
	binary.Read(buf, binary.LittleEndian, &sequence)

	return sequence
}

func UnpackPacket(data []byte) map[string]interface{} {
	buf := bytes.NewBuffer(data)

	var packetType uint8
	binary.Read(buf, binary.LittleEndian, &packetType)

	switch packetType {
	case DiffPacket:
		var sequence uint64
		binary.Read(buf, binary.LittleEndian, &sequence)
		var first uint64
		binary.Read(buf, binary.LittleEndian, &first)
		var last uint64
		binary.Read(buf, binary.LittleEndian, &last)

		var bidsCount uint64
		binary.Read(buf, binary.LittleEndian, &bidsCount)
		bids := make([][]float64, 0, bidsCount)
		for i := uint64(0); i < bidsCount; i += 1 {
			var price float64
			binary.Read(buf, binary.LittleEndian, &price)
			var size float64
			binary.Read(buf, binary.LittleEndian, &size)
			bids = append(bids, []float64{price, size})
		}

		var asksCount uint64
		binary.Read(buf, binary.LittleEndian, &asksCount)
		asks := make([][]float64, 0, asksCount)
		for i := uint64(0); i < asksCount; i += 1 {
			var price float64
			binary.Read(buf, binary.LittleEndian, &price)
			var size float64
			binary.Read(buf, binary.LittleEndian, &size)
			asks = append(asks, []float64{price, size})
		}

		return map[string]interface{}{
			"type":     "diff",
			"sequence": sequence,
			"first":    first,
			"last":     last,
			"bids":     bids,
			"asks":     asks,
		}
	case SyncPacket:
		var sequence uint64
		binary.Read(buf, binary.LittleEndian, &sequence)

		var bidsCount uint64
		binary.Read(buf, binary.LittleEndian, &bidsCount)
		bids := make([][]float64, 0, bidsCount)
		for i := uint64(0); i < bidsCount; i += 1 {
			var price float64
			binary.Read(buf, binary.LittleEndian, &price)
			var size float64
			binary.Read(buf, binary.LittleEndian, &size)
			bids = append(bids, []float64{price, size})
		}

		var asksCount uint64
		binary.Read(buf, binary.LittleEndian, &asksCount)
		asks := make([][]float64, 0, asksCount)
		for i := uint64(0); i < asksCount; i += 1 {
			var price float64
			binary.Read(buf, binary.LittleEndian, &price)
			var size float64
			binary.Read(buf, binary.LittleEndian, &size)
			asks = append(asks, []float64{price, size})
		}

		return map[string]interface{}{
			"type":     "sync",
			"sequence": sequence,
			"bids":     bids,
			"asks":     asks,
		}

	case TradePacket:
		var sequence uint64
		binary.Read(buf, binary.LittleEndian, &sequence)
		var side uint8
		binary.Read(buf, binary.LittleEndian, &side)
		var price float64
		binary.Read(buf, binary.LittleEndian, &price)
		var size float64
		binary.Read(buf, binary.LittleEndian, &size)
		return map[string]interface{}{
			"type":     "trade",
			"sequence": sequence,
			"side":     side,
			"price":    price,
			"size":     size,
		}
	}

	return map[string]interface{}{}
}
