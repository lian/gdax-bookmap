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
	return time.Unix(0, i)
}

func PackTimeKey(t time.Time) []byte {
	return []byte(fmt.Sprintf("%d", t.UnixNano()))
}

func PackUnixNanoKey(nano int64) []byte {
	return []byte(fmt.Sprintf("%d", nano))
}

func (book *Book) UpdateSync(first, last uint64) error {
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

func (book *Book) Process(t time.Time, data []byte) bool {
	buf := bytes.NewBuffer(data)

	var packetType uint8
	var sequence uint64
	var first uint64
	var last uint64
	var bidsCount uint64
	var asksCount uint64
	var price float64
	var size float64
	var side uint8

	binary.Read(buf, binary.LittleEndian, &packetType)

	switch packetType {
	case DiffPacket:
		binary.Read(buf, binary.LittleEndian, &sequence)
		binary.Read(buf, binary.LittleEndian, &first)
		binary.Read(buf, binary.LittleEndian, &last)

		if err := book.UpdateSync(first, last); err != nil {
			fmt.Println(book.ProductInfo.DatabaseKey, "UpdateSync Error", err)
			return false
		}

		binary.Read(buf, binary.LittleEndian, &bidsCount)
		for i := uint64(0); i < bidsCount; i += 1 {
			binary.Read(buf, binary.LittleEndian, &price)
			binary.Read(buf, binary.LittleEndian, &size)

			book.UpdateBidLevel(t, price, size)
		}

		binary.Read(buf, binary.LittleEndian, &asksCount)
		for i := uint64(0); i < asksCount; i += 1 {
			binary.Read(buf, binary.LittleEndian, &price)
			binary.Read(buf, binary.LittleEndian, &size)

			book.UpdateAskLevel(t, price, size)
		}

		book.Sort()

	case SyncPacket:
		binary.Read(buf, binary.LittleEndian, &sequence)

		book.Clear()
		book.Sequence = sequence

		binary.Read(buf, binary.LittleEndian, &bidsCount)
		for i := uint64(0); i < bidsCount; i += 1 {
			binary.Read(buf, binary.LittleEndian, &price)
			binary.Read(buf, binary.LittleEndian, &size)

			book.UpdateBidLevel(t, price, size)
		}

		binary.Read(buf, binary.LittleEndian, &asksCount)
		for i := uint64(0); i < asksCount; i += 1 {
			binary.Read(buf, binary.LittleEndian, &price)
			binary.Read(buf, binary.LittleEndian, &size)

			book.UpdateAskLevel(t, price, size)
		}

		book.Sort()

	case TradePacket:
		binary.Read(buf, binary.LittleEndian, &sequence)
		binary.Read(buf, binary.LittleEndian, &side)
		binary.Read(buf, binary.LittleEndian, &price)
		binary.Read(buf, binary.LittleEndian, &size)

		book.AddTrade(t, side, price, size)

	default:
		fmt.Println(book.ProductInfo.DatabaseKey, "unkown packetType", packetType)
		return false
	}

	return true
}
