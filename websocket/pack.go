package websocket

import (
	"bytes"
	"encoding/binary"
	"strconv"

	uuid "github.com/satori/go.uuid"
)

const (
	BuySide  uint8 = iota
	SellSide uint8 = iota
)

const (
	OpenPacket   uint8 = iota
	DonePacket   uint8 = iota
	MatchPacket  uint8 = iota
	ChangePacket uint8 = iota
	SyncPacket   uint8 = iota
	IgnorePacket uint8 = iota
)

func PackPacket(data map[string]interface{}) []byte {
	buf := new(bytes.Buffer)

	switch data["type"].(string) {
	case "ignore":
		binary.Write(buf, binary.LittleEndian, IgnorePacket)
		binary.Write(buf, binary.LittleEndian, uint64(data["sequence"].(float64)))

	case "open":
		binary.Write(buf, binary.LittleEndian, OpenPacket)
		binary.Write(buf, binary.LittleEndian, uint64(data["sequence"].(float64)))

		if data["side"].(string) == "buy" {
			binary.Write(buf, binary.LittleEndian, BuySide)
		} else {
			binary.Write(buf, binary.LittleEndian, SellSide)
		}

		id, _ := uuid.FromString(data["order_id"].(string))
		binary.Write(buf, binary.LittleEndian, id)

		price, _ := strconv.ParseFloat(data["price"].(string), 64)
		binary.Write(buf, binary.LittleEndian, price)

		size, _ := strconv.ParseFloat(data["remaining_size"].(string), 64)
		binary.Write(buf, binary.LittleEndian, size)

		/*
			t, _ := time.Parse(TimeFormat, data["time"].(string))
			tb, _ := t.MarshalBinary()
			binary.Write(buf, binary.LittleEndian, tb)
		*/
	case "done":
		binary.Write(buf, binary.LittleEndian, DonePacket)
		binary.Write(buf, binary.LittleEndian, uint64(data["sequence"].(float64)))
		id, _ := uuid.FromString(data["order_id"].(string))
		binary.Write(buf, binary.LittleEndian, id)
		/*
			t, _ := time.Parse(TimeFormat, data["time"].(string))
			tb, _ := t.MarshalBinary()
			binary.Write(buf, binary.LittleEndian, tb)
		*/
	case "match":
		binary.Write(buf, binary.LittleEndian, MatchPacket)
		binary.Write(buf, binary.LittleEndian, uint64(data["sequence"].(float64)))

		if data["side"].(string) == "buy" {
			binary.Write(buf, binary.LittleEndian, BuySide)
		} else {
			binary.Write(buf, binary.LittleEndian, SellSide)
		}

		maker_id, _ := uuid.FromString(data["maker_order_id"].(string))
		binary.Write(buf, binary.LittleEndian, maker_id)

		taker_id, _ := uuid.FromString(data["taker_order_id"].(string))
		binary.Write(buf, binary.LittleEndian, taker_id)

		price, _ := strconv.ParseFloat(data["price"].(string), 64)
		binary.Write(buf, binary.LittleEndian, price)

		size, _ := strconv.ParseFloat(data["size"].(string), 64)
		binary.Write(buf, binary.LittleEndian, size)

		/*
			t, _ := time.Parse(TimeFormat, data["time"].(string))
			tb, _ := t.MarshalBinary()
			binary.Write(buf, binary.LittleEndian, tb)
		*/
	case "change":
		binary.Write(buf, binary.LittleEndian, ChangePacket)
		binary.Write(buf, binary.LittleEndian, uint64(data["sequence"].(float64)))

		if data["side"].(string) == "buy" {
			binary.Write(buf, binary.LittleEndian, BuySide)
		} else {
			binary.Write(buf, binary.LittleEndian, SellSide)
		}

		id, _ := uuid.FromString(data["order_id"].(string))
		binary.Write(buf, binary.LittleEndian, id)

		price, _ := strconv.ParseFloat(data["price"].(string), 64)
		binary.Write(buf, binary.LittleEndian, price)

		old_size, _ := strconv.ParseFloat(data["old_size"].(string), 64)
		new_size, _ := strconv.ParseFloat(data["new_size"].(string), 64)
		size := old_size - new_size
		binary.Write(buf, binary.LittleEndian, size)

		/*
			t, _ := time.Parse(TimeFormat, data["time"].(string))
			tb, _ := t.MarshalBinary()
			binary.Write(buf, binary.LittleEndian, tb)
		*/
	case "sync":
		binary.Write(buf, binary.LittleEndian, SyncPacket)
		binary.Write(buf, binary.LittleEndian, uint64(data["sequence"].(float64)))

		if bids, ok := data["bids"].([][]interface{}); ok {
			binary.Write(buf, binary.LittleEndian, uint64(len(bids)))
			for i := len(bids) - 1; i >= 0; i-- {
				d := bids[i]
				id, _ := uuid.FromString(d[2].(string))
				binary.Write(buf, binary.LittleEndian, id)             // id
				binary.Write(buf, binary.LittleEndian, d[0].(float64)) // price
				binary.Write(buf, binary.LittleEndian, d[1].(float64)) // size
			}
		}

		if asks, ok := data["asks"].([][]interface{}); ok {
			binary.Write(buf, binary.LittleEndian, uint64(len(asks)))
			for i := len(asks) - 1; i >= 0; i-- {
				d := asks[i]
				id, _ := uuid.FromString(d[2].(string))
				binary.Write(buf, binary.LittleEndian, id)             // id
				binary.Write(buf, binary.LittleEndian, d[0].(float64)) // price
				binary.Write(buf, binary.LittleEndian, d[1].(float64)) // size
			}
		}
	}

	//fmt.Printf("self %s unpack %#v\n", data["type"].(string), len(UnpackPacket(buf.Bytes())) != 0)

	return buf.Bytes()
}

//const TimeFormat = "2006-01-02T15:04:05.999999Z07:00"

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
	case IgnorePacket:
		var sequence uint64
		binary.Read(buf, binary.LittleEndian, &sequence)
		return map[string]interface{}{
			"type":     "ignore",
			"sequence": sequence,
		}

	case OpenPacket:
		var sequence uint64
		binary.Read(buf, binary.LittleEndian, &sequence)
		var side uint8
		binary.Read(buf, binary.LittleEndian, &side)
		sideString := "sell"
		if side == BuySide {
			sideString = "buy"
		}
		var id uuid.UUID
		binary.Read(buf, binary.LittleEndian, &id)
		var price float64
		binary.Read(buf, binary.LittleEndian, &price)
		var size float64
		binary.Read(buf, binary.LittleEndian, &size)
		/*
			var tb [15]byte
			var t time.Time
			binary.Read(buf, binary.LittleEndian, &tb)
			t.UnmarshalBinary(tb[:])
		*/

		return map[string]interface{}{
			"type":     "open",
			"sequence": sequence,
			"side":     sideString,
			"id":       id.String(),
			"price":    price,
			"size":     size,
			//"time":     t.Format(TimeFormat),
		}
	case DonePacket:
		var sequence uint64
		binary.Read(buf, binary.LittleEndian, &sequence)
		var id uuid.UUID
		binary.Read(buf, binary.LittleEndian, &id)
		/*
			var tb [15]byte
			var t time.Time
			binary.Read(buf, binary.LittleEndian, &tb)
			t.UnmarshalBinary(tb[:])
		*/

		return map[string]interface{}{
			"type":     "done",
			"sequence": sequence,
			"id":       id.String(),
			//"time":     t.Format(TimeFormat),
		}
	case MatchPacket:
		var sequence uint64
		binary.Read(buf, binary.LittleEndian, &sequence)
		var side uint8
		binary.Read(buf, binary.LittleEndian, &side)
		sideString := "sell"
		if side == BuySide {
			sideString = "buy"
		}
		var maker_id uuid.UUID
		binary.Read(buf, binary.LittleEndian, &maker_id)
		var taker_id uuid.UUID
		binary.Read(buf, binary.LittleEndian, &taker_id)
		var price float64
		binary.Read(buf, binary.LittleEndian, &price)
		var size float64
		binary.Read(buf, binary.LittleEndian, &size)
		/*
			var tb [15]byte
			var t time.Time
			binary.Read(buf, binary.LittleEndian, &tb)
			t.UnmarshalBinary(tb[:])
		*/

		return map[string]interface{}{
			"type":           "match",
			"sequence":       sequence,
			"side":           sideString,
			"maker_order_id": maker_id.String(),
			"taker_order_id": taker_id.String(),
			"price":          price,
			"size":           size,
			//"time":           t.Format(TimeFormat),
		}
	case ChangePacket:
		var sequence uint64
		binary.Read(buf, binary.LittleEndian, &sequence)
		var side uint8
		binary.Read(buf, binary.LittleEndian, &side)
		sideString := "sell"
		if side == BuySide {
			sideString = "buy"
		}
		var id uuid.UUID
		binary.Read(buf, binary.LittleEndian, &id)
		var price float64
		binary.Read(buf, binary.LittleEndian, &price)
		var size float64
		binary.Read(buf, binary.LittleEndian, &size)
		/*
			var tb [15]byte
			var t time.Time
			binary.Read(buf, binary.LittleEndian, &tb)
			t.UnmarshalBinary(tb[:])
		*/

		return map[string]interface{}{
			"type":           "change",
			"sequence":       sequence,
			"side":           sideString,
			"maker_order_id": id.String(),
			"price":          price,
			"size":           size,
			//"time":           t.Format(TimeFormat),
		}
	case SyncPacket:
		var sequence uint64
		binary.Read(buf, binary.LittleEndian, &sequence)

		var bidsCount uint64
		binary.Read(buf, binary.LittleEndian, &bidsCount)
		bids := make([][]interface{}, 0, bidsCount)
		for i := uint64(0); i < bidsCount; i += 1 {
			var id uuid.UUID
			binary.Read(buf, binary.LittleEndian, &id)
			var price float64
			binary.Read(buf, binary.LittleEndian, &price)
			var size float64
			binary.Read(buf, binary.LittleEndian, &size)
			bids = append(bids, []interface{}{price, size, id.String()})
		}

		var asksCount uint64
		binary.Read(buf, binary.LittleEndian, &asksCount)
		asks := make([][]interface{}, 0, asksCount)
		for i := uint64(0); i < asksCount; i += 1 {
			var id uuid.UUID
			binary.Read(buf, binary.LittleEndian, &id)
			var price float64
			binary.Read(buf, binary.LittleEndian, &price)
			var size float64
			binary.Read(buf, binary.LittleEndian, &size)
			asks = append(asks, []interface{}{price, size, id.String()})
		}

		return map[string]interface{}{
			"type":     "sync",
			"sequence": sequence,
			"bids":     bids,
			"asks":     asks,
		}
	}

	return map[string]interface{}{}
}
